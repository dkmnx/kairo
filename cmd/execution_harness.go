package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/execution"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
)

// HarnessRun holds the state for a single harness execution.
type HarnessRun struct {
	AuthDir       string
	TokenPath     string
	HarnessBinary string
	CliArgs       []string
	ProviderEnv   []string
	Provider      config.Provider
	EnvVarName    string
	Harness       string
}

// runHarnessExec is the shared harness-execution primitive. It locates the
// binary in PATH, prints the Kairo banner (skipped for Crush which has its
// own), starts a signal-aware session, and runs the binary with the standard
// stdin/stdout/stderr wiring. On error it returns the error so the caller can
// decide whether to exit or recover.
func runHarnessExec(cfg ExecutionConfig, harnessPath string, cliArgs []string) error {
	if cfg.HarnessToUse != harness.Crush {
		ui.ClearScreen()
		ui.PrintBanner(ui.Banner{
			Version:      version.Version,
			ModelName:    cfg.Provider.Model,
			ProviderName: cfg.Provider.Name,
			Harness:      cfg.HarnessToUse,
		})
	}

	rootCtx := context.Background()
	if cliCtx := CLIContextFromCmd(cfg.Cmd); cliCtx != nil {
		rootCtx = cliCtx.RootCtx()
	}

	ctx, cancel, stopSig := execution.StartSession(rootCtx)
	defer cancel()
	defer stopSig()

	execCmd := cfg.Deps.Process.ExecCommandContext(ctx, harnessPath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// reportHarnessError prints a uniform harness-error line and exits the
// process. It is the standard post-exec failure path.
func reportHarnessError(cfg ExecutionConfig, displayName string, err error) {
	cfg.Cmd.Printf("Error running %s: %v\n", displayName, err)
	cfg.Deps.Process.ExitProcess(1)
}

// lookUpHarnessBinary resolves the binary in PATH. On miss it prints an error
// via the cobra command and returns the empty string so callers can early-out
// without printing a second time.
func lookUpHarnessBinary(cfg ExecutionConfig) string {
	path, err := cfg.Deps.Process.LookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return ""
	}

	return path
}

// executePi handles the Pi harness, which injects --provider/--model and runs
// directly without the wrapper script. It returns the run error so callers
// can decide whether to surface it and exit.
func executePi(cfg ExecutionConfig) error {
	cliArgs := applyYoloFlag(cfg, cfg.HarnessArgs)
	cliArgs = append(
		[]string{"--provider", cfg.ProviderName, "--model", cfg.Provider.Model},
		cliArgs...,
	)

	piPath := lookUpHarnessBinary(cfg)
	if piPath == "" {
		return nil
	}

	return runHarnessExec(cfg, piPath, cliArgs)
}

func runHarnessWithWrapper(ctx context.Context, deps *Deps, params HarnessRun) error {
	harnessPath, err := deps.Process.LookPath(params.HarnessBinary)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			fmt.Sprintf("'%s' command not found in PATH", params.HarnessBinary), err)
	}

	wrapperCfg := wrapper.ScriptConfig{
		AuthDir:    params.AuthDir,
		TokenPath:  params.TokenPath,
		CliPath:    harnessPath,
		CliArgs:    params.CliArgs,
		EnvVarName: params.EnvVarName,
	}
	wrapperScript, useCmdExe, err := deps.Wrapper.GenerateWrapperScript(wrapperCfg)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			"generating wrapper script", err)
	}

	execCmd := buildWrapperCommand(deps, WrapperCmd{
		Ctx:           ctx,
		WrapperScript: wrapperScript,
		IsWindows:     useCmdExe,
	})
	execCmd.Env = params.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// applyYoloFlag prepends the yolo flag to cliArgs when cfg.Yolo is set.
func applyYoloFlag(cfg ExecutionConfig, cliArgs []string) []string {
	if cfg.Yolo {
		if flag := harness.YoloFlag(cfg.HarnessToUse); flag != "" {
			return append([]string{flag}, cliArgs...)
		}
	}

	return cliArgs
}

// handlePi returns true if the execution was handled by the Pi harness.
func handlePi(cfg ExecutionConfig) bool {
	if cfg.HarnessToUse != harness.Pi {
		return false
	}

	if err := executePi(cfg); err != nil {
		reportHarnessError(cfg, "Pi", err)
	}

	return true
}

func executeWithAuth(cfg ExecutionConfig) {
	if handlePi(cfg) {
		return
	}

	executeWrapperWithAuth(cfg)
}

func executeWrapperWithAuth(cfg ExecutionConfig) {
	rootCtx := context.Background()
	if cliCtx := CLIContextFromCmd(cfg.Cmd); cliCtx != nil {
		rootCtx = cliCtx.RootCtx()
	}
	ctx, cancel, stopSig := execution.StartSession(rootCtx)
	defer cancel()
	defer stopSig()

	authDir, err := cfg.Deps.Wrapper.CreateTempAuthDir()
	if err != nil {
		cfg.Cmd.Printf("Error creating auth directory: %v\n", err)

		return
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if err := os.RemoveAll(authDir); err != nil {
				cfg.Cmd.Printf("Error cleaning up auth directory: %v\n", err)
			}
		})
	}
	defer cleanup()

	tokenPath, err := cfg.Deps.Wrapper.WriteTempTokenFile(authDir, cfg.APIKey)
	if err != nil {
		cfg.Cmd.Printf("Error creating secure token file: %v\n", err)

		return
	}

	cliArgs := applyYoloFlag(cfg, cfg.HarnessArgs)

	displayName, envVarName, extraArgs := harness.Dispatch(cfg.HarnessToUse, cfg.ProviderName, cfg.Provider.Model)
	cliArgs = append(extraArgs, cliArgs...)

	run := HarnessRun{
		AuthDir:       authDir,
		TokenPath:     tokenPath,
		HarnessBinary: cfg.HarnessBinary,
		CliArgs:       cliArgs,
		ProviderEnv:   cfg.ProviderEnv,
		Provider:      cfg.Provider,
		EnvVarName:    envVarName,
		Harness:       cfg.HarnessToUse,
	}

	if err := runHarnessWithWrapper(ctx, cfg.Deps, run); err != nil {
		reportHarnessError(cfg, displayName, err)
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	if handlePi(cfg) {
		return
	}

	cliArgs := applyYoloFlag(cfg, cfg.HarnessArgs)

	if cfg.HarnessToUse == harness.Qwen {
		ui.PrintError("API key not found for provider")
		ui.PrintInfo("Qwen Code requires API keys to be set in environment variables.")

		return
	}
	// Crush prompts interactively for API keys when none are set, so it needs
	// no early-exit guard here and falls through to direct execution.

	harnessPath := lookUpHarnessBinary(cfg)
	if harnessPath == "" {
		return
	}

	displayName, _, _ := harness.Dispatch(cfg.HarnessToUse, cfg.ProviderName, cfg.Provider.Model)
	if err := runHarnessExec(cfg, harnessPath, cliArgs); err != nil {
		reportHarnessError(cfg, displayName, err)
	}
}
