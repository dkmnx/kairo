package cmd

import (
	"net/http"
	"net/http/httptest"
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
