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

type modelValidationConfig struct {
	Model        string
	ProviderName string
	DisplayName  string
}

func validateConfiguredModel(cfg modelValidationConfig) error {
	if err := validate.ValidateProviderModel(cfg.Model, cfg.DisplayName); err != nil {
		return err
	}
	if providers.IsBuiltInProvider(cfg.ProviderName) || strings.TrimSpace(cfg.Model) != "" {
		return nil
	}

	return kairoerrors.NewError(kairoerrors.ValidationError,
		"model name is required for custom providers")
}

type ProviderBuildConfig struct {
	Definition providers.ProviderDefinition
	BaseURL    string
	Model      string
	Exists     bool
	Existing   *config.Provider
}

func BuildProviderConfig(cfg ProviderBuildConfig) config.Provider {
	if !cfg.Exists {
		return config.Provider{
			Name:    cfg.Definition.Name,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}
	}
	cfg.Existing.BaseURL = cfg.BaseURL
	cfg.Existing.Model = cfg.Model

	return *cfg.Existing
}
