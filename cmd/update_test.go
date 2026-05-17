package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/version"
)

func hijackAndClose(w http.ResponseWriter) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	conn.Close()
}

func TestUpdateCommand_DoesNotMigrateConfigAfterInstall(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	originalVersion := version.Version
	version.Version = "v2.3.4"
	defer func() { version.Version = originalVersion }()

	tempScriptPath := filepath.Join(tmpDir, "install.sh")
	if err := os.WriteFile(tempScriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write temp script: %v", err)
	}

	d := testDeps(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate) {
		mu.GetLatestReleaseFn = func() (*update.Release, error) {
			return &update.Release{TagName: "v2.3.5"}, nil
		}
		mu.ConfirmUpdateFn = func(string) (bool, error) {
			return true, nil
		}
		mu.DownloadToTempFileFn = func(string) (string, error) {
			return tempScriptPath, nil
		}
		mu.DownloadAndParseChecksumsFn = func(string) (map[string]string, error) {
			return map[string]string{update.GetScriptNameForChecksums(runtime.GOOS): "ignored"}, nil
		}
		mu.VerifyChecksumFn = func(string, string) error {
			return nil
		}
		mu.RunInstallScriptFn = func(string) error {
			return nil
		}
	})

	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(tmpDir)
	cliCtx.SetDeps(d)
	updateCmd.SetContext(WithCLIContext(context.Background(), cliCtx))
	updateCmd.Run(updateCmd, nil)

	updatedConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config after update: %v", err)
	}
	if !strings.Contains(string(updatedConfig), "model: glm-4.7") {
		t.Fatalf("update command should not migrate config after install, got: %s", string(updatedConfig))
	}
}

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

	originalEnvFunc := update.EnvFunc
	update.EnvFunc = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { update.EnvFunc = originalEnvFunc }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := update.GetLatestRelease()
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}

	if latest.TagName != "v1.2.0" {
		t.Errorf("expected tag v1.2.0, got %s", latest.TagName)
	}

	if !update.VersionGreaterThan(version.Version, latest.TagName) {
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

	originalEnvFunc := update.EnvFunc
	update.EnvFunc = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { update.EnvFunc = originalEnvFunc }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := update.GetLatestRelease()
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}

	if update.VersionGreaterThan(version.Version, latest.TagName) {
		t.Errorf("expected no update available when versions are equal")
	}
}

func TestUpdateCommandAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	originalEnvFunc := update.EnvFunc
	update.EnvFunc = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { update.EnvFunc = originalEnvFunc }()

	_, err := update.GetLatestRelease()
	if err == nil {
		t.Error("GetLatestRelease() should return error on API failure")
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

	originalEnvFunc := update.EnvFunc
	update.EnvFunc = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return server.URL + "/repos/dkmnx/kairo/releases/latest", true
		}
		return "", false
	}
	defer func() { update.EnvFunc = originalEnvFunc }()

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	latest, err := update.GetLatestRelease()
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}

	if !update.VersionGreaterThan(version.Version, latest.TagName) {
		t.Errorf("expected update notification for version %s vs %s", version.Version, latest.TagName)
	}
}

func TestDownloadToTempFileErrorHandlingConnectionClose(t *testing.T) {
	if runningWithRaceDetector() {
		t.Skip("Skipping error handling tests with race detector")
	}

	t.Run("returns error when server closes connection early", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hijackAndClose(w)
		}))
		defer server.Close()

		_, err := update.DownloadToTempFile(server.URL)
		if err == nil {
			t.Error("should return error when server closes early")
		}
	})
}
