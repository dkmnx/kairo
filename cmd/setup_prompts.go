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

	if len(cfg.Providers) > 0 {
		providerNames := make([]string, 0, len(cfg.Providers))
		for name := range cfg.Providers {
			providerNames = append(providerNames, name)
		}
		providerNames = append(providerNames, setupNewProvider)
		options := buildProviderListOptions(providerNames)

		fmt.Println()

		tap.Intro("Setup Provider", tap.MessageOptions{
			Hint: "Configure new provider or edit existing from Kairo",
		})

		selected := tap.Select(ctx, tap.SelectOptions[string]{
			Message: "Select provider to edit or setup new",
			Options: options,
		})

		if selected == setupNewProvider {
			providerList := providers.GetProviderList()
			providerList = append(providerList, "custom")
			options = buildProviderListOptions(providerList)

			selected = tap.Select(ctx, tap.SelectOptions[string]{
				Message: "Select provider to configure",
				Options: options,
			})
		}

		return selected
	}

	providerList := providers.GetProviderList()
	providerList = append(providerList, "custom")
	options := buildProviderListOptions(providerList)

	selected := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Select provider to configure",
		Options: options,
	})

	return selected
}

func parseProviderSelection(selection string) (string, bool) {
	if selection == "" {
		return "", false
	}

	if providers.IsBuiltInProvider(selection) {
		return selection, true
	}

	return "", false
}

func displayProviderHeader(provider config.Provider, isEdit, exists bool) {
	if isEdit && exists {
		tap.Message(fmt.Sprintf("Editing %s", provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	}
}

func promptForAPIKey(providerName string, secrets map[string]string, isEdit, exists bool) string {
	ctx := context.Background()

	if isEdit && exists {
		existingKey := secrets[APIKeyEnvVarName(providerName)]
		if existingKey == "" {
			return tap.Password(ctx, tap.PasswordOptions{Message: "API Key"})
		}
		if tap.Confirm(ctx, tap.ConfirmOptions{Message: "Modify API key?"}) {
			return tap.Password(ctx, tap.PasswordOptions{Message: "New API Key"})
		}

		return existingKey
	}

	return tap.Password(ctx, tap.PasswordOptions{Message: "API Key"})
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

	result := tap.Text(ctx, tap.TextOptions{
		Message:      cfg.Label,
		DefaultValue: cfg.DefaultValue,
		Placeholder:  cfg.DefaultValue,
	})

	result = strings.TrimSpace(result)
	if result == "" {
		return cfg.DefaultValue
	}

	return result
}

func promptForBaseURL(provider config.Provider, definition providers.ProviderDefinition, isEdit, exists bool) string {
	return promptForField(promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: provider.BaseURL,
		DefaultValue: definition.BaseURL,
		IsEdit:       isEdit,
		Exists:       exists,
	})
}

func promptForModel(provider config.Provider, definition providers.ProviderDefinition, isEdit, exists bool) string {
	return promptForField(promptFieldConfig{
		Label:        "Model",
		CurrentValue: provider.Model,
		DefaultValue: definition.Model,
		IsEdit:       isEdit,
		Exists:       exists,
	})
}
