package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/yarlson/tap"
)

// buildProviderListOptions converts a provider list to Tap SelectOptions format.
func buildProviderListOptions(providerList []string) []tap.SelectOption[string] {
	options := make([]tap.SelectOption[string], len(providerList))
	for i, name := range providerList {
		options[i] = tap.SelectOption[string]{Value: name, Label: name}
	}

	return options
}

// promptForProvider displays interactive provider selection menu using Tap TUI.
func promptForProvider(cfg *config.Config) string {
	if len(cfg.Providers) > 0 {
		// Has configured providers - show them + setup new option
		// Get names of configured providers
		providerNames := make([]string, 0, len(cfg.Providers))
		for name := range cfg.Providers {
			providerNames = append(providerNames, name)
		}

		// Add "Setup new provider" as last option
		providerNames = append(providerNames, "Setup new provider")
		options := buildProviderListOptions(providerNames)

		fmt.Println()

		tap.Intro("Setup Provider", tap.MessageOptions{
			Hint: "Configure new provider or edit existing from Kairo",
		})

		selected := tap.Select(context.Background(), tap.SelectOptions[string]{
			Message: "Select provider to edit or setup new",
			Options: options,
		})

		// Check if "Setup new provider" was selected
		if selected == "Setup new provider" {
			providerList := providers.GetProviderList()
			providerList = append(providerList, "custom")
			options = buildProviderListOptions(providerList)

			selected = tap.Select(context.Background(), tap.SelectOptions[string]{
				Message: "Select provider to configure",
				Options: options,
			})
		}

		return selected
	}

	// No configured providers - go directly to provider selection (setup flow)
	providerList := providers.GetProviderList()
	providerList = append(providerList, "custom")
	options := buildProviderListOptions(providerList)

	selected := tap.Select(context.Background(), tap.SelectOptions[string]{
		Message: "Select provider to configure",
		Options: options,
	})

	return selected
}

// parseProviderSelection validates the provider selection.
func parseProviderSelection(selection string) (string, bool) {
	if selection == "" {
		return "", false
	}

	// Verify it's a valid built-in provider
	if providers.IsBuiltInProvider(selection) {
		return selection, true
	}

	return "", false
}

// displayProviderHeader shows the appropriate header based on edit/setup mode.
func displayProviderHeader(provider config.Provider, isEdit, exists bool) {
	if isEdit && exists {
		tap.Message(fmt.Sprintf("Editing %s", provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	}
}

// promptForAPIKey prompts for API key with edit mode support.
func promptForAPIKey(providerName string, secrets map[string]string, isEdit, exists bool) string {
	if isEdit && exists {
		existingKey := secrets[apiKeyEnvVarName(providerName)]
		if existingKey == "" {
			return tap.Password(context.Background(), tap.PasswordOptions{
				Message: "API Key",
			})
		}

		modifyAPIKey := tap.Confirm(context.Background(), tap.ConfirmOptions{
			Message: "Modify API key?",
		})
		if modifyAPIKey {
			return tap.Password(context.Background(), tap.PasswordOptions{
				Message: "New API Key",
			})
		}

		return existingKey
	}

	return tap.Password(context.Background(), tap.PasswordOptions{
		Message: "API Key",
	})
}

// promptFieldConfig holds configuration for prompting a provider field.
type promptFieldConfig struct {
	Label        string
	CurrentValue string
	DefaultValue string
	IsEdit       bool
	Exists       bool
}

// promptForField prompts for a provider field with edit mode support.
// Handles the common pattern of: edit confirmation -> text input -> fallback to default.
func promptForField(cfg promptFieldConfig) string {
	if cfg.IsEdit && cfg.Exists {
		effectiveDefault := cfg.CurrentValue
		if effectiveDefault == "" {
			effectiveDefault = cfg.DefaultValue
		}

		if effectiveDefault != "" {
			modifyField := tap.Confirm(context.Background(), tap.ConfirmOptions{
				Message: fmt.Sprintf("Modify %s? (current: %s)", cfg.Label, effectiveDefault),
			})
			if modifyField {
				return strings.TrimSpace(tap.Text(context.Background(), tap.TextOptions{
					Message:      fmt.Sprintf("New %s", cfg.Label),
					DefaultValue: effectiveDefault,
					Placeholder:  effectiveDefault,
				}))
			}

			return effectiveDefault
		}

		return strings.TrimSpace(tap.Text(context.Background(), tap.TextOptions{
			Message:     cfg.Label,
			Placeholder: cfg.DefaultValue,
		}))
	}

	result := tap.Text(context.Background(), tap.TextOptions{
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

// promptForBaseURL prompts for Base URL with edit mode support.
func promptForBaseURL(provider config.Provider, definition providers.ProviderDefinition, isEdit, exists bool) string {
	return promptForField(promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: provider.BaseURL,
		DefaultValue: definition.BaseURL,
		IsEdit:       isEdit,
		Exists:       exists,
	})
}

// promptForModel prompts for Model with edit mode support.
func promptForModel(provider config.Provider, definition providers.ProviderDefinition, isEdit, exists bool) string {
	return promptForField(promptFieldConfig{
		Label:        "Model",
		CurrentValue: provider.Model,
		DefaultValue: definition.Model,
		IsEdit:       isEdit,
		Exists:       exists,
	})
}
