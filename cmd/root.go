// Package cmd implements the Kairo CLI application using the Cobra framework.
//
// Architecture:
//   - Commands are defined in individual files (root.go, setup.go, switch.go, etc.)
//   - Global state (configDir, verbose) is managed via getter/setter functions
//   - Command execution is orchestrated by rootCmd.Execute()
//
// Testing:
//   - Most commands have corresponding *_test.go files
//   - Integration tests verify end-to-end workflows
//   - External process execution can be mocked via execCommand variable
//
// Design principles:
//   - Minimal business logic in command handlers
//   - Delegation to internal packages for core functionality
//   - Consistent error handling with user-friendly messages
//
// Security:
//   - All user input is read securely using ui package
//   - No secrets are logged to stdout/stderr
//   - API keys are managed via encrypted secrets file
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/internal/wrapper"
	"github.com/dkmnx/kairo/pkg/env"
	"github.com/spf13/cobra"
)

var (
	configDir   string
	configDirMu sync.RWMutex // Protects configDir
	verbose     bool
	verboseMu   sync.RWMutex // Protects verbose
	configCache *config.ConfigCache
)

func getVerbose() bool {
	verboseMu.RLock()
	defer verboseMu.RUnlock()
	return verbose
}

func setVerbose(v bool) {
	verboseMu.Lock()
	defer verboseMu.Unlock()
	verbose = v
}

func setConfigDir(dir string) {
	configDirMu.Lock()
	defer configDirMu.Unlock()
	configDir = dir
}

func getConfigDir() string {
	configDirMu.RLock()
	defer configDirMu.RUnlock()
	if configDir != "" {
		return configDir
	}
	return env.GetConfigDir()
}

const (
	envBaseURL     = "ANTHROPIC_BASE_URL"
	envModel       = "ANTHROPIC_MODEL"
	envHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
	envSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
	envOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
	envSmallFast   = "ANTHROPIC_SMALL_FAST_MODEL"
)

var harnessFlag string

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: fmt.Sprintf(`Kairo is a CLI tool for managing Claude Code API providers with
encrypted secrets management using age encryption.

Version: %s (commit: %s, date: %s)`, kairoversion.Version, kairoversion.Commit, kairoversion.Date),
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			_ = cmd.Help()
			return
		}

		cfg, err := configCache.Get(dir)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Println("No providers configured. Run 'kairo setup' to get started.")
				return
			}
			handleConfigError(cmd, err)
			return
		}

		if len(cfg.Providers) == 0 {
			cmd.Println("No providers configured. Run 'kairo setup' to get started.")
			return
		}

		_, harnessArgs, providerName := resolveProviderAndArgs(cmd, cfg, args)
		if providerName == "" {
			return
		}

		provider, ok := cfg.Providers[providerName]
		if !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			cmd.Println("Run 'kairo list' to see configured providers")
			return
		}

		harnessToUse := getHarness(harnessFlag, cfg.DefaultHarness)
		harnessBinary := getHarnessBinary(harnessToUse)

		providerEnv, secrets, err := buildProviderEnvironment(dir, provider, providerName)
		if err != nil {
			handleSecretsError(err)
			return
		}

		apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
		if apiKey, hasKey := secrets[apiKeyKey]; hasKey {
			executeWithAuth(ExecutionConfig{
				Cmd:           cmd,
				ProviderEnv:   providerEnv,
				HarnessToUse:  harnessToUse,
				HarnessBinary: harnessBinary,
				Provider:      provider,
				HarnessArgs:   harnessArgs,
				APIKey:        apiKey,
			})
			return
		}

		executeWithoutAuth(ExecutionConfig{
			Cmd:           cmd,
			ProviderEnv:   providerEnv,
			HarnessToUse:  harnessToUse,
			HarnessBinary: harnessBinary,
			Provider:      provider,
			HarnessArgs:   harnessArgs,
		})
	},
}

