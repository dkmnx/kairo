package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
)

var createTempAuthDirFn = wrapper.CreateTempAuthDir
var writeTempTokenFileFn = wrapper.WriteTempTokenFile
var generateWrapperScriptFn = wrapper.GenerateWrapperScript

type HarnessRun struct {
	AuthDir       string
	TokenPath     string
	HarnessBinary string
	CliArgs       []string
	ProviderEnv   []string
	Provider      config.Provider
	EnvVarName    string
}

func runHarnessWithWrapper(params HarnessRun) error {
	harnessPath, err := lookPath(params.HarnessBinary)
	if err != nil {
		return fmt.Errorf("'%s' command not found in PATH", params.HarnessBinary)
	}

	wrapperCfg := wrapper.ScriptConfig{
		AuthDir:    params.AuthDir,
		TokenPath:  params.TokenPath,
		CliPath:    harnessPath,
		CliArgs:    params.CliArgs,
		EnvVarName: params.EnvVarName,
	}
	wrapperScript, useCmdExe, err := generateWrapperScriptFn(wrapperCfg)
	if err != nil {
		return fmt.Errorf("generating wrapper script: %w", err)
	}

	ui.ClearScreen()
	ui.PrintBanner(kairoversion.Version, params.Provider.Model, params.Provider.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	execCmd := buildWrapperCommand(WrapperCmd{
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
	authDir, err := createTempAuthDirFn()
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

	tokenPath, err := writeTempTokenFileFn(authDir, cfg.APIKey)
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
		run.CliArgs = append(
			[]string{"--auth-type", "anthropic", "--model", cfg.Provider.Model},
			run.CliArgs...,
		)
		run.EnvVarName = "ANTHROPIC_API_KEY"

		if err := runHarnessWithWrapper(run); err != nil {
			cfg.Cmd.Printf("Error running Qwen: %v\n", err)
			exitProcess(1)
		}

		return
	}

	if err := runHarnessWithWrapper(run); err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		exitProcess(1)
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	cliArgs := cfg.HarnessArgs

	if cfg.Yolo {
		cliArgs = append([]string{yoloModeFlag(cfg.HarnessToUse)}, cliArgs...)
	}

	if cfg.HarnessToUse == harnessQwen {
		ui.PrintError("API key not found for provider")
		ui.PrintInfo("Qwen Code requires API keys to be set in environment variables.")

		return
	}

	claudePath, err := lookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)

		return
	}

	ui.ClearScreen()
	ui.PrintBanner(kairoversion.Version, cfg.Provider.Model, cfg.Provider.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	execCmd := execCommandContext(ctx, claudePath, cliArgs...)
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		exitProcess(1)
	}
}
