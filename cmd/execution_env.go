package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/providers"
)

// EnvProvider holds provider configuration needed for environment variable construction.
type EnvProvider struct {
	BaseURL string
	Model   string
	EnvVars []string
}

// PiAPIKeyEnvVar returns the API key environment variable name for a Pi provider.
func PiAPIKeyEnvVar(providerName string) (string, bool) {
	return providers.APIKeyEnvVarFor(providerName)
}

// BuildPiEnvVars constructs the environment variables for the Pi harness.
func BuildPiEnvVars(provider EnvProvider, providerName string) []string {
	return []string{
		fmt.Sprintf("PI_PROVIDER=%s", providerName),
		fmt.Sprintf("PI_MODEL=%s", provider.Model),
	}
}

// BuildBuiltInEnvVars constructs the standard Anthropic environment variables for a provider.
func BuildBuiltInEnvVars(provider EnvProvider) []string {
	return []string{
		fmt.Sprintf("%s=%s", envBaseURL, provider.BaseURL),
		fmt.Sprintf("%s=%s", envModel, provider.Model),
		fmt.Sprintf("%s=%s", envHaikuModel, provider.Model),
		fmt.Sprintf("%s=%s", envSonnetModel, provider.Model),
		fmt.Sprintf("%s=%s", envOpusModel, provider.Model),
		fmt.Sprintf("%s=%s", envSmallFast, provider.Model),
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
	return fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
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
	provider EnvProvider,
	providerName string,
) (EnvBuildResult, error) {
	builtIn := BuildBuiltInEnvVars(provider)

	secretsResult, err := LoadSecrets(cliCtx.RootCtx(), configDir)
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
