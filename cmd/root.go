// Package cmd implements the kairo CLI command tree using Cobra.
package cmd

import (
	stderrors "errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var (
	harnessFlag string
	yoloFlag    bool
	verboseFlag bool
)

func setConfigDir(dir string) {
	defaultCLIContext.SetConfigDir(dir)
}

func configDir() string {
	return defaultCLIContext.ConfigDir()
}

func setVerbose(enabled bool) {
	defaultCLIContext.SetVerbose(enabled)
}

func verbose() bool {
	return defaultCLIContext.Verbose() || verboseFlag
}

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: fmt.Sprintf(`Kairo is a CLI tool for managing Claude Code API providers with
encrypted secrets management using age encryption.

Version: %s (commit: %s, date: %s)`, version.Version, version.Commit, version.Date),
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cliCtx := CLIContextFromCmd(cmd)

		cfg, ok := loadRootConfig(cmd, cliCtx)
		if !ok {
			return
		}

		_, harnessArgs, providerName := resolveProviderAndArgs(cmd, cfg, args)
		if providerName == "" {
			return
		}

		provider, ok := lookupProvider(cmd, cfg, providerName)
		if !ok {
			return
		}

		harnessToUse := resolveHarness(harnessFlag, cfg.DefaultHarness)

		if harnessToUse == harnessPi {
			runPiProvider(cmd, cliCtx, cfg, provider, providerName, harnessToUse, harnessArgs)
		} else {
			runStandardProvider(cmd, cliCtx, provider, providerName, harnessToUse, harnessArgs)
		}
	},
}

// Execute runs the root command.
func Execute() error {
	args := os.Args[1:]

	defer func() {
		rootCmd.SetArgs(nil)
	}()

	defaultCLIContext.SetDefaultProviderExplicit(hasDoubleDash(args))

	rootCmd.SetArgs(args)

	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude, qwen, pi, or crush)")
	rootCmd.Flags().BoolVarP(&yoloFlag, "yolo", "y", false,
		"Skip permission prompts (--dangerously-skip-permissions for Claude, --yolo for Qwen)")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if configFlag, err := cmd.Flags().GetString("config"); err == nil && configFlag != "" {
			defaultCLIContext.SetConfigDir(configFlag)
		}
		defaultCLIContext.SetVerbose(verboseFlag)
		cmd.SetContext(WithCLIContext(cmd.Context(), defaultCLIContext))
	}
}

// loadRootConfig loads and validates the configuration. Returns nil config on error
// after printing an appropriate message to cmd.
func loadRootConfig(cmd *cobra.Command, cliCtx *CLIContext) (*config.Config, bool) {
	configDir := cliCtx.ConfigDir()
	if configDir == "" {
		cmd.Println("Error: config directory not found")
		if err := cmd.Help(); err != nil {
			cmd.Println(err)
		}

		return nil, false
	}

	cfg, err := cliCtx.ConfigCache().Get(cliCtx.RootCtx(), configDir)
	if err != nil {
		if stderrors.Is(err, fs.ErrNotExist) {
			cmd.Println("No providers configured. Run 'kairo setup' to get started.")

			return nil, false
		}
		handleConfigError(cmd, err)

		return nil, false
	}

	if len(cfg.Providers) == 0 {
		cmd.Println("No providers configured. Run 'kairo setup' to get started.")

		return nil, false
	}

	return cfg, true
}

// lookupProvider finds the named provider in the configuration.
// Prints an error and returns false if not found.
func lookupProvider(cmd *cobra.Command, cfg *config.Config, providerName string) (config.Provider, bool) {
	provider, ok := cfg.Providers[providerName]
	if !ok {
		cmd.Printf("Error: provider '%s' not configured\n", providerName)
		cmd.Println("Run 'kairo list' to see configured providers")

		return config.Provider{}, false
	}

	return provider, true
}

