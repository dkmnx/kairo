package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
)

// PiAPIKeyEnvVar returns the API key environment variable name for a Pi provider.
func PiAPIKeyEnvVar(providerName string) (string, bool) {
	return providers.APIKeyEnvVarFor(providerName)
}

// HarnessAPIKeyEnvVar returns the environment variable name for a provider's
// API key suitable for harness execution. It first checks the provider registry
// for a specific env var name, falling back to the standard PROVIDERNAME_API_KEY
// convention.
func HarnessAPIKeyEnvVar(providerName string) string {
	if envVar, ok := providers.APIKeyEnvVarFor(providerName); ok {
		return envVar
	}

	return APIKeyEnvVarName(providerName)
}

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

// BuildSecretsEnvVars converts a secrets map into environment variable strings.
func BuildSecretsEnvVars(secrets map[string]string) []string {
	envVars := make([]string, 0, len(secrets))
	for k, v := range secrets {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	return envVars
}

// APIKeyEnvVarName returns the conventional environment variable name for a provider's API key.
func APIKeyEnvVarName(providerName string) string {
	return harness.APIKeyEnvVar(providerName)
}

// RequiresAPIKey reports whether the named provider requires an API key.
func RequiresAPIKey(providerName string) bool {
	return providers.RequiresAPIKey(providerName)
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
		if RequiresAPIKey(providerName) {
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
