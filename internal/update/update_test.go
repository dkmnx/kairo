package update

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
	t.Run("returns value and true when env var is set", func(t *testing.T) {
		t.Setenv("KAIRO_TEST_VAR", "test-value")
		value, ok := EnvFunc("KAIRO_TEST_VAR")
		if !ok {
			t.Error("EnvFunc() ok = false, want true")
		}
		if value != "test-value" {
			t.Errorf("EnvFunc() = %q, want 'test-value'", value)
		}
	})
	t.Run("returns empty string and false when env var is not set", func(t *testing.T) {
		_ = os.Unsetenv("KAIRO_NONEXISTENT_VAR")
		value, ok := EnvFunc("KAIRO_NONEXISTENT_VAR")
		if ok {
			t.Error("EnvFunc() ok = true, want false")
		}
		if value != "" {
			t.Errorf("EnvFunc() = %q, want empty string", value)
		}
	})
	t.Run("returns false for empty env var", func(t *testing.T) {
		t.Setenv("KAIRO_EMPTY_VAR", "")
		value, ok := EnvFunc("KAIRO_EMPTY_VAR")
		if ok {
			t.Error("EnvFunc() ok = true, want false for empty value")
		}
		if value != "" {
			t.Errorf("EnvFunc() = %q, want empty string", value)
		}
	})
}

func TestGetLatestRelease(t *testing.T) {
	t.Run("returns release on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://github.com/dkmnx/kairo/releases/tag/v2.0.0","body":"Release v2.0.0"}`))
		}))
		defer server.Close()
		original := EnvFunc
		EnvFunc = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return original(key)
		}
		defer func() { EnvFunc = original }()
		release, err := GetLatestRelease()
		if err != nil {
			t.Fatalf("GetLatestRelease() error = %v", err)
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
		original := EnvFunc
		EnvFunc = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return original(key)
		}
		defer func() { EnvFunc = original }()
		_, err := GetLatestRelease()
		if err == nil {
			t.Error("GetLatestRelease() should return error for 500 status")
		}
	})
	t.Run("returns error on invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid": json}`))
		}))
		defer server.Close()
		original := EnvFunc
		EnvFunc = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return original(key)
		}
		defer func() { EnvFunc = original }()
		_, err := GetLatestRelease()
		if err == nil {
			t.Error("GetLatestRelease() should return error for invalid JSON")
		}
	})
}

func TestGetLatestReleaseURL(t *testing.T) {
	t.Run("uses environment variable when set", func(t *testing.T) {
		t.Setenv("KAIRO_UPDATE_URL", "https://custom.example.com/releases/latest")
		url := GetLatestReleaseURL()
		if url != "https://custom.example.com/releases/latest" {
			t.Errorf("GetLatestReleaseURL() = %q, want %q", url, "https://custom.example.com/releases/latest")
		}
	})
	t.Run("uses default URL when env var is not set", func(t *testing.T) {
		_ = os.Unsetenv("KAIRO_UPDATE_URL")
		url := GetLatestReleaseURL()
		if url != constants.GitHubAPIReleasesLatest {
			t.Errorf("GetLatestReleaseURL() = %q, want %q", url, constants.GitHubAPIReleasesLatest)
		}
	})
}

func TestGetInstallScriptURL(t *testing.T) {
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
			result := GetInstallScriptURL(tt.goos, tt.tag)
			if result != tt.expected {
				t.Errorf("GetInstallScriptURL(%q, %q) = %q, want %q", tt.goos, tt.tag, result, tt.expected)
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
	tempFile, err := DownloadToTempFile(server.URL)
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
	_, err := DownloadToTempFile(server.URL)
	if err == nil {
		t.Error("DownloadToTempFile() should return error on HTTP failure")
	}
}

func TestDownloadToTempFileErrorHandling(t *testing.T) {
	t.Run("returns error for invalid URL", func(t *testing.T) {
		_, err := DownloadToTempFile("://invalid-url")
		if err == nil {
			t.Error("should return error for invalid URL")
		}
	})
	t.Run("returns error on HTTP 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		_, err := DownloadToTempFile(server.URL)
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
		tempFile, err := DownloadToTempFile(server.URL)
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
	tempFile, err := DownloadToTempFile(server.URL)
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

func TestGetChecksumsURL(t *testing.T) {
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
			result := GetChecksumsURL(tt.tag)
			if result != tt.expected {
				t.Errorf("GetChecksumsURL(%q) = %q, want %q", tt.tag, result, tt.expected)
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
	t.Run("parses valid checksums file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Comment\n\n07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790  scripts/install.sh\n"))
		}))
		defer server.Close()
		checksums, err := DownloadAndParseChecksums(server.URL)
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
		_, err := DownloadAndParseChecksums(server.URL)
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

func TestGetScriptNameForChecksums(t *testing.T) {
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
			result := GetScriptNameForChecksums(tt.goos)
			if result != tt.expected {
				t.Errorf("GetScriptNameForChecksums(%q) = %q, want %q", tt.goos, result, tt.expected)
			}
		})
	}
}

func TestScriptNameMatchesChecksumFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Kairo release checksums\n07203eb32c914886d316468e4dedc18a1df65c3e84ad3bff63474b3ce1bb2790  scripts/install.sh\na197cd3c17f40fad8ae08df1ce42633e454491319df40097abf78da01db5aaae  scripts/install.ps1\n"))
	}))
	defer server.Close()
	checksums, err := DownloadAndParseChecksums(server.URL)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		scriptName := GetScriptNameForChecksums(goos)
		if _, ok := checksums[scriptName]; !ok {
			t.Errorf("GetScriptNameForChecksums(%q) = %q not found in checksums map keys: %v",
				goos, scriptName, maps.Keys(checksums))
		}
	}
}
