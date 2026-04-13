package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/yarlson/tap"
)

const setupNewProvider = "Setup new provider"

func buildProviderListOptions(providerList []string) []tap.SelectOption[string] {
	options := make([]tap.SelectOption[string], len(providerList))
	for i, name := range providerList {
		options[i] = tap.SelectOption[string]{Value: name, Label: name}
	}

	return options
}

func promptForProvider(cfg *config.Config) string {
	ctx := context.Background()

	if len(cfg.Providers) == 0 {
		return promptForNewProvider(ctx)
	}

	return promptForExistingOrNewProvider(ctx, cfg)
}

func promptForNewProvider(ctx context.Context) string {
	allProviders := append(providers.GetProviderList(), "custom")
	options := buildProviderListOptions(allProviders)

	return tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Select provider to configure",
		Options: options,
	})
}

func promptForExistingOrNewProvider(ctx context.Context, cfg *config.Config) string {
	existingNames := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		existingNames = append(existingNames, name)
	}
	existingNames = append(existingNames, setupNewProvider)
	options := buildProviderListOptions(existingNames)

	fmt.Println()

	tap.Intro("Setup Provider", tap.MessageOptions{
		Hint: "Configure new provider or edit existing from Kairo",
	})

	selected := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Select provider to edit or setup new",
		Options: options,
	})

	if selected == setupNewProvider {
		return promptForNewProvider(ctx)
	}

	return selected
}

type providerPromptConfig struct {
	ProviderName string
	Provider     config.Provider
	Definition   providers.ProviderDefinition
	Secrets      map[string]string
	IsEdit       bool
	Exists       bool
}

func displayProviderHeader(cfg providerPromptConfig) {
	if cfg.IsEdit && cfg.Exists {
		tap.Message(fmt.Sprintf("Editing %s", cfg.Provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	}
}

func promptForAPIKey(cfg providerPromptConfig) string {
	ctx := context.Background()

	if !cfg.IsEdit || !cfg.Exists {
		return tap.Password(ctx, tap.PasswordOptions{Message: "API Key"})
	}

	existingKey := cfg.Secrets[APIKeyEnvVarName(cfg.ProviderName)]
	if existingKey == "" {
		return tap.Password(ctx, tap.PasswordOptions{Message: "API Key"})
	}

	if tap.Confirm(ctx, tap.ConfirmOptions{Message: "Modify API key?"}) {
		return tap.Password(ctx, tap.PasswordOptions{Message: "New API Key"})
	}

	return existingKey
}

type promptFieldConfig struct {
	Label        string
	CurrentValue string
	DefaultValue string
	IsEdit       bool
	Exists       bool
}

func promptForField(cfg promptFieldConfig) string {
	ctx := context.Background()

	if cfg.IsEdit && cfg.Exists {
		return promptForFieldEdit(ctx, cfg)
	}

	result := strings.TrimSpace(tap.Text(ctx, tap.TextOptions{
		Message:      cfg.Label,
		DefaultValue: cfg.DefaultValue,
		Placeholder:  cfg.DefaultValue,
	}))

	if result == "" {
		return cfg.DefaultValue
	}

	return result
}

func promptForFieldEdit(ctx context.Context, cfg promptFieldConfig) string {
	effectiveDefault := cfg.CurrentValue
	if effectiveDefault == "" {
		effectiveDefault = cfg.DefaultValue
	}

	if effectiveDefault != "" {
		if tap.Confirm(ctx, tap.ConfirmOptions{
			Message: fmt.Sprintf("Modify %s? (current: %s)", cfg.Label, effectiveDefault),
		}) {
			return strings.TrimSpace(tap.Text(ctx, tap.TextOptions{
				Message:      fmt.Sprintf("New %s", cfg.Label),
				DefaultValue: effectiveDefault,
				Placeholder:  effectiveDefault,
			}))
		}

		return effectiveDefault
	}

	return strings.TrimSpace(tap.Text(ctx, tap.TextOptions{
		Message:     cfg.Label,
		Placeholder: cfg.DefaultValue,
	}))
}

func promptForBaseURL(cfg providerPromptConfig) string {
	return promptForField(promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: cfg.Provider.BaseURL,
		DefaultValue: cfg.Definition.BaseURL,
		IsEdit:       cfg.IsEdit,
		Exists:       cfg.Exists,
	})
}

func promptForModel(cfg providerPromptConfig) string {
	return promptForField(promptFieldConfig{
		Label:        "Model",
		CurrentValue: cfg.Provider.Model,
		DefaultValue: cfg.Definition.Model,
		IsEdit:       cfg.IsEdit,
		Exists:       cfg.Exists,
	})
}
