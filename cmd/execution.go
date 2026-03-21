package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
	"github.com/spf13/cobra"
)

// harnessQwen is the harness name for Qwen Code.
const harnessQwen = "qwen"

// createTempAuthDirFn wraps wrapper.CreateTempAuthDir for testability.
var createTempAuthDirFn = wrapper.CreateTempAuthDir

// writeTempTokenFileFn wraps wrapper.WriteTempTokenFile for testability.
var writeTempTokenFileFn = wrapper.WriteTempTokenFile

// handleConfigError provides user-friendly guidance for config errors.
func handleConfigError(cmd *cobra.Command, err error) {
	errStr := err.Error()

	// Check for unknown field error (outdated binary)
	// This can appear in two forms:
	// 1. Raw YAML error: "field X not found in type config.Config"
	// 2. Wrapped error: "configuration file contains field(s) not recognized"
	if isOutdatedBinaryError(errStr) {
		promptUpgrade(cmd, err)

		return
	}

	// Default error handling
	cmd.Printf("Error loading config: %v\n", err)
}

// isOutdatedBinaryError checks if the error indicates an outdated binary.
func isOutdatedBinaryError(errStr string) bool {
	return (strings.Contains(errStr, "field") && strings.Contains(errStr, "not found in type")) ||
		strings.Contains(errStr, "configuration file contains field(s) not recognized") ||
		strings.Contains(errStr, "your installed kairo binary is outdated")
}

// promptUpgrade provides upgrade instructions for outdated binaries.
func promptUpgrade(cmd *cobra.Command, err error) {
	cmd.Println("Error: Your kairo binary is outdated and cannot read your configuration file.")
	cmd.Println()
	cmd.Println("The configuration file contains newer fields that this version doesn't recognize.")
	cmd.Println()
	cmd.Println("How to fix:")
	cmd.Println("  Run the installation script for your platform:")
	cmd.Println()

	switch runtime.GOOS {
	case config.WindowsGOOS:
		cmd.Println("    irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex")
	default: // linux, darwin (macOS)
		cmd.Println("    curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh")
	}

	cmd.Println()
	cmd.Println("  For manual installation, see:")
	cmd.Println("    https://github.com/dkmnx/kairo/blob/main/docs/guides/user-guide.md#manual-installation")
	cmd.Println()
	if verbose {
		cmd.Printf("Technical details: %v\n", err)
	}
}

// buildProviderEnvironment builds the environment variables for a provider.
// Returns: (providerEnv, secrets, error)
//
// SECURITY: Provider environment does NOT include any secrets from the encrypted
// secrets file. Secrets are returned separately and should only be passed via the
// secure wrapper script mechanism to avoid exposing credentials in child process
// environments (/proc/<pid>/environ).
func buildProviderEnvironment(
	cliCtx *CLIContext,
	configDir string,
	provider config.Provider,
	providerName string,
) ([]string, map[string]string, error) {
	builtInEnvVars := buildBuiltInEnvVars(provider)

	secrets, _, _, err := LoadAndDecryptSecrets(cliCtx.GetRootCtx(), configDir)
	if err != nil {
		if providers.RequiresAPIKey(providerName) {
			return nil, nil, err
		}
		secrets = make(map[string]string)
	}

	// SECURITY: Do NOT include secrets in provider environment.
	// Secrets are returned separately for secure injection via wrapper script.
	// Only include built-in vars (from provider config) and non-secret provider EnvVars.
	providerEnv := mergeEnvVars(os.Environ(), builtInEnvVars, provider.EnvVars)

	return providerEnv, secrets, nil
}

// buildBuiltInEnvVars creates environment variables from provider configuration.
func buildBuiltInEnvVars(provider config.Provider) []string {
	return []string{
		fmt.Sprintf("%s=%s", envBaseURL, provider.BaseURL),
		fmt.Sprintf("%s=%s", envModel, provider.Model),
		fmt.Sprintf("%s=%s", envHaikuModel, provider.Model),
		fmt.Sprintf("%s=%s", envSonnetModel, provider.Model),
		fmt.Sprintf("%s=%s", envOpusModel, provider.Model),
		fmt.Sprintf("%s=%s", envSmallFast, provider.Model),
		"NODE_OPTIONS=--no-deprecation",
	}
}

// buildSecretsEnvVars converts secrets map to environment variable strings.
func buildSecretsEnvVars(secrets map[string]string) []string {
	secretsEnvVars := make([]string, 0, len(secrets))
	for key, value := range secrets {
		secretsEnvVars = append(secretsEnvVars, fmt.Sprintf("%s=%s", key, value))
	}

	return secretsEnvVars
}

// apiKeyEnvVarName formats the environment variable name for a provider's API key.
func apiKeyEnvVarName(providerName string) string {
	return fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
}

func handleSecretsError(err error) {
	ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
	ui.PrintInfo("Restore 'age.key' and 'secrets.age' from backup,")
	ui.PrintInfo("or remove both files and run 'kairo setup --reset-secrets' to re-enter API keys.")
	ui.PrintInfo("Use --verbose for more details.")
}

// ExecutionConfig holds configuration for executing a harness CLI.
type ExecutionConfig struct {
	Cmd           *cobra.Command
	ProviderEnv   []string
	HarnessToUse  string
	HarnessBinary string
	Provider      config.Provider
	HarnessArgs   []string
	APIKey        string
	Yolo          bool // skip permission prompts
}

// HarnessWrapperParams holds all parameters for running a harness with auth wrapper.
type HarnessWrapperParams struct {
	AuthDir       string
	TokenPath     string
	HarnessBinary string
	CliArgs       []string
	ProviderEnv   []string
	Provider      config.Provider
	EnvVarName    string // Optional: set for Qwen compatibility
}

// BuildWrapperCommandParams holds parameters for building a wrapper command.
type BuildWrapperCommandParams struct {
	Ctx           context.Context
	WrapperScript string
	IsWindows     bool
}

// runHarnessWithWrapper executes a harness CLI using an auth wrapper script.
// Handles wrapper script generation, context setup, signal handling, and command execution.
func runHarnessWithWrapper(params HarnessWrapperParams) error {
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
	wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(wrapperCfg)
	if err != nil {
		return fmt.Errorf("generating wrapper script: %w", err)
	}

	ui.ClearScreen()
	ui.PrintBanner(kairoversion.Version, params.Provider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	execCmd := buildWrapperCommand(BuildWrapperCommandParams{
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

// buildWrapperCommand creates the appropriate exec.Cmd for the wrapper script.
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
	wrapperParams := HarnessWrapperParams{
		AuthDir:       authDir,
		TokenPath:     tokenPath,
		HarnessBinary: cfg.HarnessBinary,
		CliArgs:       cliArgs,
		ProviderEnv:   cfg.ProviderEnv,
		Provider:      cfg.Provider,
	}

	if cfg.HarnessToUse == harnessQwen {
		wrapperParams.CliArgs = append(
			[]string{"--auth-type", "anthropic", "--model", cfg.Provider.Model},
			wrapperParams.CliArgs...,
		)
		wrapperParams.EnvVarName = "ANTHROPIC_API_KEY"

		err = runHarnessWithWrapper(wrapperParams)
		if err != nil {
			cfg.Cmd.Printf("Error running Qwen: %v\n", err)
			exitProcess(1)
		}

		return
	}

	err = runHarnessWithWrapper(wrapperParams)
	if err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		exitProcess(1)
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	cliArgs := cfg.HarnessArgs

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
	ui.PrintBanner(kairoversion.Version, cfg.Provider)

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
