package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/yarlson/tap"
)

var providerNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func ValidateCustomProviderName(name string) (string, error) {
	if name == "" {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name is required")
	}
	if len(name) > validate.MaxProviderNameLength {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("provider name must be at most %d characters (got %d)", validate.MaxProviderNameLength, len(name)))
	}
	if !providerNamePattern.MatchString(name) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name must start with a letter and contain only alphanumeric characters, underscores, and hyphens")
	}
	lowerName := strings.ToLower(name)
	if providers.IsBuiltInProvider(lowerName) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("reserved provider name: %s", lowerName))
	}

	return name, nil
}

func BuildProviderConfig(definition providers.ProviderDefinition, baseURL, model string) config.Provider {
	provider := config.Provider{
		Name:    definition.Name,
		BaseURL: baseURL,
		Model:   model,
	}
	if len(definition.EnvVars) > 0 {
		provider.EnvVars = definition.EnvVars
	}

	return provider
}

func GetProviderDefinition(providerName string) providers.ProviderDefinition {
	definition, _ := providers.GetBuiltInProvider(providerName)
	if definition.Name == "" {
		definition.Name = providerName
	}

	return definition
}

func ResolveProviderName(providerName string) (string, error) {
	if providerName != "custom" {
		return providerName, nil
	}

	customName := tap.Text(context.Background(), tap.TextOptions{
		Message: "Provider name",
	})

	return ValidateCustomProviderName(customName)
}

func BuildProviderConfigFromInput(
	definition providers.ProviderDefinition,
	baseURL, model string,
	exists bool,
	existing config.Provider,
) config.Provider {
	if !exists {
		return config.Provider{
			Name:    definition.Name,
			BaseURL: baseURL,
			Model:   model,
		}
	}
	existing.BaseURL = baseURL
	existing.Model = model

	return existing
}

// Backward compatibility aliases
type BuildProviderConfigParams struct {
	Definition providers.ProviderDefinition
	BaseURL    string
	Model      string
	Exists     bool
	Existing   config.Provider
}

func buildProviderConfigFromInput(params BuildProviderConfigParams) config.Provider {
	return BuildProviderConfigFromInput(params.Definition, params.BaseURL, params.Model, params.Exists, params.Existing)
}

func buildProviderConfig(definition providers.ProviderDefinition, baseURL, model string) config.Provider {
	return BuildProviderConfig(definition, baseURL, model)
}

func validateCustomProviderName(name string) (string, error) {
	return ValidateCustomProviderName(name)
}

func getProviderDefinition(providerName string) providers.ProviderDefinition {
	return GetProviderDefinition(providerName)
}

func resolveProviderName(providerName string) (string, error) {
	return ResolveProviderName(providerName)
}
