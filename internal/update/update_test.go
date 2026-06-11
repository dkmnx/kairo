package update

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/constants"
)

func TestVersionGreaterThan(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v1.0.0", "v1.1.0", true},
		{"v1.0.0", "v2.0.0", true},
		{"v1.1.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"v2.0.0", "v1.0.0", false},
	}
	for _, tt := range tests {
		got := VersionGreaterThan(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("VersionGreaterThan(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestVersionGreaterThanEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		wantBool bool
	}{
		{"patch version", "v1.0.0", "v1.0.1", true},
		{"minor version", "v1.0.0", "v1.1.0", true},
		{"major version", "v1.0.0", "v2.0.0", true},
		{"pre-release after patch", "v1.0.0", "v1.0.1-alpha", true},
		{"pre-release beta", "v1.0.0", "v1.0.1-beta.1", true},
		{"rc version", "v1.0.0", "v1.0.1-rc.1", true},
		{"alpha vs beta", "v1.0.1-alpha", "v1.0.1-beta", true},
		{"build metadata", "v1.0.0+build123", "v1.0.1", true},
		{"v0 versions", "v0.9.0", "v0.10.0", true},
		{"many patch digits", "v1.0.0", "v1.0.10", true},
		{"many minor digits", "v1.0.0", "v1.10.0", true},
		{"pre-release vs release", "v1.0.1-alpha", "v1.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VersionGreaterThan(tt.current, tt.latest)
			if got != tt.wantBool {
				t.Errorf("VersionGreaterThan(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.wantBool)
			}
		})
	}
}

func TestVersionGreaterThanInvalidVersions(t *testing.T) {
	t.Run("returns false for invalid current version", func(t *testing.T) {
		if VersionGreaterThan("invalid-version", "v1.0.0") {
			t.Error("should return false for invalid current version")
		}
	})
	t.Run("returns false for invalid latest version", func(t *testing.T) {
		if VersionGreaterThan("v1.0.0", "not-a-version") {
			t.Error("should return false for invalid latest version")
		}
	})
	t.Run("returns false for both invalid versions", func(t *testing.T) {
		if VersionGreaterThan("bad", "also-bad") {
			t.Error("should return false for both invalid versions")
		}
	})
}

func TestEnvFunc(t *testing.T) {
	c := NewClient()
	t.Run("returns value and true when env var is set", func(t *testing.T) {
		t.Setenv("KAIRO_TEST_VAR", "test-value")
		value, ok := c.EnvFunc("KAIRO_TEST_VAR")
		if !ok {
			t.Error("EnvFunc() ok = false, want true")
		}
		if value != "test-value" {
			t.Errorf("EnvFunc() = %q, want 'test-value'", value)
		}
	})
	t.Run("returns empty string and false when env var is not set", func(t *testing.T) {
		_ = os.Unsetenv("KAIRO_NONEXISTENT_VAR")
		value, ok := c.EnvFunc("KAIRO_NONEXISTENT_VAR")
		if ok {
			t.Error("EnvFunc() ok = true, want false")
		}
		if value != "" {
			t.Errorf("EnvFunc() = %q, want empty string", value)
		}
	})
	t.Run("returns false for empty env var", func(t *testing.T) {
		t.Setenv("KAIRO_EMPTY_VAR", "")
		value, ok := c.EnvFunc("KAIRO_EMPTY_VAR")
		if ok {
			t.Error("EnvFunc() ok = true, want false for empty value")
		}
		if value != "" {
			t.Errorf("EnvFunc() = %q, want empty string", value)
		}
	})
}

func TestFetchLatestRelease(t *testing.T) {
	t.Run("returns release on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://github.com/dkmnx/kairo/releases/tag/v2.0.0","body":"Release v2.0.0"}`))
		}))
		defer server.Close()
		c := &Client{
			HTTPClient: &http.Client{},
			EnvFunc: func(key string) (string, bool) {
				if key == "KAIRO_UPDATE_URL" {
					return server.URL, true
				}
				return "", false
			},
		}
		release, err := c.FetchLatestRelease(context.Background())
		if err != nil {
			t.Fatalf("FetchLatestRelease() error = %v", err)
		}
		if release.TagName != "v2.0.0" {
			t.Errorf("release.TagName = %q, want 'v2.0.0'", release.TagName)
		}
	})
	t.Run("returns error on HTTP failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		c := &Client{
			HTTPClient: &http.Client{},
			EnvFunc: func(key string) (string, bool) {
				if key == "KAIRO_UPDATE_URL" {
					return server.URL, true
				}
				return "", false
			},
		}
		_, err := c.FetchLatestRelease(context.Background())
		if err == nil {
			t.Error("FetchLatestRelease() should return error for 500 status")
		}
	})
	t.Run("returns error on invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid": json}`))
		}))
		defer server.Close()
		c := &Client{
			HTTPClient: &http.Client{},
			EnvFunc: func(key string) (string, bool) {
				if key == "KAIRO_UPDATE_URL" {
					return server.URL, true
				}
				return "", false
			},
		}
		_, err := c.FetchLatestRelease(context.Background())
		if err == nil {
			t.Error("FetchLatestRelease() should return error for invalid JSON")
		}
	})
}

