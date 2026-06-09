// Package cmd implements the kairo CLI command tree using Cobra.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var (
	harnessFlag         string
	skipPermissionsFlag bool
	verboseFlag         bool
)

// verbose reports whether verbose output should be emitted. It reads from the
// CLIContext in the command's context (set by Execute) and falls back to the
// --verbose flag.
func verbose(cmd *cobra.Command) bool {
	cliCtx := CLIContextFromCmd(cmd)
	if cliCtx != nil && cliCtx.Verbose() {
		return true
	}

	return verboseFlag
}

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: fmt.Sprintf(`Kairo is a CLI tool for managing Claude Code API providers with
encrypted secrets management using age encryption.

Version: %s (commit: %s, date: %s)`, version.Version, version.Commit, version.Date),
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		OrchestrateExecution(cmd, args)
	},
}

// Execute runs the root command.
func Execute() error {
	cliCtx := NewCLIContext()
	promptRootCtx = cliCtx.RootCtx()

	args := os.Args[1:]
	cliCtx.SetDefaultProviderExplicit(hasArgsSeparator(args))

	rootCmd.SetArgs(args)

	// Inject the CLIContext into the root command so PersistentPreRun and all
	// subcommands can retrieve it via CLIContextFromCmd.
	ctx := rootCmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	rootCmd.SetContext(WithCLIContext(ctx, cliCtx))

	defer func() {
		rootCmd.SetArgs(nil)
	}()

	return rootCmd.Execute()
}

// SetArgs overrides os.Args for the next Execute call. Production code never
// calls this; tests use it to inject a deterministic argv without polluting
// the global os.Args.
func SetArgs(args ...string) {
	os.Args = append([]string{os.Args[0]}, args...)
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().StringVar(&harnessFlag, "harness", "", "CLI harness to use (claude, qwen, pi, or crush)")
	rootCmd.Flags().BoolVarP(&skipPermissionsFlag, "yolo", "y", false,
		"Skip permission prompts (--dangerously-skip-permissions for Claude, --yolo for Qwen)")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cliCtx := CLIContextFromCmd(cmd)
		if cliCtx == nil {
			cliCtx = NewCLIContext()
			cmd.SetContext(WithCLIContext(cmd.Context(), cliCtx))
		}

		if configFlag, err := cmd.Flags().GetString("config"); err == nil && configFlag != "" {
			cliCtx.SetConfigDir(configFlag)
		}
		cliCtx.SetVerbose(verboseFlag)
	}
}

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
		piEnvVar, ok := providers.APIKeyEnvVarFor(pName)
		if !ok {
			if p.EnvKey == "" {
				piEnvVar = harness.APIKeyEnvVar(pName)
			} else {
				piEnvVar = p.EnvKey
			}
		}
		val, found := lookupAPIKeyWithFallback(secrets, pName)
		if found {
			providerEnv = append(providerEnv, fmt.Sprintf("%s=%s", piEnvVar, val))
			hasAnyKey = true
		}
	}

	execCfg := buildExecutionConfig(cmd, cliCtx, providerEnv, provider, providerName, harnessToUse, harnessArgs, "")

	if hasAnyKey {
		executeWithAuth(execCfg)
	} else {
		executeWithoutAuth(execCfg)
	}
}

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

	apiKey, hasKey := lookupAPIKeyWithFallback(envResult.Secrets, providerName)

	execCfg := buildExecutionConfig(
		cmd, cliCtx, envResult.ProviderEnv, provider,
		providerName, harnessToUse, harnessArgs, apiKey,
	)

	if hasKey {
		executeWithAuth(execCfg)
	} else {
		executeWithoutAuth(execCfg)
	}
}

func lookupAPIKeyWithFallback(secrets map[string]string, providerName string) (string, bool) {
	if val, ok := secrets[harness.APIKeyEnvVar(providerName)]; ok {
		return val, true
	}

	if providerName != customProviderName {
		if val, ok := secrets[harness.APIKeyEnvVar(customProviderName)]; ok {
			return val, true
		}
	}

	return "", false
}

func buildExecutionConfig(
	cmd *cobra.Command,
	cliCtx *CLIContext,
	providerEnv []string,
	provider config.Provider,
	providerName, harnessToUse string,
	harnessArgs []string,
	apiKey string,
) ExecutionConfig {
	return ExecutionConfig{
		Cmd:           cmd,
		ProviderEnv:   providerEnv,
		HarnessToUse:  harnessToUse,
		HarnessBinary: harnessToUse,
		Provider:      provider,
		ProviderName:  providerName,
		HarnessArgs:   harnessArgs,
		APIKey:        apiKey,
		Yolo:          skipPermissionsFlag,
		Deps:          cliCtx.Deps(),
	}
}

func resolveAPIKey(secrets map[string]string, providerName string) (string, bool) {
	return lookupAPIKeyWithFallback(secrets, providerName)
}
