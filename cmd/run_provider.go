package cmd

import (
	"github.com/dkmnx/kairo/internal/app"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

// runPiProvider launches the Pi harness. Pi does not use the temp-token
// wrapper path; keys for all configured providers are injected so a single
// Pi session can switch models/providers without relaunching.
func runPiProvider(
	cmd *cobra.Command,
	cliCtx *CLIContext,
	cfg *config.Config,
	provider config.Provider,
	providerName, harnessToUse string,
	harnessArgs []string,
) {
	envResult, err := app.BuildProviderEnv(
		cliCtx.RootCtx(), cliCtx.Crypto(), cliCtx.ConfigDir(), provider, providerName,
	)
	if err != nil {
		handleSecretsError(err)

		return
	}

	hasAnyKey := app.InjectPiAPIKeys(&envResult, cfg)

	execCfg := buildExecutionConfig(
		cmd, cliCtx, envResult.ProviderEnv, provider,
		providerName, harnessToUse, harnessArgs, "",
	)

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
	envResult, err := app.BuildProviderEnv(
		cliCtx.RootCtx(), cliCtx.Crypto(), cliCtx.ConfigDir(), provider, providerName,
	)
	if err != nil {
		handleSecretsError(err)

		return
	}

	apiKey, hasKey := app.LookupAPIKeyWithFallback(envResult.Secrets, providerName)

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
		RootCtx:       cliCtx.RootCtx(),
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
