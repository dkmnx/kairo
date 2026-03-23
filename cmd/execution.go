package cmd

import (
	"context"
	"os/exec"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

// ExecutionConfig holds configuration for executing a harness CLI.
type ExecutionConfig struct {
	Cmd           *cobra.Command
	ProviderEnv   []string
	HarnessToUse  string
	HarnessBinary string
	Provider      config.Provider
	HarnessArgs   []string
	APIKey        string
	Yolo          bool
}

type BuildWrapperCommandParams struct {
	Ctx           context.Context
	WrapperScript string
	IsWindows     bool
}

func buildWrapperCommand(params BuildWrapperCommandParams) *exec.Cmd {
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