func TestLatestReleaseURL(t *testing.T) {
	t.Run("uses environment variable when set", func(t *testing.T) {
		c := &Client{
			EnvFunc: func(key string) (string, bool) {
				if key == "KAIRO_UPDATE_URL" {
					return "https://custom.example.com/releases/latest", true
				}
				return "", false
			},
		}
		url := c.LatestReleaseURL()
		if url != "https://custom.example.com/releases/latest" {
			t.Errorf("LatestReleaseURL() = %q, want %q", url, "https://custom.example.com/releases/latest")
		}
	})
	t.Run("uses default URL when env var is not set", func(t *testing.T) {
		c := &Client{
			EnvFunc: func(string) (string, bool) { return "", false },
		}
		url := c.LatestReleaseURL()
		if url != constants.GitHubAPIReleasesLatest {
			t.Errorf("LatestReleaseURL() = %q, want %q", url, constants.GitHubAPIReleasesLatest)
		}
	})
}

func TestInstallScriptURL(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		tag      string
		expected string
	}{
		{"windows returns ps1", "windows", "v1.0.0", "https://raw.githubusercontent.com/dkmnx/kairo/v1.0.0/scripts/install.ps1"},
		{"linux returns sh", "linux", "v2.0.0", "https://raw.githubusercontent.com/dkmnx/kairo/v2.0.0/scripts/install.sh"},
		{"darwin returns sh", "darwin", "v1.5.0", "https://raw.githubusercontent.com/dkmnx/kairo/v1.5.0/scripts/install.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InstallScriptURL(tt.goos, tt.tag)
			if result != tt.expected {
				t.Errorf("InstallScriptURL(%q, %q) = %q, want %q", tt.goos, tt.tag, result, tt.expected)
			}
		})
	}
}

func TestDownloadToTempFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("#!/bin/bash\necho 'install script content'"))
	}))
	defer server.Close()
	c := NewClient()
	tempFile, err := c.DownloadToTempFile(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("DownloadToTempFile() error = %v", err)
	}
	defer os.Remove(tempFile)
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(content) != "#!/bin/bash\necho 'install script content'" {
		t.Errorf("temp file content = %q", string(content))
	}
}

func TestDownloadToTempFileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	c := NewClient()
	_, err := c.DownloadToTempFile(context.Background(), server.URL)
	if err == nil {
		t.Error("DownloadToTempFile() should return error on HTTP failure")
	}
}

func TestDownloadToTempFileErrorHandling(t *testing.T) {
	c := NewClient()
	t.Run("returns error for invalid URL", func(t *testing.T) {
		_, err := c.DownloadToTempFile(context.Background(), "://invalid-url")
		if err == nil {
			t.Error("should return error for invalid URL")
		}
	})
	t.Run("returns error on HTTP 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		_, err := c.DownloadToTempFile(context.Background(), server.URL)
		if err == nil {
			t.Error("should return error on 500 status")
		}
	})
	t.Run("handles large download", func(t *testing.T) {
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(largeData)
		}))
		defer server.Close()
		tempFile, err := c.DownloadToTempFile(context.Background(), server.URL)
		if err != nil {
			t.Errorf("failed with large download: %v", err)
		}
		defer os.Remove(tempFile)
		info, err := os.Stat(tempFile)
		if err != nil {
			t.Fatalf("failed to stat temp file: %v", err)
		}
		if info.Size() != int64(len(largeData)) {
			t.Errorf("file size = %d, want %d", info.Size(), len(largeData))
		}
	})
}

func TestDownloadToTempFileExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()
	c := NewClient()
	tempFile, err := c.DownloadToTempFile(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("DownloadToTempFile() error = %v", err)
	}
	defer os.Remove(tempFile)
	expectedExt := ".sh"
	if runtime.GOOS == "windows" {
		expectedExt = ".ps1"
	}
	if !strings.HasSuffix(tempFile, expectedExt) {
		t.Errorf("temp file %q should have %s extension", tempFile, expectedExt)
	}
}

func TestRunInstallScript_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0"), 0644); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	if err := RunInstallScript(scriptPath); err != nil {
		t.Errorf("RunInstallScript() error = %v", err)
	}
}

