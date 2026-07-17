package cmd

import (
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/version"
)

func TestCatalogReleaseTag(t *testing.T) {
	origVersion := version.Version

	t.Run("dev version returns latest", func(t *testing.T) {
		version.Version = "dev"
		defer func() { version.Version = origVersion }()

		if got := catalogReleaseTag(); got != "latest" {
			t.Errorf("catalogReleaseTag() = %q, want %q", got, "latest")
		}
	})

	t.Run("release version returns version", func(t *testing.T) {
		version.Version = "v1.2.3"
		defer func() { version.Version = origVersion }()

		if got := catalogReleaseTag(); got != "v1.2.3" {
			t.Errorf("catalogReleaseTag() = %q, want %q", got, "v1.2.3")
		}
	})
}

func TestCatalogDownloadURL(t *testing.T) {
	origVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = origVersion }()

	got := catalogDownloadURL()
	want := "https://github.com/" + constants.GitHubRepo + "/releases/download/v1.0.0/catalog.json"
	if got != want {
		t.Errorf("catalogDownloadURL() = %q, want %q", got, want)
	}
}

func TestCatalogBundleDownloadURL(t *testing.T) {
	origVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = origVersion }()

	got := catalogBundleDownloadURL()
	want := "https://github.com/" + constants.GitHubRepo + "/releases/download/v1.0.0/catalog.json.sigstore.json"
	if got != want {
		t.Errorf("catalogBundleDownloadURL() = %q, want %q", got, want)
	}
}

func TestCatalogChecksumURL(t *testing.T) {
	origVersion := version.Version
	version.Version = "v1.0.0"
	defer func() { version.Version = origVersion }()

	got := catalogChecksumURL()
	want := "https://github.com/" + constants.GitHubRepo + "/releases/download/v1.0.0/catalog.json.sha256"
	if got != want {
		t.Errorf("catalogChecksumURL() = %q, want %q", got, want)
	}
}

func TestProviderCatalogCachePath(t *testing.T) {
	got, err := providerCatalogCachePath()
	if err != nil {
		t.Fatalf("providerCatalogCachePath() error = %v", err)
	}
	if !strings.HasSuffix(got, "/providers.catalog.json") {
		t.Errorf("providerCatalogCachePath() = %q, want suffix /providers.catalog.json", got)
	}
}
