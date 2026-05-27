package cmd

import (
	"context"
	"os"
	"os/exec"

	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/update"
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
func (prodUpdateService) RunInstallScript(scriptPath string) error {
	return update.RunInstallScript(scriptPath)
}
func (s *prodUpdateService) VerifyCosignBundle(ctx context.Context, tag string) error {
	return s.client.VerifyCosignBundle(ctx, tag)
}

// NewDeps returns a Deps with production implementations.
func NewDeps() *Deps {
	return &Deps{
		Process: osProcessRunner{},
		Wrapper: prodWrapperService{},
		Update:  &prodUpdateService{client: update.NewClient()},
		Crypto:  crypto.DefaultService{},
	}
}