// runPiProvider handles the Pi harness execution path, which injects all provider
// API keys as environment variables rather than using a single API key.
func runPiProvider(
	cmd *cobra.Command,
	cliCtx *CLIContext,
	cfg *config.Config,
	provider config.Provider,
	providerName, harnessToUse string,
	harnessArgs []string,
) {
	envResult, err := BuildProviderEnv(cliCtx, cliCtx.ConfigDir(), provider, providerName)
	if err != nil {
		handleSecretsError(err)

		return
	}

	providerEnv := envResult.ProviderEnv
	secrets := envResult.Secrets

	hasAnyKey := false
	for pName, p := range cfg.Providers {
		piEnvVar, ok := PiAPIKeyEnvVar(pName)
		if !ok {
			if p.EnvKey == "" {
				piEnvVar = APIKeyEnvVarName(pName)
			} else {
				piEnvVar = p.EnvKey
			}
		}
		key := APIKeyEnvVarName(pName)
		val, found := secrets[key]
		if !found && pName != customProviderName {
			key = APIKeyEnvVarName(customProviderName)
			val, found = secrets[key]
		}
		if found {
			providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", piEnvVar, val))
			hasAnyKey = true
		}
	}

	execCfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   providerEnv,
		HarnessToUse:  harnessToUse,
		HarnessBinary: harnessBinary(harnessToUse),
		Provider:      provider,
		ProviderName:  providerName,
		HarnessArgs:   harnessArgs,
		Yolo:          yoloFlag,
		Deps:          cliCtx.Deps(),
	}

	if hasAnyKey {
		executeWithAuth(execCfg)
	} else {
		executeWithoutAuth(execCfg)
	}
}

// runStandardProvider handles the Claude/Qwen harness execution path with
// single-provider API key lookup and fallback to the custom provider key.
func runStandardProvider(
	cmd *cobra.Command,
	cliCtx *CLIContext,
	provider config.Provider,
	providerName, harnessToUse string,
	harnessArgs []string,
) {
	envResult, err := BuildProviderEnv(cliCtx, cliCtx.ConfigDir(), provider, providerName)
	if err != nil {
		handleSecretsError(err)

		return
	}

	apiKey, hasKey := resolveAPIKey(envResult.Secrets, providerName)

	execCfg := ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   envResult.ProviderEnv,
		HarnessToUse:  harnessToUse,
		HarnessBinary: harnessBinary(harnessToUse),
		Provider:      provider,
		ProviderName:  providerName,
		HarnessArgs:   harnessArgs,
		APIKey:        apiKey,
		Yolo:          yoloFlag,
		Deps:          cliCtx.Deps(),
	}

	if hasKey {
		executeWithAuth(execCfg)
	} else {
		executeWithoutAuth(execCfg)
	}
}

// resolveAPIKey looks up the API key for the named provider, falling back to
// the custom provider key if the provider-specific key is not found.
func resolveAPIKey(secrets map[string]string, providerName string) (string, bool) {
	if key, ok := secrets[APIKeyEnvVarName(providerName)]; ok {
		return key, true
	}

	if key, ok := secrets[APIKeyEnvVarName(customProviderName)]; ok {
		return key, true
	}

	return "", false
}

func splitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}

	return args, nil
}

func hasDoubleDash(args []string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			return true
		}
		if args[i] == "-" || !strings.HasPrefix(args[i], "-") {
			return false
		}
		if strings.Contains(args[i], "=") {
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}

	return false
}

func providerFromArgs(cmd *cobra.Command, cfg *config.Config, args []string) (string, []string) {
	kairoArgs, harnessArgs := splitArgs(args)

	switch {
	case len(args) > 0 && strings.HasPrefix(args[0], "-") && cfg.DefaultProvider != "":
		args = []string{cfg.DefaultProvider}
		harnessArgs = kairoArgs
	case len(kairoArgs) > 0 && len(args) > 1 && kairoArgs[0] != args[0]:
		args = append([]string{args[0]}, kairoArgs...)
	case len(args) > 1:
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

func resolveProviderAndArgs(cmd *cobra.Command, cfg *config.Config, args []string) ([]string, []string, string) {
	cliCtx := CLIContextFromCmd(cmd)

	if len(args) == 0 || cliCtx.DefaultProviderExplicit() || harnessFlag != "" {
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

		return []string{cfg.DefaultProvider}, args, cfg.DefaultProvider
	}

	providerName, harnessArgs := providerFromArgs(cmd, cfg, args)

	return args, harnessArgs, providerName
}
