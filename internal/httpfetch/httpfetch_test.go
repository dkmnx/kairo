package httpfetch

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func TestVerifySHA256(t *testing.T) {
	t.Run("matching hash", func(t *testing.T) {
		data := []byte("hello world")
		// sha256("hello world") = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
		err := VerifySHA256(data, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
		if err != nil {
			t.Errorf("VerifySHA256() error = %v", err)
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		data := []byte("hello world")
		err := VerifySHA256(data, "0000000000000000000000000000000000000000000000000000000000000000")
		if err == nil {
			t.Error("should return error on hash mismatch")
		}
	})
	t.Run("case insensitive", func(t *testing.T) {
		data := []byte("hello world")
		err := VerifySHA256(data, "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9")
		if err != nil {
			t.Errorf("should be case-insensitive, error = %v", err)
		}
	})
	t.Run("empty data", func(t *testing.T) {
		// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
		err := VerifySHA256([]byte{}, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		if err != nil {
			t.Errorf("VerifySHA256() error = %v", err)
		}
	})
	t.Run("trims whitespace in expected hash", func(t *testing.T) {
		data := []byte("test")
		// sha256("test") = 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
		err := VerifySHA256(data, "  9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n")
		if err != nil {
			t.Errorf("VerifySHA256() should trim whitespace, error = %v", err)
		}
	})
}

func TestDoHTTPRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.Header.Get("User-Agent") != "kairo-cli" {
				t.Errorf("expected User-Agent kairo-cli, got %s", r.Header.Get("User-Agent"))
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))
		defer srv.Close()

		resp, err := DoHTTPRequest(context.Background(), srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		resp, err := DoHTTPRequest(context.Background(), srv.Client(), srv.URL)
		if err == nil {
			resp.Body.Close()
			t.Fatal("expected error for non-200 status")
		}
		var kErr *kairoerrors.KairoError
		if !errors.As(err, &kErr) {
			t.Errorf("expected KairoError, got %T", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		_, err := DoHTTPRequest(context.Background(), http.DefaultClient, "http://127.0.0.1:1") //nolint:bodyclose // response is nil on error
		if err == nil {
			t.Fatal("expected error for unreachable host")
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := DoHTTPRequest(ctx, srv.Client(), srv.URL) //nolint:bodyclose // response is nil on error
		if err == nil {
			t.Fatal("expected error for canceled context")
		}
	})
}

func TestDoHTTPGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := []byte("hello world")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write(expected)
		}))
		defer srv.Close()

		body, err := DoHTTPGet(context.Background(), srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(body, expected) {
			t.Errorf("body = %q, want %q", body, expected)
		}
	})

	t.Run("propagates non-200 error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := DoHTTPGet(context.Background(), srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error for 500 status")
		}
	})
}

func TestEnsureBodyWithinLimit(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		err := ensureBodyWithinLimit(strings.NewReader(""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		err := ensureBodyWithinLimit(bytes.NewReader(make([]byte, 1)))
		if err == nil {
			t.Error("expected error for body exceeding limit")
		}
	})
}

func TestWriteStreamToTemp(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		data := []byte("test data")
		path, err := WriteStreamToTemp(bytes.NewReader(data), "httpfetch-test-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("file content = %q, want %q", got, data)
		}
	})

	t.Run("body within limit", func(t *testing.T) {
		path, err := WriteStreamToTemp(bytes.NewReader([]byte("ok")), "httpfetch-limit-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		if info.Size() != 2 {
			t.Errorf("size = %d, want 2", info.Size())
		}
	})
}

func TestDataToTempFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		data := []byte("payload")
		path, err := DataToTempFile(data, "httpfetch-data-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("content = %q, want %q", got, data)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		path, err := DataToTempFile([]byte{}, "httpfetch-empty-*")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat failed: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("size = %d, want 0", info.Size())
		}
	})
}

func TestCosignVerifyBlob(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capturedName string
		var capturedArgs []string
		mockExec := func(_ context.Context, name string, args ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = args
			return exec.Command("true")
		}

		err := CosignVerifyBlob(context.Background(), mockExec, "/usr/bin/cosign", "bundle.sigstore", "artifact.bin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedName != "/usr/bin/cosign" {
			t.Errorf("cosign path = %s, want /usr/bin/cosign", capturedName)
		}
		argsStr := strings.Join(capturedArgs, " ")
		if !strings.Contains(argsStr, "--bundle=bundle.sigstore") {
			t.Errorf("expected --bundle flag, got args: %v", capturedArgs)
		}
		if !strings.Contains(argsStr, "--certificate-identity-regexp=") {
			t.Errorf("expected --certificate-identity-regexp flag, got args: %v", capturedArgs)
		}
	})

	t.Run("verification failure", func(t *testing.T) {
		mockExec := func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		}

		err := CosignVerifyBlob(context.Background(), mockExec, "/usr/bin/cosign", "bundle.sigstore", "artifact.bin")
		if err == nil {
			t.Fatal("expected error for cosign failure")
		}
		var kErr *kairoerrors.KairoError
		if !errors.As(err, &kErr) {
			t.Errorf("expected KairoError, got %T", err)
		}
	})
}
