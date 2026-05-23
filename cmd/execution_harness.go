package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
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

func qwenAuthArgs(model string) []string {
	return []string{"--auth-type", "anthropic", "--model", model}
}

func executePi(cfg ExecutionConfig) error {
	cliArgs := cfg.HarnessArgs

	if cfg.Yolo {
		flag := yoloModeFlag(cfg.HarnessToUse)
		if flag != "" {
			cliArgs = append([]string{flag}, cliArgs...)
		}
	}

	cliArgs = append(
		[]string{"--provider", cfg.ProviderName, "--model", cfg.Provider.Model},
		cliArgs...,
	)

	piPath, err := cfg.Deps.Process.LookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return nil
	}

	ui.ClearScreen()
	ui.PrintBanner(ui.Banner{
		Version:      version.Version,
		ModelName:    cfg.Provider.Model,
		ProviderName: cfg.Provider.Name,
		Harness:      cfg.HarnessToUse,
	})

	ctx, cancel := context.WithCancel(CLIContextFromCmd(cfg.Cmd).RootCtx())
	defer cancel()
	stopSignalHandler := setupSignalHandler(cancel)
	defer stopSignalHandler()

	execCmd := cfg.Deps.Process.ExecCommandContext(ctx, piPath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
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

	// Crush displays its own banner on startup; suppress Kairo's to avoid duplication.
	if params.Harness != harnessCrush {
		ui.ClearScreen()
		ui.PrintBanner(ui.Banner{
			Version:      version.Version,
			ModelName:    params.Provider.Model,
			ProviderName: params.Provider.Name,
			Harness:      params.Harness,
		})
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

func executeWithAuth(cfg ExecutionConfig) {
	if cfg.HarnessToUse == harnessPi {
		if err := executePi(cfg); err != nil {
			cfg.Cmd.Printf("Error running Pi: %v\n", err)
			cfg.Deps.Process.ExitProcess(1)
		}

		return
	}

	executeWrapperWithAuth(cfg)
}

func executeWrapperWithAuth(cfg ExecutionConfig) {
	ctx, cancel := context.WithCancel(CLIContextFromCmd(cfg.Cmd).RootCtx())
	defer cancel()
	stopSignalHandler := setupSignalHandler(cancel)
	defer stopSignalHandler()

	authDir, err := cfg.Deps.Wrapper.CreateTempAuthDir()
	if err != nil {
		cfg.Cmd.Printf("Error creating auth directory: %v\n", err)

		return
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = os.RemoveAll(authDir)
		})
	}
	defer cleanup()

	tokenPath, err := cfg.Deps.Wrapper.WriteTempTokenFile(authDir, cfg.APIKey)
	if err != nil {
		cfg.Cmd.Printf("Error creating secure token file: %v\n", err)

		return
	}

	cliArgs := cfg.HarnessArgs
	if cfg.Yolo {
		cliArgs = append([]string{yoloModeFlag(cfg.HarnessToUse)}, cliArgs...)
	}

	displayName, envVarName, extraArgs := harnessDispatch(cfg.HarnessToUse, cfg.ProviderName, cfg.Provider.Model)
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
		cfg.Cmd.Printf("Error running %s: %v\n", displayName, err)
		cfg.Deps.Process.ExitProcess(1)
	}
}

// harnessDispatch returns the display name, environment variable name, and any
// extra CLI arguments for the given harness.
func harnessDispatch(harness, providerName, model string) (displayName, envVarName string, extraArgs []string) {
	switch harness {
	case harnessQwen:
		return "Qwen", "ANTHROPIC_API_KEY", qwenAuthArgs(model)
	case harnessCrush:
		return "Crush", HarnessAPIKeyEnvVar(providerName), nil
	case harnessPi:
		return "Pi", "", nil
	default:
		return "Claude", "", nil
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	if cfg.HarnessToUse == harnessPi {
		if err := executePi(cfg); err != nil {
			cfg.Cmd.Printf("Error running Pi: %v\n", err)
			cfg.Deps.Process.ExitProcess(1)
		}

		return
	}

	cliArgs := cfg.HarnessArgs

	if cfg.Yolo {
		cliArgs = append([]string{yoloModeFlag(cfg.HarnessToUse)}, cliArgs...)
	}

	if cfg.HarnessToUse == harnessQwen {
		ui.PrintError("API key not found for provider")
		ui.PrintInfo("Qwen Code requires API keys to be set in environment variables.")

		return
	}
	// Crush prompts interactively for API keys when none are set, so it needs
	// no early-exit guard here and falls through to direct execution.

	harnessPath, err := cfg.Deps.Process.LookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return
	}

	// Crush displays its own banner on startup; suppress Kairo's to avoid duplication.
	if cfg.HarnessToUse != harnessCrush {
		ui.ClearScreen()
		ui.PrintBanner(ui.Banner{
			Version:      version.Version,
			ModelName:    cfg.Provider.Model,
			ProviderName: cfg.Provider.Name,
			Harness:      cfg.HarnessToUse,
		})
	}

	ctx, cancel := context.WithCancel(CLIContextFromCmd(cfg.Cmd).RootCtx())
	defer cancel()
	stopSignalHandler := setupSignalHandler(cancel)
	defer stopSignalHandler()

	execCmd := cfg.Deps.Process.ExecCommandContext(ctx, harnessPath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	displayName, _, _ := harnessDispatch(cfg.HarnessToUse, cfg.ProviderName, cfg.Provider.Model)

	if err := execCmd.Run(); err != nil {
		cfg.Cmd.Printf("Error running %s: %v\n", displayName, err)
		cfg.Deps.Process.ExitProcess(1)
	}
}
