package cmd

import (
	"errors"
	"io/fs"

	"github.com/dkmnx/kairo/internal/app"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/spf13/cobra"
)

// OrchestrateExecution runs the full execution pipeline: load config, resolve
// provider/harness via app.ResolveExecution, and dispatch to the Pi or
// standard provider path. User-facing messages stay in cmd.
func OrchestrateExecution(cmd *cobra.Command, args []string) {
	cliCtx := CLIContextFromCmd(cmd)

	cfg, ok := loadRootConfig(cmd, cliCtx)
	if !ok {
		return
	}

	res, err := app.ResolveExecution(cfg, app.ResolveOptions{
		Args:                    args,
		HarnessFlag:             harnessFlag,
		DefaultProviderExplicit: cliCtx.DefaultProviderExplicit(),
	})
	if err != nil {
		printResolveError(cmd, err)

		return
	}

	if res.Harness == harness.Pi {
		runPiProvider(cmd, cliCtx, cfg, res.Provider, res.ProviderName, res.Harness, res.HarnessArgs)
	} else {
		runStandardProvider(cmd, cliCtx, res.Provider, res.ProviderName, res.Harness, res.HarnessArgs)
	}
}

// printResolveError maps app resolution errors to recovery guidance.
func printResolveError(cmd *cobra.Command, err error) {
	switch {
	case errors.Is(err, app.ErrNoDefaultProvider):
		cmd.Println("No default provider set.")
		cmd.Println()
		cmd.Println("Usage:")
		cmd.Println("  kairo setup            # Configure providers")
		cmd.Println("  kairo default <name>   # Set default provider")
		cmd.Println("  kairo list             # List providers")
		cmd.Println("  kairo <provider>       # Use specific provider")
	case errors.Is(err, app.ErrFlagWithoutProvider):
		cmd.Println("Error: No default provider set and first argument looks like a flag")
		cmd.Println("Run 'kairo setup' to configure a provider")
	case errors.Is(err, app.ErrProviderNotConfigured):
		var notCfg *app.ProviderNotConfiguredError
		if errors.As(err, &notCfg) {
			cmd.Printf("Error: provider '%s' not configured\n", notCfg.Name)
		} else {
			cmd.Printf("Error: provider not configured\n")
		}
		cmd.Println("Run 'kairo list' to see configured providers")
	default:
		cmd.Printf("Error: %v\n", err)
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
		if errors.Is(err, fs.ErrNotExist) {
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
