package cmd

import (
	"context"
	"os/exec"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

// ExecutionConfig holds all parameters needed to execute a CLI harness.
type ExecutionConfig struct {
	Cmd           *cobra.Command
	ProviderEnv   []string
	HarnessToUse  string
	HarnessBinary string
	Provider      config.Provider
	ProviderName  string
	HarnessArgs   []string
	APIKey        string
	Yolo          bool
}

// WrapperCmd holds parameters for building a wrapper shell command.
type WrapperCmd struct {
	Ctx           context.Context
	WrapperScript string
	IsWindows     bool
}

func buildWrapperCommand(params WrapperCmd) *exec.Cmd {
	if params.IsWindows {
		return execCommandContext(
			params.Ctx,
			"powershell",
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-File",
			params.WrapperScript,
		)
	}

	return execCommandContext(params.Ctx, params.WrapperScript)
}
