package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dkmnx/kairo/internal/version"
)

func TestUpdateCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/dkmnx/kairo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"tag_name": "v1.2.0",
				"html_url": "https://github.com/dkmnx/kairo/releases/tag/v1.2.0",
				"body": "Release v1.2.0"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	originalGetter := envGetter
	envGetter = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { envGetter = originalGetter }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := getLatestRelease()
	if err != nil {
		t.Fatalf("getLatestRelease() error = %v", err)
	}

	if latest.TagName != "v1.2.0" {
		t.Errorf("expected tag v1.2.0, got %s", latest.TagName)
	}

	if !versionGreaterThan(version.Version, latest.TagName) {
		t.Errorf("expected version to be less than latest")
	}
}

func TestUpdateCommandNoNewVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/dkmnx/kairo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name": "v1.0.0"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	originalGetter := envGetter
	envGetter = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { envGetter = originalGetter }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := getLatestRelease()
	if err != nil {
		t.Fatalf("getLatestRelease() error = %v", err)
	}

	if versionGreaterThan(version.Version, latest.TagName) {
		t.Errorf("expected no update available when versions are equal")
	}
}

func TestUpdateCommandAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	originalGetter := envGetter
	envGetter = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { envGetter = originalGetter }()

	_, err := getLatestRelease()
	if err == nil {
		t.Error("getLatestRelease() should return error on API failure")
	}
}

func TestVersionNotification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/dkmnx/kairo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"tag_name": "v1.5.0",
				"html_url": "https://github.com/dkmnx/kairo/releases/tag/v1.5.0"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	originalGetter := envGetter
	envGetter = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { envGetter = originalGetter }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := getLatestRelease()
	if err != nil {
		t.Fatalf("getLatestRelease() error = %v", err)
	}

	if !versionGreaterThan(version.Version, latest.TagName) {
		t.Errorf("expected update notification for version %s vs %s", version.Version, latest.TagName)
	}
}

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
		got := versionGreaterThan(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("versionGreaterThan(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
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
			got := versionGreaterThan(tt.current, tt.latest)
			if got != tt.wantBool {
				t.Errorf("versionGreaterThan(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.wantBool)
			}
		})
	}
}

func TestVersionGreaterThanInvalidVersions(t *testing.T) {
	t.Run("returns false for invalid current version", func(t *testing.T) {
		result := versionGreaterThan("invalid-version", "v1.0.0")
		if result {
			t.Error("versionGreaterThan() should return false for invalid current version")
		}
	})

	t.Run("returns false for invalid latest version", func(t *testing.T) {
		result := versionGreaterThan("v1.0.0", "not-a-version")
		if result {
			t.Error("versionGreaterThan() should return false for invalid latest version")
		}
	})

	t.Run("returns false for both invalid versions", func(t *testing.T) {
		result := versionGreaterThan("bad", "also-bad")
		if result {
			t.Error("versionGreaterThan() should return false for both invalid versions")
		}
	})
}

func TestGetEnvFunc(t *testing.T) {
	t.Run("returns value and true when env var is set", func(t *testing.T) {
		// Set a temporary environment variable
		t.Setenv("KAIRO_TEST_VAR", "test-value")

		value, ok := getEnvFunc("KAIRO_TEST_VAR")
		if !ok {
			t.Error("getEnvFunc() ok = false, want true")
		}
		if value != "test-value" {
			t.Errorf("getEnvFunc() = %q, want 'test-value'", value)
		}
	})

	t.Run("returns empty string and false when env var is not set", func(t *testing.T) {
		// Unset to ensure it's not set
		_ = os.Unsetenv("KAIRO_NONEXISTENT_VAR")

		value, ok := getEnvFunc("KAIRO_NONEXISTENT_VAR")
		if ok {
			t.Error("getEnvFunc() ok = true, want false")
		}
		if value != "" {
			t.Errorf("getEnvFunc() = %q, want empty string", value)
		}
	})

	t.Run("returns false for empty env var", func(t *testing.T) {
		t.Setenv("KAIRO_EMPTY_VAR", "")

		value, ok := getEnvFunc("KAIRO_EMPTY_VAR")
		if ok {
			t.Error("getEnvFunc() ok = true, want false for empty value")
		}
		if value != "" {
			t.Errorf("getEnvFunc() = %q, want empty string", value)
		}
	})
}

