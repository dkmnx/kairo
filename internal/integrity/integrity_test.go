package integrity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

// testTransport routes requests by URL path suffix to canned responses.
type testTransport struct {
	artifactResp   []byte
	bundleResp     []byte
	checksumResp   []byte
	bundleStatus   int
	checksumStatus int
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path

	if strings.HasSuffix(path, ".sha256") {
		if t.checksumStatus != 0 {
			return &http.Response{
				StatusCode: t.checksumStatus,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		if t.checksumResp == nil {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(t.checksumResp)),
			Header:     make(http.Header),
		}, nil
	}

	if strings.HasSuffix(path, ".sigstore.json") {
		if t.bundleStatus != 0 {
			return &http.Response{
				StatusCode: t.bundleStatus,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		if t.bundleResp == nil {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(t.bundleResp)),
			Header:     make(http.Header),
		}, nil
	}

	if strings.HasSuffix(path, "catalog.json") || strings.Contains(path, "catalog") {
		if t.artifactResp == nil {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(t.artifactResp)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Header:     make(http.Header),
	}, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestFetchVerified_CosignSuccess(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)
	transport := &testTransport{
		artifactResp: artifact,
		bundleResp:   []byte(`{"payload":"sigstore-bundle"}`),
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "/usr/bin/cosign", nil
	}
	execCmd := func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "true")
	}

	data, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		execCmd,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err != nil {
		t.Fatalf("FetchVerified() error = %v", err)
	}
	if !bytes.Equal(data, artifact) {
		t.Errorf("FetchVerified() returned %s, want %s", data, artifact)
	}
}

func TestFetchVerified_SHA256Fallback(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)
	expectedHash := sha256Hex(artifact)

	transport := &testTransport{
		artifactResp: artifact,
		checksumResp: []byte(expectedHash),
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "", fmt.Errorf("cosign not found")
	}

	data, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err != nil {
		t.Fatalf("FetchVerified() SHA256 fallback error = %v", err)
	}
	if !bytes.Equal(data, artifact) {
		t.Errorf("FetchVerified() returned %s, want %s", data, artifact)
	}
}

func TestFetchVerified_SHA256Mismatch(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)

	transport := &testTransport{
		artifactResp: artifact,
		checksumResp: []byte("0000000000000000000000000000000000000000000000000000000000000000"),
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "", fmt.Errorf("cosign not found")
	}

	_, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err == nil {
		t.Fatal("FetchVerified() should return error on SHA256 mismatch")
	}
	if !strings.Contains(err.Error(), "SHA256 integrity check failed") {
		t.Errorf("error should mention SHA256 integrity check, got: %v", err)
	}
}

func TestFetchVerified_ChecksumDownloadFails(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)

	transport := &testTransport{
		artifactResp:   artifact,
		checksumStatus: http.StatusInternalServerError,
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "", fmt.Errorf("cosign not found")
	}

	_, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err == nil {
		t.Fatal("FetchVerified() should return error when checksum download fails")
	}
	if !strings.Contains(err.Error(), "failed to download checksum") {
		t.Errorf("error should mention checksum download, got: %v", err)
	}
}

func TestFetchVerified_BundleDownloadFails(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)

	transport := &testTransport{
		artifactResp: artifact,
		bundleStatus: http.StatusInternalServerError,
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "/usr/bin/cosign", nil
	}

	_, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err == nil {
		t.Fatal("FetchVerified() should return error when bundle download fails")
	}
	if !strings.Contains(err.Error(), "failed to download sigstore bundle") {
		t.Errorf("error should mention bundle download, got: %v", err)
	}
}

func TestFetchVerified_ArtifactDownloadFails(t *testing.T) {
	transport := &testTransport{
		artifactResp: nil,
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "/usr/bin/cosign", nil
	}

	_, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err == nil {
		t.Fatal("FetchVerified() should return error when artifact download fails")
	}
	if !strings.Contains(err.Error(), "failed to download artifact") {
		t.Errorf("error should mention artifact download, got: %v", err)
	}
}

func TestFetchVerified_InvalidChecksumFormat(t *testing.T) {
	artifact := []byte(`{"providers":{}}`)

	transport := &testTransport{
		artifactResp: artifact,
		checksumResp: []byte("not-a-valid-hash"),
	}
	httpClient := &http.Client{Transport: transport}

	lookPath := func(string) (string, error) {
		return "", fmt.Errorf("cosign not found")
	}

	_, err := FetchVerified(
		context.Background(),
		httpClient,
		lookPath,
		nil,
		"https://example.com/catalog.json",
		"https://example.com/catalog.json.sigstore.json",
		"https://example.com/catalog.json.sha256",
	)
	if err == nil {
		t.Fatal("FetchVerified() should return error for invalid checksum format")
	}
	if !strings.Contains(err.Error(), "does not contain a valid 64-character hex hash") {
		t.Errorf("error should mention invalid checksum format, got: %v", err)
	}
}

func TestHexHashPattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid lowercase", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", true},
		{"valid uppercase", "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2", true},
		{"too short", "abc123", false},
		{"too long", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2extra", false},
		{"invalid chars", "g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", false},
		{"with spaces", " a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hexHashPattern.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("hexHashPattern.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

func TestVerifyChecksum_Exported(t *testing.T) {
	data := []byte("test data")
	hash := sha256Hex(data)

	_, err := verifyChecksum(
		context.Background(),
		&http.Client{Transport: &testTransport{
			artifactResp: data,
			checksumResp: []byte(hash),
		}},
		"https://example.com/catalog.json.sha256",
		data,
	)
	if err != nil {
		t.Errorf("verifyChecksum() error = %v", err)
	}
}

func TestVerifyChecksum_EmptyExpectedHash(t *testing.T) {
	data := []byte("test data")

	_, err := verifyChecksum(
		context.Background(),
		&http.Client{Transport: &testTransport{
			artifactResp: data,
			checksumResp: []byte(""),
		}},
		"https://example.com/catalog.json.sha256",
		data,
	)
	if err == nil {
		t.Fatal("verifyChecksum() should return error for empty expected hash")
	}
	if !strings.Contains(err.Error(), "does not contain a valid 64-character hex hash") {
		t.Errorf("error should mention invalid checksum format, got: %v", err)
	}
}

func TestVerifyChecksum_HashMismatch(t *testing.T) {
	data := []byte("test data")

	_, err := verifyChecksum(
		context.Background(),
		&http.Client{Transport: &testTransport{
			artifactResp: data,
			checksumResp: []byte("0000000000000000000000000000000000000000000000000000000000000000"),
		}},
		"https://example.com/catalog.json.sha256",
		data,
	)
	if err == nil {
		t.Fatal("verifyChecksum() should return error on hash mismatch")
	}
	if !strings.Contains(err.Error(), "SHA256 integrity check failed") {
		t.Errorf("error should mention SHA256 integrity check, got: %v", err)
	}
}
