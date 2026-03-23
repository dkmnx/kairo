package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

// EnvProvider wraps provider config for environment building.
type EnvProvider struct {
	BaseURL string
	Model   string
	EnvVars []string
}

// BuildBuiltInEnvVars creates built-in environment variables.
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

// Backward compatibility
func buildBuiltInEnvVars(provider config.Provider) []string {
	return BuildBuiltInEnvVars(EnvProvider{
		BaseURL: provider.BaseURL,
		Model:   provider.Model,
		EnvVars: provider.EnvVars,
	})
}

// BuildSecretsEnvVars converts secrets map to env vars.
func BuildSecretsEnvVars(secrets map[string]string) []string {
	envVars := make([]string, 0, len(secrets))
	for k, v := range secrets {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	return envVars
}

// Backward compatibility
func buildSecretsEnvVars(secrets map[string]string) []string {
	return BuildSecretsEnvVars(secrets)
}

// APIKeyEnvVarName formats API key environment variable name.
func APIKeyEnvVarName(providerName string) string {
	return fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
}

// Backward compatibility alias
func apiKeyEnvVarName(providerName string) string {
	return APIKeyEnvVarName(providerName)
}

// RequiresAPIKey checks if provider needs API key.
func RequiresAPIKey(providerName string) bool {
	return providers.RequiresAPIKey(providerName)
}

// EnvBuildResult holds environment building result.
type EnvBuildResult struct {
	ProviderEnv []string
	Secrets     map[string]string
}

// BuildProviderEnv builds environment for provider execution.
func BuildProviderEnv(
	cliCtx *CLIContext,
	configDir string,
	provider EnvProvider,
	providerName string,
) (EnvBuildResult, error) {
	builtIn := BuildBuiltInEnvVars(provider)

	secretsResult, err := LoadSecrets(cliCtx.GetRootCtx(), configDir)
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

// Backward compatibility for tests
func buildProviderEnvironment(
	cliCtx *CLIContext,
	configDir string,
	provider config.Provider,
	providerName string,
) ([]string, map[string]string, error) {
	result, err := BuildProviderEnv(cliCtx, configDir, EnvProvider{
		BaseURL: provider.BaseURL,
		Model:   provider.Model,
		EnvVars: provider.EnvVars,
	}, providerName)
	if err != nil {
		return nil, nil, err
	}

	return result.ProviderEnv, result.Secrets, nil
}
