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

	piPath, err := cfg.Deps.LookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return nil
	}

	ui.ClearScreen()
	ui.PrintBanner(version.Version, cfg.Provider.Model, cfg.Provider.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	execCmd := cfg.Deps.ExecCommandContext(ctx, piPath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

func runHarnessWithWrapper(deps *Deps, params HarnessRun) error {
	harnessPath, err := deps.LookPath(params.HarnessBinary)
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
	wrapperScript, useCmdExe, err := deps.GenerateWrapperScript(wrapperCfg)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.RuntimeError,
			"generating wrapper script", err)
	}

	ui.ClearScreen()
	ui.PrintBanner(version.Version, params.Provider.Model, params.Provider.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

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
			cfg.Deps.ExitProcess(1)
		}

		return
	}

	executeWrapperWithAuth(cfg)
}

func executeWrapperWithAuth(cfg ExecutionConfig) {
	authDir, err := cfg.Deps.CreateTempAuthDir()
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

	tokenPath, err := cfg.Deps.WriteTempTokenFile(authDir, cfg.APIKey)
	if err != nil {
		cfg.Cmd.Printf("Error creating secure token file: %v\n", err)

		return
	}

	cliArgs := cfg.HarnessArgs
	run := HarnessRun{
		AuthDir:       authDir,
		TokenPath:     tokenPath,
		HarnessBinary: cfg.HarnessBinary,
		CliArgs:       cliArgs,
		ProviderEnv:   cfg.ProviderEnv,
		Provider:      cfg.Provider,
	}

	if cfg.Yolo {
		run.CliArgs = append([]string{yoloModeFlag(cfg.HarnessToUse)}, run.CliArgs...)
	}

	if cfg.HarnessToUse == harnessQwen {
		run.CliArgs = append(qwenAuthArgs(cfg.Provider.Model), run.CliArgs...)
		run.EnvVarName = "ANTHROPIC_API_KEY"

		if err := runHarnessWithWrapper(cfg.Deps, run); err != nil {
			cfg.Cmd.Printf("Error running Qwen: %v\n", err)
			cfg.Deps.ExitProcess(1)
		}

		return
	}

	if err := runHarnessWithWrapper(cfg.Deps, run); err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		cfg.Deps.ExitProcess(1)
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	if cfg.HarnessToUse == harnessPi {
		if err := executePi(cfg); err != nil {
			cfg.Cmd.Printf("Error running Pi: %v\n", err)
			cfg.Deps.ExitProcess(1)
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

	claudePath, err := cfg.Deps.LookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return
	}

	ui.ClearScreen()
	ui.PrintBanner(version.Version, cfg.Provider.Model, cfg.Provider.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	execCmd := cfg.Deps.ExecCommandContext(ctx, claudePath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		cfg.Deps.ExitProcess(1)
	}
}
