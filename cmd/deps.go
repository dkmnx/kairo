package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/integrity"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
)

// osProcessRunner delegates process operations to the os/exec and os packages.
type osProcessRunner struct{}

func (osProcessRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }
func (osProcessRunner) ExecCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}
func (osProcessRunner) ExitProcess(code int) { os.Exit(code) }

// prodWrapperService delegates wrapper operations to the wrapper package.
type prodWrapperService struct{}

func (prodWrapperService) CreateTempAuthDir() (string, error) {
	return wrapper.CreateTempAuthDir()
}
func (prodWrapperService) WriteTempTokenFile(authDir, token string) (string, error) {
	return wrapper.WriteTempTokenFile(authDir, token)
}
func (prodWrapperService) GenerateWrapperScript(cfg wrapper.ScriptConfig) (string, bool, error) {
	return wrapper.GenerateWrapperScript(cfg)
}

// prodUpdateService delegates update operations to the update and ui packages.
type prodUpdateService struct {
	client *update.Client
}

func (s *prodUpdateService) FetchLatestRelease(ctx context.Context) (*update.Release, error) {
	return s.client.FetchLatestRelease(ctx)
}
func (prodUpdateService) ConfirmUpdate(message string) (bool, error) {
	return ui.Confirm(message)
}
func (s *prodUpdateService) DownloadToTempFile(ctx context.Context, url string) (string, error) {
	return s.client.DownloadToTempFile(ctx, url)
}
func (s *prodUpdateService) DownloadAndParseChecksums(ctx context.Context, url string) (map[string]string, error) {
	return s.client.DownloadAndParseChecksums(ctx, url)
}
func (prodUpdateService) VerifyChecksum(scriptPath, expectedHash string) error {
	return update.VerifyChecksum(scriptPath, expectedHash)
}
func (s *prodUpdateService) RunInstallScript(scriptPath string) error {
	return s.client.RunInstallScript(scriptPath)
}
func (s *prodUpdateService) VerifyCosignBundle(ctx context.Context, tag string) error {
	return s.client.VerifyCosignBundle(ctx, tag)
}

// providerCatalogCachePath returns the filesystem path for the cached
// provider catalog JSON file.
func providerCatalogCachePath() (string, error) {
	cfgDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}

	return cfgDir + "/providers.catalog.json", nil
}

// catalogReleaseTag returns the Git tag used for catalog release artifacts.
// In production builds it matches the running binary version; in dev it falls
// back to "latest" (the latest stable release).
func catalogReleaseTag() string {
	if version.Version != "dev" {
		return version.Version
	}

	return "latest"
}

// catalogDownloadURL returns the download URL for the catalog.json artifact.
func catalogDownloadURL() string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json",
		constants.GitHubRepo, catalogReleaseTag())
}

// catalogBundleDownloadURL returns the download URL for the catalog sigstore bundle.
func catalogBundleDownloadURL() string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json.sigstore.json",
		constants.GitHubRepo, catalogReleaseTag())
}

// catalogChecksumURL returns the download URL for the catalog SHA256 checksum.
func catalogChecksumURL() string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/catalog.json.sha256",
		constants.GitHubRepo, catalogReleaseTag())
}

// prodCatalogService is the production CatalogService that delegates to
// the providers.DefaultRegistry and fetches verified remote catalogs.
type prodCatalogService struct{}

func (prodCatalogService) ProviderList() []string {
	return providers.ProviderList()
}

func (prodCatalogService) ProviderSource(name string) string {
	return providers.DefaultRegistry.ProviderSource(name)
}

func (prodCatalogService) BuiltInProvider(name string) (providers.ProviderDefinition, bool) {
	return providers.BuiltInProvider(name)
}

func (prodCatalogService) RefreshFromRemote(ctx context.Context) (int, error) {
	cachePath, err := providerCatalogCachePath()
	if err != nil {
		return 0, err
	}

	artifactURL := catalogDownloadURL()
	bundleURL := catalogBundleDownloadURL()
	checksumURL := catalogChecksumURL()

	if u, ok := os.LookupEnv("KAIRO_PROVIDER_CATALOG_URL"); ok && u != "" {
		artifactURL = u
	}
	if u, ok := os.LookupEnv("KAIRO_PROVIDER_CATALOG_BUNDLE_URL"); ok && u != "" {
		bundleURL = u
	}
	if u, ok := os.LookupEnv("KAIRO_PROVIDER_CATALOG_CHECKSUM_URL"); ok && u != "" {
		checksumURL = u
	}

	data, err := integrity.FetchVerified(
		ctx,
		&http.Client{Timeout: 30 * time.Second},
		exec.LookPath,
		exec.CommandContext,
		artifactURL,
		bundleURL,
		checksumURL,
	)
	if err != nil {
		return 0, err
	}

	return providers.DefaultRegistry.RefreshCacheFromBytes(data, cachePath)
}

func loadProviderCacheOrDisk() {
	cachePath, err := providerCatalogCachePath()
	if err != nil {
		return
	}

	_ = providers.DefaultRegistry.LoadCache(cachePath)
}

func init() {
	loadProviderCacheOrDisk()
}

// NewDeps returns a Deps with production implementations.
func NewDeps() *Deps {
	return &Deps{
		Process: osProcessRunner{},
		Wrapper: prodWrapperService{},
		Update:  &prodUpdateService{client: update.NewClient()},
		Crypto:  crypto.DefaultService{},
		Catalog: prodCatalogService{},
	}
}
