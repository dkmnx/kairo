// Package cmd implements the Kairo CLI using Cobra.
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

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
	yoloFlag    bool
	verboseFlag bool
)

func setConfigDir(dir string) {
	defaultCLIContext.SetConfigDir(dir)
}

func getConfigDir() string {
	return defaultCLIContext.GetConfigDir()
}

func setVerbose(enabled bool) {
	defaultCLIContext.SetVerbose(enabled)
}

func getVerbose() bool {
	return defaultCLIContext.GetVerbose() || verboseFlag
}

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

		envResult, err := BuildProviderEnv(cliCtx, configDir, EnvProvider{
			BaseURL: provider.BaseURL,
			Model:   provider.Model,
			EnvVars: provider.EnvVars,
		}, providerName)
		if err != nil {
			handleSecretsError(err)

			return
		}

		providerEnv := envResult.ProviderEnv
		secrets := envResult.Secrets

		apiKeyKey := APIKeyEnvVarName(providerName)
		if apiKey, hasKey := secrets[apiKeyKey]; hasKey {
			executeWithAuth(ExecutionConfig{
				Cmd:           cmd,
				ProviderEnv:   providerEnv,
				HarnessToUse:  harnessToUse,
				HarnessBinary: harnessBinary,
				Provider:      provider,
				HarnessArgs:   harnessArgs,
				APIKey:        apiKey,
				Yolo:          yoloFlag,
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
			Yolo:          yoloFlag,
		})
	},
}

func Execute() error {
	args := os.Args[1:]

	defer func() {
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs(args)

	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude or qwen)")
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

func splitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}

	return args, nil
}

func getProviderFromArgs(cmd *cobra.Command, cfg *config.Config, args []string) (string, []string) {
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