func TestGetLatestRelease(t *testing.T) {
	t.Run("returns release on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"tag_name": "v2.0.0",
				"html_url": "https://github.com/dkmnx/kairo/releases/tag/v2.0.0",
				"body": "Release v2.0.0"
			}`))
		}))
		defer server.Close()

		// Override the update URL
		originalEnvGetter := envGetter
		envGetter = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return originalEnvGetter(key)
		}
		defer func() { envGetter = originalEnvGetter }()

		release, err := getLatestRelease()
		if err != nil {
			t.Fatalf("getLatestRelease() error = %v", err)
		}

		if release.TagName != "v2.0.0" {
			t.Errorf("release.TagName = %q, want 'v2.0.0'", release.TagName)
		}

		if release.HTMLURL != "https://github.com/dkmnx/kairo/releases/tag/v2.0.0" {
			t.Errorf("release.HTMLURL = %q, want 'https://github.com/dkmnx/kairo/releases/tag/v2.0.0'", release.HTMLURL)
		}
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		originalEnvGetter := envGetter
		envGetter = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return originalEnvGetter(key)
		}
		defer func() { envGetter = originalEnvGetter }()

		_, err := getLatestRelease()
		if err == nil {
			t.Error("getLatestRelease() should return error for 500 status")
		}
	})

	t.Run("returns error on timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate timeout by not responding
			<-r.Context().Done()
		}))
		defer server.Close()

		originalEnvGetter := envGetter
		envGetter = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return originalEnvGetter(key)
		}
		defer func() { envGetter = originalEnvGetter }()

		_, err := getLatestRelease()
		if err == nil {
			t.Error("getLatestRelease() should return error on timeout")
		}
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid": json}`))
		}))
		defer server.Close()

		originalEnvGetter := envGetter
		envGetter = func(key string) (string, bool) {
			if key == "KAIRO_UPDATE_URL" {
				return server.URL, true
			}
			return originalEnvGetter(key)
		}
		defer func() { envGetter = originalEnvGetter }()

		_, err := getLatestRelease()
		if err == nil {
			t.Error("getLatestRelease() should return error for invalid JSON")
		}
	})
}

func TestGetLatestReleaseURL(t *testing.T) {
	t.Run("uses environment variable when set", func(t *testing.T) {
		t.Setenv("KAIRO_UPDATE_URL", "https://custom.example.com/releases/latest")

		url := getLatestReleaseURL()
		expected := "https://custom.example.com/releases/latest"
		if url != expected {
			t.Errorf("getLatestReleaseURL() = %q, want %q", url, expected)
		}
	})

	t.Run("uses default URL when env var is not set", func(t *testing.T) {
		// Unset to ensure it's not set
		_ = os.Unsetenv("KAIRO_UPDATE_URL")

		url := getLatestReleaseURL()
		expected := defaultUpdateURL
		if url != expected {
			t.Errorf("getLatestReleaseURL() = %q, want %q", url, expected)
		}
	})

	t.Run("uses default URL when env var is empty", func(t *testing.T) {
		t.Setenv("KAIRO_UPDATE_URL", "")

		url := getLatestReleaseURL()
		expected := defaultUpdateURL
		if url != expected {
			t.Errorf("getLatestReleaseURL() = %q, want %q", url, expected)
		}
	})
}

func TestIsWindows(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected bool
	}{
		{"windows", "windows", true},
		{"linux", "linux", false},
		{"darwin", "darwin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWindows(tt.goos)
			if result != tt.expected {
				t.Errorf("isWindows(%q) = %v, want %v", tt.goos, result, tt.expected)
			}
		})
	}
}

func TestGetInstallScriptURL(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{
			name:     "windows returns ps1 script",
			goos:     "windows",
			expected: "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1",
		},
		{
			name:     "linux returns sh script",
			goos:     "linux",
			expected: "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh",
		},
		{
			name:     "darwin returns sh script",
			goos:     "darwin",
			expected: "https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstallScriptURL(tt.goos)
			if result != tt.expected {
				t.Errorf("getInstallScriptURL(%q) = %q, want %q", tt.goos, result, tt.expected)
			}
		})
	}
}

func TestDownloadToTempFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/install.sh" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("#!/bin/bash\necho 'install script content'"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	tempFile, err := downloadToTempFile(server.URL + "/install.sh")
	if err != nil {
		t.Fatalf("downloadToTempFile() error = %v", err)
	}
	defer os.Remove(tempFile)

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	expectedContent := "#!/bin/bash\necho 'install script content'"
	if string(content) != expectedContent {
		t.Errorf("temp file content = %q, want %q", string(content), expectedContent)
	}
}

func TestDownloadToTempFileCreatesTempFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	tempFile, err := downloadToTempFile(server.URL)
	if err != nil {
		t.Fatalf("downloadToTempFile() error = %v", err)
	}
	defer os.Remove(tempFile)

	info, err := os.Stat(tempFile)
	if err != nil {
		t.Fatalf("failed to stat temp file: %v", err)
	}

	// Check it's a regular file
	if info.Mode().IsDir() {
		t.Error("downloadToTempFile() created a directory, not a file")
	}
}

func TestDownloadToTempFileHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := downloadToTempFile(server.URL)
	if err == nil {
		t.Error("downloadToTempFile() should return error on HTTP failure")
	}
}
