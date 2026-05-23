package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/yarlson/tap"
)

const customProviderName = "custom"

var providerNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// ValidateCustomProviderName validates that a custom provider name is well-formed
// and not reserved.
func ValidateCustomProviderName(name string) (string, error) {
	if name == "" {
		return "", errors.NewError(errors.ValidationError,
			"provider name is required")
	}
	if len(name) > validate.MaxProviderNameLength {
		return "", errors.NewError(errors.ValidationError,
			fmt.Sprintf("provider name must be at most %d characters (got %d)", validate.MaxProviderNameLength, len(name)))
	}
	if !providerNamePattern.MatchString(name) {
		return "", errors.NewError(errors.ValidationError,
			"provider name must start with a letter and contain only alphanumeric characters, underscores, and hyphens")
	}
	lowerName := strings.ToLower(name)
	if providers.IsBuiltInProvider(lowerName) {
		return "", errors.NewError(errors.ValidationError,
			fmt.Sprintf("reserved provider name: %s", lowerName))
	}

	return name, nil
}

// ProviderDefinition returns the built-in definition for the given provider,
// falling back to the provider name as the display name for custom providers.
func ProviderDefinition(providerName string) providers.ProviderDefinition {
	definition, _ := providers.BuiltInProvider(providerName)
	if definition.Name == "" {
		definition.Name = providerName
	}

	return definition
}

// ResolveProviderName resolves "custom" to a user-entered name, passing through all others.
func ResolveProviderName(providerName string) (string, error) {
	if providerName != customProviderName {
		return providerName, nil
	}

	customName := tap.Text(defaultCLIContext.RootCtx(), tap.TextOptions{
		Message: "Provider name",
	})

	return ValidateCustomProviderName(customName)
}

type modelValidationConfig struct {
	Model        string
	ProviderName string
	DisplayName  string
}

func validateConfiguredModel(cfg modelValidationConfig) error {
	if err := validate.ValidateProviderModel(cfg.ProviderName, cfg.Model); err != nil {
		return err
	}
	if providers.IsBuiltInProvider(cfg.ProviderName) || strings.TrimSpace(cfg.Model) != "" {
		return nil
	}

	return errors.NewError(errors.ValidationError,
		"model name is required for custom providers")
}

// ProviderBuildConfig holds parameters for building a provider configuration entry.
type ProviderBuildConfig struct {
	Definition providers.ProviderDefinition
	BaseURL    string
	Model      string
	EnvKey     string
	Exists     bool
	Existing   *config.Provider
}

// BuildProviderConfig constructs a Provider from the given build configuration,
// preserving existing settings when editing.
func BuildProviderConfig(cfg ProviderBuildConfig) config.Provider {
	if !cfg.Exists {
		return config.Provider{
			Name:    cfg.Definition.Name,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			EnvKey:  cfg.EnvKey,
		}
	}
	cfg.Existing.BaseURL = cfg.BaseURL
	cfg.Existing.Model = cfg.Model
	cfg.Existing.EnvKey = cfg.EnvKey

	return *cfg.Existing
}
