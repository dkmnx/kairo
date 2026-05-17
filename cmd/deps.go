package cmd

import (
	"context"
	"os"
	"os/exec"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/wrapper"
)

// Deps holds all external dependencies that can be replaced in tests.
// Production code uses NewDeps(); tests inject custom implementations via CLIContext.
type Deps struct {
	// Process operations
	LookPath           func(file string) (string, error)
	ExecCommand        func(name string, arg ...string) *exec.Cmd
	ExecCommandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
	ExitProcess        func(code int)

	// Wrapper operations
	CreateTempAuthDir     func() (string, error)
	WriteTempTokenFile    func(authDir, token string) (string, error)
	GenerateWrapperScript func(cfg wrapper.ScriptConfig) (string, bool, error)

	// Update operations
	GetLatestRelease          func() (*update.Release, error)
	ConfirmUpdate             func(message string) (bool, error)
	DownloadToTempFile        func(url string) (string, error)
	DownloadAndParseChecksums func(url string) (map[string]string, error)
	VerifyChecksum            func(scriptPath, expectedHash string) error
	RunInstallScript          func(scriptPath string) error
}

// NewDeps returns a Deps with production implementations.
func NewDeps() *Deps {
	return &Deps{
		LookPath:           exec.LookPath,
		ExecCommand:        exec.Command,
		ExecCommandContext: exec.CommandContext,
		ExitProcess:        os.Exit,

		CreateTempAuthDir:     wrapper.CreateTempAuthDir,
		WriteTempTokenFile:    wrapper.WriteTempTokenFile,
		GenerateWrapperScript: wrapper.GenerateWrapperScript,

		GetLatestRelease:          update.GetLatestRelease,
		ConfirmUpdate:             ui.Confirm,
		DownloadToTempFile:        update.DownloadToTempFile,
		DownloadAndParseChecksums: update.DownloadAndParseChecksums,
		VerifyChecksum:            update.VerifyChecksum,
		RunInstallScript:          update.RunInstallScript,
	}
}
