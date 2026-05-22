package cmd

import (
	"context"
	"os/exec"

	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/wrapper"
)

// ProcessRunner provides process execution operations.
type ProcessRunner interface {
	LookPath(file string) (string, error)
	ExecCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd
	ExitProcess(code int)
}

// WrapperService provides wrapper script generation and temp auth operations.
type WrapperService interface {
	CreateTempAuthDir() (string, error)
	WriteTempTokenFile(authDir, token string) (string, error)
	GenerateWrapperScript(cfg wrapper.ScriptConfig) (string, bool, error)
}

// UpdateService provides version checking and self-update operations.
type UpdateService interface {
	FetchLatestRelease(ctx context.Context) (*update.Release, error)
	ConfirmUpdate(message string) (bool, error)
	DownloadToTempFile(ctx context.Context, url string) (string, error)
	DownloadAndParseChecksums(ctx context.Context, url string) (map[string]string, error)
	VerifyChecksum(scriptPath, expectedHash string) error
	VerifyCosignBundle(ctx context.Context, tag string) error
	RunInstallScript(scriptPath string) error
}

// Deps holds all external dependencies as interfaces.
// Production code uses NewDeps(); tests inject mocks via CLIContext.SetDeps.
type Deps struct {
	Process ProcessRunner
	Wrapper WrapperService
	Update  UpdateService
}
