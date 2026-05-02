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

// Injectable tap function variables for testability.
// These follow the same pattern as lookPath, execCommandContext, etc.
var (
	tapSelectFn   func(ctx context.Context, opts tap.SelectOptions[string]) string
	tapTextFn     = tapText
	tapPasswordFn = tapPassword
	tapConfirmFn  = tapConfirm
	tapIntroFn    = tapIntroFunc
	tapOutroFn    = tapOutroFunc
	tapMessageFn  = tapMessageFunc
)

// Production default for tapSelectFn, called during init.
func init() {
	tapSelectFn = defaultTapSelect
}

func defaultTapSelect(ctx context.Context, opts tap.SelectOptions[string]) string {
	return tap.Select(ctx, opts)
}

func tapText(ctx context.Context, opts tap.TextOptions) string {
	return tap.Text(ctx, opts)
}

func tapPassword(ctx context.Context, opts tap.PasswordOptions) string {
	return tap.Password(ctx, opts)
}

func tapConfirm(ctx context.Context, opts tap.ConfirmOptions) bool {
	return tap.Confirm(ctx, opts)
}

func tapIntroFunc(title string, opts ...tap.MessageOptions) {
	if len(opts) > 0 {
		tap.Intro(title, opts[0])
	} else {
		tap.Intro(title)
	}
}

func tapOutroFunc(message string, opts ...tap.MessageOptions) {
	if len(opts) > 0 {
		tap.Outro(message, opts[0])
	} else {
		tap.Outro(message)
	}
}

func tapMessageFunc(message string, opts ...tap.MessageOptions) {
	if len(opts) > 0 {
		tap.Message(message, opts[0])
	} else {
		tap.Message(message)
	}
}

func buildProviderListOptions(providerList []string) []tap.SelectOption[string] {
	options := make([]tap.SelectOption[string], len(providerList))
	for i, name := range providerList {
		options[i] = tap.SelectOption[string]{Value: name, Label: name}
	}

	return options
}

func promptForProvider(ctx context.Context, cfg *config.Config) string {
	if len(cfg.Providers) == 0 {
		return promptForNewProvider(ctx)
	}

	return promptForExistingOrNewProvider(ctx, cfg)
}

func promptForNewProvider(ctx context.Context) string {
	allProviders := append(providers.GetProviderList(), "custom")
	options := buildProviderListOptions(allProviders)

	return tapSelectFn(ctx, tap.SelectOptions[string]{
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

	tapIntroFn("Setup Provider", tap.MessageOptions{
		Hint: "Configure new provider or edit existing from Kairo",
	})

	selected := tapSelectFn(ctx, tap.SelectOptions[string]{
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
		tapMessageFn(fmt.Sprintf("Editing %s", cfg.Provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	}
}

func promptForAPIKey(ctx context.Context, cfg providerPromptConfig) string {
	if !cfg.IsEdit || !cfg.Exists {
		return tapPasswordFn(ctx, tap.PasswordOptions{Message: "API Key"})
	}

	existingKey := cfg.Secrets[APIKeyEnvVarName(cfg.ProviderName)]
	if existingKey == "" {
		return tapPasswordFn(ctx, tap.PasswordOptions{Message: "API Key"})
	}

	if tapConfirmFn(ctx, tap.ConfirmOptions{Message: "Modify API key?"}) {
		return tapPasswordFn(ctx, tap.PasswordOptions{Message: "New API Key"})
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

func promptForField(ctx context.Context, cfg promptFieldConfig) string {
	if cfg.IsEdit && cfg.Exists {
		return promptForFieldEdit(ctx, cfg)
	}

	result := strings.TrimSpace(tapTextFn(ctx, tap.TextOptions{
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
		if tapConfirmFn(ctx, tap.ConfirmOptions{
			Message: fmt.Sprintf("Modify %s? (current: %s)", cfg.Label, effectiveDefault),
		}) {
			return strings.TrimSpace(tapTextFn(ctx, tap.TextOptions{
				Message:      fmt.Sprintf("New %s", cfg.Label),
				DefaultValue: effectiveDefault,
				Placeholder:  effectiveDefault,
			}))
		}

		return effectiveDefault
	}

	return strings.TrimSpace(tapTextFn(ctx, tap.TextOptions{
		Message:     cfg.Label,
		Placeholder: cfg.DefaultValue,
	}))
}

func promptForBaseURL(ctx context.Context, cfg providerPromptConfig) string {
	return promptForField(ctx, promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: cfg.Provider.BaseURL,
		DefaultValue: cfg.Definition.BaseURL,
		IsEdit:       cfg.IsEdit,
		Exists:       cfg.Exists,
	})
}

func promptForModel(ctx context.Context, cfg providerPromptConfig) string {
	return promptForField(ctx, promptFieldConfig{
		Label:        "Model",
		CurrentValue: cfg.Provider.Model,
		DefaultValue: cfg.Definition.Model,
		IsEdit:       cfg.IsEdit,
		Exists:       cfg.Exists,
	})
}
