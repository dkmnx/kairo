package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
)

// MaxProviderNameLength and MaxModelNameLength define upper bounds for provider
// and model name input lengths.
const (
	MaxProviderNameLength = 50
	MaxModelNameLength    = 100
)

// ValidateCrossProviderConfig checks that no two providers bind the same
// environment variable in a conflicting way: regular env vars conflict when
// set to different values, and API-key env vars conflict whenever shared by
// two providers (secret values are distinct, and sharing a key variable
// would overwrite one key with the other). All conflicts are reported in a
// single error.
func ValidateCrossProviderConfig(providerMap map[string]config.Provider) error {
	envVarMap := buildEnvVarBindings(providerMap)

	conflictingKeys := make([]string, 0)
	for key, sources := range envVarMap {
		if len(sources) >= 2 && envVarConflict(sources) {
			conflictingKeys = append(conflictingKeys, key)
		}
	}

	if len(conflictingKeys) > 0 {
		sort.Strings(conflictingKeys)

		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("config: conflicting environment variables across providers: %s", strings.Join(conflictingKeys, ", "))).
			WithContext("env_vars", strings.Join(conflictingKeys, ", "))
	}

	return nil
}

type envVarSource struct {
	value    string
	isKeyEnv bool
}

// buildEnvVarBindings maps every environment variable key to the providers
// that bind it, including each provider's API-key env var.
func buildEnvVarBindings(providerMap map[string]config.Provider) map[string]map[string]envVarSource {
	envVarMap := make(map[string]map[string]envVarSource)

	for providerName, provider := range providerMap {
		for _, envVar := range provider.EnvVars {
			key, value, ok := parseEnvVarEntry(envVar)
			if !ok {
				continue
			}
			addEnvVarBinding(envVarMap, key, providerName, envVarSource{value: value})
		}

		// The effective API-key env var: the config's explicit binding, or
		// the built-in catalog's declared variable.
		if keyEnv := effectiveKeyEnvVar(providerName, provider); keyEnv != "" {
			addEnvVarBinding(envVarMap, keyEnv, providerName, envVarSource{isKeyEnv: true})
		}
	}

	return envVarMap
}

func addEnvVarBinding(envVarMap map[string]map[string]envVarSource, key, providerName string, source envVarSource) {
	if _, exists := envVarMap[key]; !exists {
		envVarMap[key] = make(map[string]envVarSource)
	}
	envVarMap[key][providerName] = source
}

// effectiveKeyEnvVar returns the environment variable a provider uses for
// its API key: the explicit config binding, else the catalog's declared var.
func effectiveKeyEnvVar(providerName string, provider config.Provider) string {
	if provider.EnvKey != "" {
		return provider.EnvKey
	}
	if def, ok := providers.BuiltInProvider(providerName); ok {
		return def.APIKeyEnvVar
	}

	return ""
}

// envVarConflict reports whether the providers binding one env var conflict.
// A shared API-key variable always conflicts (the exported keys are distinct
// secrets and one would overwrite the other); regular env vars conflict only
// when their values differ.
func envVarConflict(sources map[string]envVarSource) bool {
	firstValue := ""
	firstSet := false
	for _, s := range sources {
		if s.isKeyEnv {
			return true
		}
		if !firstSet {
			firstValue = s.value
			firstSet = true

			continue
		}
		if s.value != firstValue {
			return true
		}
	}

	return false
}

// parseEnvVarEntry splits an env var entry into key and value. A valueless
// entry counts as an empty value (it still binds the key). Entries with an
// empty key are malformed and skipped.
func parseEnvVarEntry(envVar string) (key, value string, ok bool) {
	parts := strings.SplitN(envVar, "=", 2)
	key = strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false
	}
	value = ""
	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}

	return key, value, true
}

// ValidateProviderModel validates a model name against length and character
// constraints. The charset is uniform for every provider so the same input is
// accepted or rejected identically regardless of provider defaults. Empty and
// whitespace-only names pass so callers can enforce their own required-field
// rules.
func ValidateProviderModel(providerName, modelName string) error {
	if strings.TrimSpace(modelName) == "" {
		return nil
	}

	return validateModelName(modelName, providerName)
}

func validateModelName(modelName, providerName string) error {
	if len(modelName) > MaxModelNameLength {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: model name '%s' is too long (max %d characters)",
				providerName, modelName, MaxModelNameLength)).
			WithContext("model", modelName).
			WithContext("provider", providerName)
	}

	for _, r := range modelName {
		if !isValidModelRune(r) {
			return errors.NewError(errors.ValidationError,
				fmt.Sprintf("%s: model name '%s' contains invalid characters", providerName, modelName)).
				WithContext("model", modelName).
				WithContext("provider", providerName)
		}
	}

	return nil
}

// isValidModelRune reports whether r is allowed in a model identifier:
// alphanumerics plus the punctuation common in real model IDs (dashes,
// dots, underscores, brackets, colons, slashes, plus, at-sign, parens).
func isValidModelRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' ||
		r == '[' || r == ']' ||
		r == ':' || r == '/' || r == '+' ||
		r == '@' || r == '(' || r == ')'
}
