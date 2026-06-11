package cmd

import (
	stderrors "errors"
	"io/fs"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/spf13/cobra"
)

// OrchestrateExecution runs the full execution pipeline: load config, resolve
// provider/harness, and dispatch to the appropriate execution path.
func OrchestrateExecution(cmd *cobra.Command, args []string) {
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

	if harnessToUse == harness.Pi {
		runPiProvider(cmd, cliCtx, cfg, provider, providerName, harnessToUse, harnessArgs)
	} else {
		runStandardProvider(cmd, cliCtx, provider, providerName, harnessToUse, harnessArgs)
	}
}

// loadRootConfig loads and validates the configuration. Returns nil config on
// error after printing an appropriate message.
func loadRootConfig(cmd *cobra.Command, cliCtx *CLIContext) (*config.Config, bool) {
	if cliCtx == nil {
		cmd.Println("Error: no CLI context available")
		if err := cmd.Help(); err != nil {
			cmd.Println(err)
		}

		return nil, false
	}

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

// resolveProviderAndArgs resolves the provider name and harness arguments from
// the command-line args and configuration.
func resolveProviderAndArgs(cmd *cobra.Command, cfg *config.Config, args []string) ([]string, []string, string) {
	cliCtx := CLIContextFromCmd(cmd)

	if len(args) == 0 || cliCtx.DefaultProviderExplicit() {
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

	// When --harness is set and the first arg is not a known provider,
	// treat all args as harness args and use the default provider.
	if harnessFlag != "" && !isKnownProvider(providerName, cfg) && cfg.DefaultProvider != "" {
		return []string{cfg.DefaultProvider}, args, cfg.DefaultProvider
	}

	return args, harnessArgs, providerName
}

// isKnownProvider reports whether name matches a configured or built-in provider.
func isKnownProvider(name string, cfg *config.Config) bool {
	if _, ok := cfg.Providers[name]; ok {
		return true
	}

	return providers.IsBuiltInProvider(name)
}

// providerFromArgs extracts the provider name from the first non-flag argument.
func providerFromArgs(cmd *cobra.Command, cfg *config.Config, args []string) (string, []string) {
	kairoArgs, harnessArgs := splitArgs(args)

	if len(kairoArgs) > 0 && !strings.HasPrefix(kairoArgs[0], "-") {
		harnessArgs = append(kairoArgs[1:], harnessArgs...)

		return kairoArgs[0], harnessArgs
	}

	if cfg.DefaultProvider != "" {
		return cfg.DefaultProvider, kairoArgs
	}

	cmd.Println("Error: No default provider set and first argument looks like a flag")
	cmd.Println("Run 'kairo setup' to configure a provider")

	return "", nil
}

// splitArgs splits args at the first "--" separator.
func splitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}

	return args, nil
}

// hasArgsSeparator reports whether args contain the "--" separator outside of flag values.
// It walks past flags and their values (e.g. --harness pi, -v value) looking for "--".
// Once a non-flag argument is seen, "--" is no longer valid as a separator.
func hasArgsSeparator(args []string) bool {
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
