package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/providers"
)

// BuildPiEnvVars constructs the environment variables for the Pi harness.
func BuildPiEnvVars(provider config.Provider, providerName string) []string {
	return []string{
		fmt.Sprintf("PI_PROVIDER=%s", providerName),
		fmt.Sprintf("PI_MODEL=%s", provider.Model),
	}
}

// BuildBuiltInEnvVars constructs the standard Anthropic environment variables for a provider.
func BuildBuiltInEnvVars(provider config.Provider) []string {
	return []string{
		fmt.Sprintf("%s=%s", constants.EnvBaseURL, provider.BaseURL),
		fmt.Sprintf("%s=%s", constants.EnvModel, provider.Model),
		fmt.Sprintf("%s=%s", constants.EnvHaikuModel, provider.Model),
		fmt.Sprintf("%s=%s", constants.EnvSonnetModel, provider.Model),
		fmt.Sprintf("%s=%s", constants.EnvOpusModel, provider.Model),
		fmt.Sprintf("%s=%s", constants.EnvSmallFast, provider.Model),
		"NODE_OPTIONS=--no-deprecation",
	}
}

// EnvBuildResult holds the result of building provider environment variables.
type EnvBuildResult struct {
	ProviderEnv []string
	Secrets     map[string]string
}

// BuildProviderEnv assembles the complete environment for running a CLI harness,
// including provider env vars, secrets, and API key.
func BuildProviderEnv(
	cliCtx *CLIContext,
	configDir string,
	provider config.Provider,
	providerName string,
) (EnvBuildResult, error) {
	builtIn := BuildBuiltInEnvVars(provider)

	secretsResult, err := LoadSecrets(cliCtx, configDir)
	if err != nil {
		if providers.RequiresAPIKey(providerName) {
			return EnvBuildResult{}, err
		}
		secretsResult.Secrets = make(map[string]string)
	}

	providerEnv := mergeEnvVars(os.Environ(), builtIn, provider.EnvVars)

	return EnvBuildResult{
		ProviderEnv: providerEnv,
		Secrets:     secretsResult.Secrets,
	}, nil
}
