// Package cmd implements the Kairo CLI application using the Cobra framework.
//
// Architecture:
//   - Commands are defined in individual files (root.go, setup.go, default.go, etc.)
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
//
// # Global State Design
//
// This CLI uses global state for configuration directory, verbose mode, and
// root context. This pattern is appropriate for CLI applications where:
//
//  1. Single execution: Commands run once and exit - no long-running state
//  2. Test isolation: Tests use setter functions (setConfigDir, setVerbose)
//     to isolate state between test cases
//  3. Simplicity: Avoids passing context through deep call stacks
//
// Thread safety is ensured via sync.RWMutex for configDir and verbose.
// The rootCtx is initialized lazily and never modified after creation.
//
// Alternative approaches considered:
//   - Dependency injection: Would require significant refactoring for minimal benefit
//   - Context values: Would require threading context through all functions
//
// This design prioritizes CLI simplicity over general-purpose library use.
package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var (
	configDir   string
	configDirMu sync.RWMutex // Protects configDir
	verbose     bool
	verboseMu   sync.RWMutex // Protects verbose
)

func getVerbose() bool {
	verboseMu.RLock()
	defer verboseMu.RUnlock()

	return verbose
}

func setVerbose(enabled bool) {
	verboseMu.Lock()
	defer verboseMu.Unlock()

	verbose = enabled
}

func setConfigDir(dir string) {
	configDirMu.Lock()
	defer configDirMu.Unlock()

	configDir = dir
	defaultCLIContext.SetConfigDir(dir)
}

func getConfigDir() string {
	configDirMu.RLock()
	defer configDirMu.RUnlock()

	if configDir != "" {
		return configDir
	}

	dir, err := config.GetConfigDir()
	if err != nil {
		return ""
	}

	return dir
}

const (
	envBaseURL     = "ANTHROPIC_BASE_URL"
	envModel       = "ANTHROPIC_MODEL"
	envHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
	envSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
	envOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
	envSmallFast   = "ANTHROPIC_SMALL_FAST_MODEL"

	configCacheTTL = 5 * time.Minute
)

var (
	harnessFlag string
	yoloFlag    bool // yolo mode - skips permission prompts
)

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: fmt.Sprintf(`Kairo is a CLI tool for managing Claude Code API providers with
encrypted secrets management using age encryption.

Version: %s (commit: %s, date: %s)`, kairoversion.Version, kairoversion.Commit, kairoversion.Date),
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cliCtx := GetCLIContext(cmd)
		configDir := cliCtx.GetConfigDir()
		if configDir == "" {
			cmd.Println("Error: config directory not found")
			_ = cmd.Help()

			return
		}

		cfg, err := cliCtx.GetConfigCache().Get(cliCtx.GetRootCtx(), configDir)
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

		providerEnv, secrets, err := buildProviderEnvironment(cliCtx, configDir, provider, providerName)
		if err != nil {
			handleSecretsError(err)

			return
		}

		apiKeyKey := apiKeyEnvVarName(providerName)
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
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude or qwen)")
	rootCmd.Flags().BoolVarP(&yoloFlag, "yolo", "y", false, "Skip permission prompts (--dangerously-skip-permissions for Claude, --yolo for Qwen)")

	// Sync flag values to defaultCLIContext before each command runs.
	// This ensures that even though flags are bound to globals, the CLIContext
	// used by commands is kept in sync.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		defaultCLIContext.SetConfigDir(configDir)
		defaultCLIContext.SetVerbose(verbose)
		// Set the CLIContext on the command for GetCLIContext to find
		cmd.SetContext(WithCLIContext(cmd.Context(), defaultCLIContext))
	}
}

func splitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}

	return args, nil
}

// getProviderFromArgs extracts provider name and remaining args from command args
func getProviderFromArgs(cmd *cobra.Command, cfg *config.Config, args []string) (string, []string) {
	kairoArgs, harnessArgs := splitArgs(args)

	// Determine argument handling strategy based on input pattern
	switch {
	case len(args) > 0 && strings.HasPrefix(args[0], "-") && cfg.DefaultProvider != "":
		// First arg looks like a flag and default provider is set - use default
		args = []string{cfg.DefaultProvider}
		harnessArgs = kairoArgs
	case len(kairoArgs) > 0 && len(args) > 1 && kairoArgs[0] != args[0]:
		// Kairo args differ from original args - prepend first arg
		args = append([]string{args[0]}, kairoArgs...)
	case len(args) > 1:
		// Multiple args - first is provider, rest are harness args
		harnessArgs = args[1:]
		args = args[:1]
	}

	providerName := args[0]

	if strings.HasPrefix(providerName, "-") {
		if cfg.DefaultProvider == "" {
			cmd.Println("Error: No default provider set and first argument looks like a flag")
			cmd.Println("Run 'kairo setup' to configure a provider")

			return "", nil
		}
		providerName = cfg.DefaultProvider
	}

	return providerName, harnessArgs
}

// resolveProviderAndArgs determines which provider to use and separates harness arguments.
// Returns: (kairoArgs, harnessArgs, providerName)
func resolveProviderAndArgs(cmd *cobra.Command, cfg *config.Config, args []string) ([]string, []string, string) {
	if len(args) == 0 {
		if cfg.DefaultProvider == "" {
			cmd.Println("No default provider set.")
			cmd.Println()
			cmd.Println("Usage:")
			cmd.Println("  kairo setup            # Configure providers")
			cmd.Println("  kairo default <name>   # Set default provider")
			cmd.Println("  kairo list             # List providers")
			cmd.Println("  kairo <provider>       # Use specific provider")

			return nil, nil, ""
		}
		args = []string{cfg.DefaultProvider}
	}

	providerName, harnessArgs := getProviderFromArgs(cmd, cfg, args)

	return args, harnessArgs, providerName
}
