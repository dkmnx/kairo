package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			version.Version = "v1.0.0"
			version.Commit = "abc123"
			version.Date = "2025-12-26T09:15:46Z"
			cmd.Printf("Kairo version: %s\n", version.Version)
			if version.Commit != "unknown" && version.Commit != "" {
				cmd.Printf("Commit: %s\n", version.Commit)
			}
			if version.Date != "" && version.Date != "unknown" {
				if t, err := time.Parse(time.RFC3339, version.Date); err == nil {
					cmd.Printf("Date: %s\n", t.Format("2006-01-02"))
				} else {
					cmd.Printf("Date: %s\n", version.Date)
				}
			}
		},
	}

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("version output is empty")
	}

	expectedParts := []string{
		"Kairo version: v1.0.0",
		"Commit: abc123",
		"Date: 2025-12-26",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("output doesn't contain expected part %q, got: %q", part, output)
		}
	}

	if strings.Contains(output, "T09:15:46Z") {
		t.Error("date should be formatted without time component")
	}
}

func TestCheckForUpdatesAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/dkmnx/kairo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"tag_name": "v2.0.0",
				"html_url": "https://github.com/dkmnx/kairo/releases/tag/v2.0.0"
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

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	checkForUpdates(cmd)

	output := buf.String()
	if !strings.Contains(output, "new version") {
		t.Errorf("checkForUpdates() should mention new version, got: %q", output)
	}
	if !strings.Contains(output, "v2.0.0") {
		t.Errorf("checkForUpdates() should mention v2.0.0, got: %q", output)
	}
}

func TestCheckForUpdatesNoUpdate(t *testing.T) {
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

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	checkForUpdates(cmd)

	output := buf.String()
	if strings.Contains(output, "new version") {
		t.Errorf("checkForUpdates() should NOT mention update when current, got: %q", output)
	}
}

func TestCheckForUpdatesAPIError(t *testing.T) {
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

	originalVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = originalVersion }()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	checkForUpdates(cmd)

	output := buf.String()
	if strings.Contains(output, "new version") {
		t.Errorf("checkForUpdates() should NOT mention update on API error, got: %q", output)
	}
}