// Execute runs the kairo CLI application.
// It processes command-line arguments and executes the appropriate Cobra command.
// Returns an error if command execution fails.
func Execute() error {
	args := os.Args[1:]

	// Clean up args after execution to prevent test pollution
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func init() {
	// Initialize cache with 5 minute TTL
	configCache = config.NewConfigCache(5 * time.Minute)

	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude or qwen)")
}

// handleConfigError provides user-friendly guidance for config errors.
func handleConfigError(cmd *cobra.Command, err error) {
	errStr := err.Error()

	// Check for unknown field error (outdated binary)
	// This can appear in two forms:
	// 1. Raw YAML error: "field X not found in type config.Config"
	// 2. Wrapped error: "configuration file contains field(s) not recognized"
	if (strings.Contains(errStr, "field") && strings.Contains(errStr, "not found in type")) ||
		strings.Contains(errStr, "configuration file contains field(s) not recognized") ||
		strings.Contains(errStr, "your installed kairo binary is outdated") {
		cmd.Println("Error: Your kairo binary is outdated and cannot read your configuration file.")
		cmd.Println()
		cmd.Println("The configuration file contains newer fields that this version doesn't recognize.")
		cmd.Println()
		cmd.Println("How to fix:")
		cmd.Println("  Run the installation script for your platform:")
		cmd.Println()

		// Display platform-specific installation script
		switch runtime.GOOS {
		case "windows":
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
		return
	}

	// Default error handling
	cmd.Printf("Error loading config: %v\n", err)
}

func splitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func resolveProviderAndArgs(cmd *cobra.Command, cfg *config.Config, args []string) ([]string, []string, string) {
	if len(args) == 0 {
		if cfg.DefaultProvider == "" {
			cmd.Println("No default provider set.")
			cmd.Println()
			cmd.Println("Usage:")
			cmd.Println("  kairo setup            # Configure providers")
			cmd.Println("  kairo edit <provider> # Configure a provider")
			cmd.Println("  kairo list             # List providers")
			cmd.Println("  kairo <provider>       # Use specific provider")
			return nil, nil, ""
		}
		if harnessFlag != "" {
			args = []string{cfg.DefaultProvider}
		} else {
			cmd.Printf("Default provider: %s\n", cfg.DefaultProvider)
			cmd.Println("Usage:")
			cmd.Println("  kairo <provider> [args]  # Use specific provider")
			return nil, nil, ""
		}
	}

	kairoArgs, harnessArgs := splitArgs(args)

	if len(args) > 0 && strings.HasPrefix(args[0], "-") && cfg.DefaultProvider != "" {
		args = []string{cfg.DefaultProvider}
		harnessArgs = kairoArgs
	} else if len(kairoArgs) > 0 && len(args) > 1 && kairoArgs[0] != args[0] {
		args = append([]string{args[0]}, kairoArgs...)
	} else if len(args) > 1 {
		harnessArgs = args[1:]
		args = args[:1]
	}

	providerName := args[0]

	if strings.HasPrefix(providerName, "-") {
		if cfg.DefaultProvider == "" {
			cmd.Println("Error: No default provider set and first argument looks like a flag")
			cmd.Println("Run 'kairo setup' to configure a provider")
			return nil, nil, ""
		}
		providerName = cfg.DefaultProvider
	}

	return args, harnessArgs, providerName
}

func buildProviderEnvironment(dir string, provider config.Provider, providerName string) ([]string, map[string]string, error) {
	builtInEnvVars := []string{
		fmt.Sprintf("%s=%s", envBaseURL, provider.BaseURL),
		fmt.Sprintf("%s=%s", envModel, provider.Model),
		fmt.Sprintf("%s=%s", envHaikuModel, provider.Model),
		fmt.Sprintf("%s=%s", envSonnetModel, provider.Model),
		fmt.Sprintf("%s=%s", envOpusModel, provider.Model),
		fmt.Sprintf("%s=%s", envSmallFast, provider.Model),
		"NODE_OPTIONS=--no-deprecation",
	}

	secrets, _, _, err := LoadAndDecryptSecrets(dir)
	if err != nil {
		if providers.RequiresAPIKey(providerName) {
			return nil, nil, err
		}
		secrets = make(map[string]string)
	}

	secretsEnvVars := make([]string, 0, len(secrets))
	for key, value := range secrets {
		secretsEnvVars = append(secretsEnvVars, fmt.Sprintf("%s=%s", key, value))
	}

	providerEnv := mergeEnvVars(os.Environ(), builtInEnvVars, provider.EnvVars, secretsEnvVars)
	return providerEnv, secrets, nil
}

func handleSecretsError(err error) {
	ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
	ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
	ui.PrintInfo("Use --verbose for more details.")
}

type ExecutionConfig struct {
	Cmd           *cobra.Command
	ProviderEnv   []string
	HarnessToUse  string
	HarnessBinary string
	Provider      config.Provider
	HarnessArgs   []string
	APIKey        string
}

func executeWithAuth(cfg ExecutionConfig) {
	authDir, err := wrapper.CreateTempAuthDir()
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

	tokenPath, err := wrapper.WriteTempTokenFile(authDir, cfg.APIKey)
	if err != nil {
		cfg.Cmd.Printf("Error creating secure token file: %v\n", err)
		return
	}

	cliArgs := cfg.HarnessArgs

	if cfg.HarnessToUse == "qwen" {
		cliArgs = append([]string{"--auth-type", "anthropic", "--model", cfg.Provider.Model}, cliArgs...)

		ui.ClearScreen()
		ui.PrintBanner(kairoversion.Version, cfg.Provider)

		qwenPath, err := lookPath(cfg.HarnessBinary)
		if err != nil {
			cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)
			cfg.Cmd.Printf("Please install %s CLI or use 'kairo harness set claude'\n", cfg.HarnessToUse)
			return
		}

		wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(wrapper.WrapperScriptConfig{
			AuthDir:    authDir,
			TokenPath:  tokenPath,
			CliPath:    qwenPath,
			CliArgs:    cliArgs,
			EnvVarName: "ANTHROPIC_API_KEY",
		})
		if err != nil {
			cfg.Cmd.Printf("Error generating wrapper script: %v\n", err)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		setupSignalHandler(cancel)

		var execCmd *exec.Cmd
		if useCmdExe {
			execCmd = execCommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", wrapperScript)
		} else {
			execCmd = execCommandContext(ctx, wrapperScript)
		}
		execCmd.Env = cfg.ProviderEnv
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			cfg.Cmd.Printf("Error running Qwen: %v\n", err)
			exitProcess(1)
		}
		return
	}

	claudePath, err := lookPath(cfg.HarnessBinary)
	if err != nil {
		cfg.Cmd.Printf("Error: '%s' command not found in PATH\n", cfg.HarnessBinary)
		return
	}

	wrapperScript, useCmdExe, err := wrapper.GenerateWrapperScript(wrapper.WrapperScriptConfig{
		AuthDir:   authDir,
		TokenPath: tokenPath,
		CliPath:   claudePath,
		CliArgs:   cliArgs,
	})
	if err != nil {
		cfg.Cmd.Printf("Error generating wrapper script: %v\n", err)
		return
	}

	ui.ClearScreen()
	ui.PrintBanner(kairoversion.Version, cfg.Provider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)

	var execCmd *exec.Cmd
	if useCmdExe {
		execCmd = execCommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", wrapperScript)
	} else {
		execCmd = execCommandContext(ctx, wrapperScript)
	}
	execCmd.Env = cfg.ProviderEnv
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		cfg.Cmd.Printf("Error running Claude: %v\n", err)
		exitProcess(1)
	}
}

func executeWithoutAuth(cfg ExecutionConfig) {
	cliArgs := cfg.HarnessArgs

	if cfg.HarnessToUse == "qwen" {
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