func TestRunInstallScript_ExecutionFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1"), 0644); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	if err := RunInstallScript(scriptPath); err == nil {
		t.Error("should return error when script fails")
	}
}

func TestRunInstallScript_ScriptNotFound(t *testing.T) {
	if err := RunInstallScript("/nonexistent/path/to/script.sh"); err == nil {
		t.Error("should return error when script not found")
	}
}

func TestChecksumsURL(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{"v1", "v1.0.0", "https://raw.githubusercontent.com/dkmnx/kairo/v1.0.0/scripts/checksums.txt"},
		{"v2", "v2.0.0", "https://raw.githubusercontent.com/dkmnx/kairo/v2.0.0/scripts/checksums.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChecksumsURL(tt.tag)
			if result != tt.expected {
				t.Errorf("ChecksumsURL(%q) = %q, want %q", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestParseChecksumLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantHash string
		wantFile string
		wantOk   bool
	}{
		{"valid sh", "07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790  scripts/install.sh", "07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790", "scripts/install.sh", true},
		{"valid ps1", "a197cd3c17f40fad8ae08df1ce42633e454491319df40097abf78da01db5aaae  scripts/install.ps1", "a197cd3c17f40fad8ae08df1ce42633e454491319df40097abf78da01db5aaae", "scripts/install.ps1", true},
		{"uppercase", "ABCD1234567890ABCD1234567890ABCD1234567890ABCD1234567890ABCD1234  scripts/test.sh", "abcd1234567890abcd1234567890abcd1234567890abcd1234567890abcd1234", "scripts/test.sh", true},
		{"comment", "# This is a comment", "", "", false},
		{"empty", "", "", "", false},
		{"whitespace", "   ", "", "", false},
		{"invalid hash", "abc123  scripts/test.sh", "", "", false},
		{"too few fields", "07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, filename, ok := ParseChecksumLine(tt.line)
			if ok != tt.wantOk {
				t.Errorf("ParseChecksumLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOk)
			}
			if ok {
				if hash != tt.wantHash {
					t.Errorf("hash = %q, want %q", hash, tt.wantHash)
				}
				if filename != tt.wantFile {
					t.Errorf("filename = %q, want %q", filename, tt.wantFile)
				}
			}
		})
	}
}

func TestDownloadAndParseChecksums(t *testing.T) {
	c := NewClient()
	t.Run("parses valid checksums file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Comment\n\n07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790  scripts/install.sh\n"))
		}))
		defer server.Close()
		checksums, err := c.DownloadAndParseChecksums(context.Background(), server.URL)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(checksums) != 1 {
			t.Errorf("expected 1 checksum, got %d", len(checksums))
		}
	})
	t.Run("returns error on HTTP failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		_, err := c.DownloadAndParseChecksums(context.Background(), server.URL)
		if err == nil {
			t.Error("should return error on 404")
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	t.Run("verifies matching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'test'"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := VerifyChecksum(scriptPath, "bd78896dd21dbaed057f004b4e194c0bc2444d8a8e16775a3e6a511d17ab32ad"); err != nil {
			t.Errorf("VerifyChecksum() error = %v", err)
		}
	})
	t.Run("returns error on mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'test'"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := VerifyChecksum(scriptPath, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
			t.Error("should return error on hash mismatch")
		}
	})
	t.Run("is case-insensitive", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test.sh")
		if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'test'"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := VerifyChecksum(scriptPath, "BD78896DD21DBAED057F004B4E194C0BC2444D8A8E16775A3E6A511D17AB32AD"); err != nil {
			t.Errorf("should be case-insensitive, error = %v", err)
		}
	})
}

func TestScriptNameForChecksums(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{"linux", "linux", "scripts/install.sh"},
		{"darwin", "darwin", "scripts/install.sh"},
		{"windows", "windows", "scripts/install.ps1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScriptNameForChecksums(tt.goos)
			if result != tt.expected {
				t.Errorf("ScriptNameForChecksums(%q) = %q, want %q", tt.goos, result, tt.expected)
			}
		})
	}
}

func TestDoHTTPGet_InvalidURL(t *testing.T) {
	c := NewClient()
	_, err := c.doHTTPGet(context.Background(), "://invalid-url")
	if err == nil {
		t.Error("doHTTPGet() should return error for invalid URL")
	}
}

func TestDoHTTPGet_ConnectionRefused(t *testing.T) {
	c := &Client{
		HTTPClient:   &http.Client{Timeout: 100 * time.Millisecond},
		EnvFunc:      func(string) (string, bool) { return "", false },
		LookPathFunc: func(string) (string, error) { return "", fmt.Errorf("not found") },
	}
	_, err := c.doHTTPGet(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Error("doHTTPGet() should return error when connection is refused")
	}
}

func TestDoHTTPGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient()
	_, err := c.doHTTPGet(context.Background(), server.URL)
	if err == nil {
		t.Error("doHTTPGet() should return error for 404")
	}
}

func TestDoHTTPGet_BodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, maxHTTPBodySize))
	}))
	defer server.Close()

	c := NewClient()
	_, err := c.doHTTPGet(context.Background(), server.URL)
	if err == nil {
		t.Error("doHTTPGet() should return error when body exceeds size limit")
	}
}

func TestDownloadToTempFile_BodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, maxHTTPBodySize))
	}))
	defer server.Close()

	c := NewClient()
	_, err := c.DownloadToTempFile(context.Background(), server.URL)
	if err == nil {
		t.Error("DownloadToTempFile() should return error when body exceeds size limit")
	}
}

func TestDownloadToTempFile_WriteFails(t *testing.T) {
	// Server that closes connection prematurely, causing io.Copy to fail
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Hijack and close the connection to cause write failure
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server does not support hijacking")
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	c := NewClient()
	_, err := c.DownloadToTempFile(context.Background(), server.URL)
	if err == nil {
		t.Error("DownloadToTempFile() should return error when write fails")
	}
}

func TestRunInstallScript_ChmodFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	err := RunInstallScript("/invalid/path/to/script.sh")
	if err == nil {
		t.Error("RunInstallScript() should return error for non-writable path")
	}
}

func TestVerifyChecksum_FileNotFound(t *testing.T) {
	err := VerifyChecksum("/nonexistent/file.sh", "abcdef")
	if err == nil {
		t.Error("VerifyChecksum() should return error for nonexistent file")
	}
}

func TestScriptNameMatchesChecksumFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Kairo release checksums\n07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790  scripts/install.sh\na197cd3c17f40fad8ae08df1ce42633e454491319df40097abf78da01db5aaae  scripts/install.ps1\n"))
	}))
	defer server.Close()
	c := NewClient()
	checksums, err := c.DownloadAndParseChecksums(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		scriptName := ScriptNameForChecksums(goos)
		if _, ok := checksums[scriptName]; !ok {
			t.Errorf("ScriptNameForChecksums(%q) = %q not found in checksums map keys: %v",
				goos, scriptName, maps.Keys(checksums))
		}
	}
}

func TestChecksumsBundleURL(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{"v1", "v1.0.0", "https://raw.githubusercontent.com/dkmnx/kairo/v1.0.0/scripts/checksums.txt.sigstore.json"},
		{"v2", "v2.5.1", "https://raw.githubusercontent.com/dkmnx/kairo/v2.5.1/scripts/checksums.txt.sigstore.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChecksumsBundleURL(tt.tag)
			if result != tt.expected {
				t.Errorf("ChecksumsBundleURL(%q) = %q, want %q", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestFetchLatestRelease_InvalidURL(t *testing.T) {
	c := &Client{
		HTTPClient: &http.Client{},
		EnvFunc: func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return "://invalid-url", true
			}
			return "", false
		},
	}
	_, err := c.FetchLatestRelease(context.Background())
	if err == nil {
		t.Error("FetchLatestRelease() should return error for invalid URL")
	}
}

func TestVerifyCosignBundle_CosignNotInstalled(t *testing.T) {
	c := &Client{
		HTTPClient:   &http.Client{},
		EnvFunc:      func(string) (string, bool) { return "", false },
		LookPathFunc: func(string) (string, error) { return "", fmt.Errorf("not found") },
	}
	err := c.VerifyCosignBundle(context.Background(), "v1.0.0")
	if err != nil {
		t.Errorf("VerifyCosignBundle should return nil when cosign not installed, got: %v", err)
	}
}

func TestVerifyCosignBundle_BundleDownloadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &Client{
		HTTPClient: &http.Client{},
		EnvFunc: func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return "", false
		},
		LookPathFunc: func(string) (string, error) { return "/usr/bin/cosign", nil },
	}
	err := c.VerifyCosignBundle(context.Background(), "v1.0.0")
	if err == nil {
		t.Error("VerifyCosignBundle should return error when bundle download fails")
	}
}

func TestVerifyCosignBundle_ChecksumsDownloadFails(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: bundle download succeeds
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": "test"}`))
		} else {
			// Second call: checksums download fails
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	c := &Client{
		HTTPClient: &http.Client{},
		EnvFunc: func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return "", false
		},
		LookPathFunc: func(string) (string, error) { return "/usr/bin/cosign", nil },
	}
	err := c.VerifyCosignBundle(context.Background(), "v1.0.0")
	if err == nil {
		t.Error("VerifyCosignBundle should return error when checksums download fails")
	}
}
