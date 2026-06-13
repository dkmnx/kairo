package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

// successTransport returns canned responses for the cosign bundle and checksums
// URLs. Unrecognized requests return 404.
type successTransport struct {
	nonce         int
	bundleResp    []byte
	checksumsResp []byte
}

// TestRunInstallScript_ShellSuccess covers the Unix sh path of RunInstallScript.
func TestRunInstallScript_ShellSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RunInstallScript(script); err != nil {
		t.Errorf("RunInstallScript() unexpected error: %v", err)
	}
}

func TestRunInstallScript_ShellFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "install.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RunInstallScript(script); err == nil {
		t.Error("RunInstallScript() should fail when script exits non-zero")
	}
}

func TestRunInstallScript_ChmodOnMissingDirFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}

	dir := t.TempDir()
	// Path inside a non-existent directory so chmod fails.
	bad := filepath.Join(dir, "missing", "install.sh")

	err := RunInstallScript(bad)
	if err == nil {
		t.Error("RunInstallScript() should error when chmod fails (missing dir)")
	}
}

func TestVerifyCosignBundle_CosignMissing(t *testing.T) {
	c := &Client{
		LookPathFunc: func(string) (string, error) {
			return "", os.ErrNotExist
		},
	}

	if err := c.VerifyCosignBundle(context.Background(), "v1.0.0"); err != nil {
		t.Errorf("VerifyCosignBundle() should return nil when cosign is not installed, got: %v", err)
	}
}

func TestParseChecksumLine_Malformed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
		wantHash string
		wantFile string
	}{
		{"empty", "", false, "", ""},
		{"comment", "# this is a comment", false, "", ""},
		{"single token", "abc123", false, "", ""},
		{"bad hash length", "abc install.sh", false, "", ""},
		{"bad hash chars", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz install.sh", false, "", ""},
		{"good", "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899 install.sh", true, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899", "install.sh"},
		{"uppercase hash normalized", "AABBCCDDEEFF00112233445566778899AABBCCDDEEFF00112233445566778899 install.sh", true, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899", "install.sh"},
		{"two-token star-style", "abc123*  install.sh", false, "", ""},
		{"trailing whitespace", "  aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899  install.sh  ", true, "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899", "install.sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, filename, ok := ParseChecksumLine(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParseChecksumLine(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if hash != tt.wantHash {
				t.Errorf("ParseChecksumLine(%q) hash = %q, want %q", tt.input, hash, tt.wantHash)
			}
			if filename != tt.wantFile {
				t.Errorf("ParseChecksumLine(%q) filename = %q, want %q", tt.input, filename, tt.wantFile)
			}
		})
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(script, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	bogus := strings.Repeat("0", 64)
	if err := VerifyChecksum(script, bogus); err == nil {
		t.Error("VerifyChecksum() should fail when hashes do not match")
	}
}

func TestScriptNameForChecksumsExtra(t *testing.T) {
	if got := ScriptNameForChecksums("linux"); got != "scripts/install.sh" {
		t.Errorf("ScriptNameForChecksums(linux) = %q, want %q", got, "scripts/install.sh")
	}
	if got := ScriptNameForChecksums("windows"); got != "scripts/install.ps1" {
		t.Errorf("ScriptNameForChecksums(windows) = %q, want %q", got, "scripts/install.ps1")
	}
}

func TestVerifyCosignBundle_LookPathPermissionDenied(t *testing.T) {
	c := &Client{
		LookPathFunc: func(string) (string, error) {
			return "", os.ErrPermission
		},
	}

	if err := c.VerifyCosignBundle(context.Background(), "v1.0.0"); err != nil {
		t.Errorf("VerifyCosignBundle() should silently skip when cosign cannot be located: %v", err)
	}
}

func (s *successTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.nonce++

	if strings.Contains(req.URL.Path, "checksums.txt.sigstore.json") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(s.bundleResp)),
			Header:     make(http.Header),
		}, nil
	}

	if strings.Contains(req.URL.Path, "checksums.txt") {
		// Return 404 when checksumsResp is unset (nil or empty).
		if len(s.checksumsResp) == 0 {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(s.checksumsResp)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Header:     make(http.Header),
	}, nil
}

// TestVerifyCosignBundle_VerifySuccess exercises the full cosign verification path:
// bundle download, checksums download, and cosign subprocess execution (exit 0).
func TestVerifyCosignBundle_VerifySuccess(t *testing.T) {
	c := &Client{
		HTTPClient: &http.Client{
			Transport: &successTransport{
				bundleResp:    []byte(`{"payload": "test-bundle"}`),
				checksumsResp: []byte("abc123  scripts/install.sh\n"),
			},
		},
		LookPathFunc: func(string) (string, error) { return "/usr/bin/cosign", nil },
		ExecCommand: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "true")
		},
	}

	if err := c.VerifyCosignBundle(context.Background(), "v1.0.0"); err != nil {
		t.Errorf("VerifyCosignBundle should succeed when cosign exits 0, got: %v", err)
	}
}

// TestVerifyCosignBundle_VerifyFails exercises the cosign verification failure path:
// cosign exits non-zero, VerifyCosignBundle must surface the error.
func TestVerifyCosignBundle_VerifyFails(t *testing.T) {
	c := &Client{
		HTTPClient: &http.Client{
			Transport: &successTransport{
				bundleResp:    []byte(`{"payload": "test-bundle"}`),
				checksumsResp: []byte("abc123  scripts/install.sh\n"),
			},
		},
		LookPathFunc: func(string) (string, error) { return "/usr/bin/cosign", nil },
		ExecCommand: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	err := c.VerifyCosignBundle(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("VerifyCosignBundle should return error when cosign exits non-zero")
	}
	if !strings.Contains(err.Error(), "cosign bundle verification failed") {
		t.Errorf("error should mention cosign verification, got: %v", err)
	}
}
