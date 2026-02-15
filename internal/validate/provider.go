package validate

import (
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

// Validation limits for provider and model names.
const (
	// MaxProviderNameLength is the maximum length for custom provider names.
	// Most provider names are short identifiers (e.g., "anthropic", "openai").
	MaxProviderNameLength = 50

	// MaxModelNameLength is the maximum length for model names.
	// Most LLM model names are reasonable length (e.g., "claude-3-opus-20240229").
	MaxModelNameLength = 100
)

// validateCrossProviderConfig validates configuration across all providers to detect conflicts.
//
// This function checks for environment variable collisions where multiple providers
// attempt to set the same environment variable with different values. Collisions
// with identical values are allowed (idempotent).
func ValidateCrossProviderConfig(cfg *config.Config) error {
	// Build a map of env var names to their values and which providers set them
	type envVarSource struct {
		provider string
		value    string
	}
	envVarMap := make(map[string][]envVarSource)

	for providerName, provider := range cfg.Providers {
		for _, envVar := range provider.EnvVars {
			// Parse env var to get key and value
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			envVarMap[key] = append(envVarMap[key], envVarSource{
				provider: providerName,
				value:    value,
			})
		}
	}

	// Check for collisions - env vars set by multiple providers with different values
	for key, sources := range envVarMap {
		if len(sources) > 1 {
			// Check if all sources have the same value
			firstValue := sources[0].value
			allSame := true
			for _, s := range sources {
				if s.value != firstValue {
					allSame = false
					break
				}
			}
			if !allSame {
				return fmt.Errorf("environment variable collision: '%s' is set to different values by providers: %v",
					key, sources)
			}
		}
	}

	return nil
}

// ValidateProviderModel validates a model name against provider capabilities.
// For built-in providers with default models, this ensures the model is reasonable.
// Returns an error if the model name is invalid.
func ValidateProviderModel(providerName, modelName string) error {
	if modelName == "" {
		return nil // Empty model is allowed (will use provider default)
	}

	// Check if this is a built-in provider
	if def, ok := providers.GetBuiltInProvider(providerName); ok {
		// If provider has a default model, do basic validation
		if def.Model != "" {
			// Check model name length (most LLM model names are reasonable length)
			if len(modelName) > MaxModelNameLength {
				return fmt.Errorf("model name '%s' for provider '%s' is too long (max %d characters)", modelName, providerName, MaxModelNameLength)
			}
			// Check for valid characters (alphanumeric, hyphens, underscores, dots)
			for _, r := range modelName {
				if !isValidModelRune(r) {
					return fmt.Errorf("model name '%s' for provider '%s' contains invalid characters", modelName, providerName)
				}
			}
		}
	}

	return nil
}

// isValidModelRune returns true if the rune is valid in a model name.
func isValidModelRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.'
}
