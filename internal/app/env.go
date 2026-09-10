// Package app holds harness-environment assembly shared by the CLI without
// depending on cobra or CLIContext.
package app

import (
	"context"
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/envutil"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/secrets"
)

// CustomProviderName is the reserved provider key for user-defined providers.
const CustomProviderName = "custom"

// BuiltInEnvVars constructs the standard Anthropic environment variables for a provider.
func BuiltInEnvVars(provider config.Provider) []string {
	return []string{
		fmt.Sprintf("%s=%s", constants.EnvBaseURL, provider.BaseURL),
		fmt.Sprintf("%s=%s", constants.EnvModel, provider.Model),
		"NODE_OPTIONS=--no-deprecation",
	}
}

// EnvBuildResult holds the result of building provider environment variables.
type EnvBuildResult struct {
	ProviderEnv []string
	Secrets     map[string]string
}

// BuildProviderEnv assembles the complete environment for running a CLI harness,
// including provider env vars and decrypted secrets.
func BuildProviderEnv(
	ctx context.Context,
	cryptoSvc crypto.Service,
	configDir string,
	provider config.Provider,
	providerName string,
) (EnvBuildResult, error) {
	builtIn := BuiltInEnvVars(provider)

	secretsResult, err := secrets.Load(ctx, cryptoSvc, configDir)
	if err != nil {
		if providers.RequiresAPIKey(providerName) {
			return EnvBuildResult{}, err
		}
		secretsResult.Secrets = make(map[string]string)
	}

	providerEnv := envutil.Merge(os.Environ(), builtIn, provider.EnvVars)

	return EnvBuildResult{
		ProviderEnv: providerEnv,
		Secrets:     secretsResult.Secrets,
	}, nil
}

// Secrets vs process-env naming
//
// Two different names exist for a provider API key and must not be confused:
//
//  1. secrets.age map key: always the conventional PROVIDER_API_KEY form
//     (harness.APIKeyEnvVar). That is the on-disk storage format.
//  2. child-process env var: APIKeyEnvVarName (catalog-defined name first,
//     e.g. HF_TOKEN or GEMINI_API_KEY, then a configured EnvKey, then the
//     conventional form). That is what the harness/tool actually reads.
//
// Setup and delete operate on (1). Pi injection and Crush wrapper exec use (2).
// APIKeyEnvVarName resolves the env var that should receive a provider's API
// key. Catalog-defined names win, then a configured EnvKey, then the
// conventional PROVIDER_API_KEY form.
func APIKeyEnvVarName(providerName string, provider config.Provider) string {
	if envVar, ok := providers.APIKeyEnvVarFor(providerName); ok {
		return envVar
	}

	if provider.EnvKey != "" {
		return provider.EnvKey
	}

	return harness.APIKeyEnvVar(providerName)
}

// InjectPiAPIKeys appends every configured provider's available API key to
// the child environment. Pi is designed for multi-provider sessions (switch
// providers mid-run without relaunching), so it needs all stored keys — not
// only the one used to start the session. Returns whether any key was found.
func InjectPiAPIKeys(envResult *EnvBuildResult, cfg *config.Config) bool {
	hasAnyKey := false
	for pName, p := range cfg.Providers {
		val, found := LookupAPIKeyWithFallback(envResult.Secrets, pName)
		if !found {
			continue
		}

		envVar := APIKeyEnvVarName(pName, p)
		envResult.ProviderEnv = append(envResult.ProviderEnv, fmt.Sprintf("%s=%s", envVar, val))
		hasAnyKey = true
	}

	return hasAnyKey
}

// LookupAPIKeyWithFallback finds a provider API key in the secrets map,
// falling back to the reserved custom-provider key for non-custom providers.
func LookupAPIKeyWithFallback(secretsMap map[string]string, providerName string) (string, bool) {
	if val, ok := secretsMap[harness.APIKeyEnvVar(providerName)]; ok {
		return val, true
	}

	if providerName != CustomProviderName {
		if val, ok := secretsMap[harness.APIKeyEnvVar(CustomProviderName)]; ok {
			return val, true
		}
	}

	return "", false
}
