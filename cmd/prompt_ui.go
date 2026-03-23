package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

type PromptUI interface {
	Select(ctx context.Context, message string, options []SelectOption) string
	Text(ctx context.Context, opts TextOptions) string
	Password(ctx context.Context, opts PasswordOptions) string
	Confirm(ctx context.Context, opts ConfirmOptions) bool
	Intro(title string, opts MessageOptions)
	Message(msg string, opts MessageOptions)
}

type SelectOption struct {
	Value string
	Label string
}

type TextOptions struct {
	Message      string
	DefaultValue string
	Placeholder  string
}

type PasswordOptions struct {
	Message string
}

type ConfirmOptions struct {
	Message string
}

type MessageOptions struct {
	Hint string
}

func selectProviderWithExisting(_ *config.Config, providerNames []string, ui PromptUI) string {
	ctx := context.Background()

	options := make([]SelectOption, len(providerNames))
	for i, name := range providerNames {
		options[i] = SelectOption{Value: name, Label: name}
	}

	ui.Intro("Setup Provider", MessageOptions{
		Hint: "Test hint",
	})

	return ui.Select(ctx, "Select provider", options)
}

func selectProviderNoExisting(ui PromptUI) string {
	ctx := context.Background()

	providerList := providers.GetProviderList()
	providerList = append(providerList, "custom")
	options := make([]SelectOption, len(providerList))
	for i, name := range providerList {
		options[i] = SelectOption{Value: name, Label: name}
	}

	return ui.Select(ctx, "Select provider to configure", options)
}

//nolint:unparam // providerName is always "anthropic" but used to construct API key env var name
func promptAPIKey(providerName string, secrets map[string]string, isEdit, exists bool, ui PromptUI) string {
	ctx := context.Background()

	if isEdit && exists {
		existingKey := secrets[APIKeyEnvVarName(providerName)]
		if existingKey == "" {
			return ui.Password(ctx, PasswordOptions{Message: "API Key"})
		}

		if ui.Confirm(ctx, ConfirmOptions{Message: "Modify API key?"}) {
			return ui.Password(ctx, PasswordOptions{Message: "New API Key"})
		}

		return existingKey
	}

	return ui.Password(ctx, PasswordOptions{Message: "API Key"})
}

type testPromptFieldConfig struct {
	Label        string
	CurrentValue string
	DefaultValue string
	IsEdit       bool
	Exists       bool
}

func promptField(cfg testPromptFieldConfig, ui PromptUI) string {
	ctx := context.Background()

	if cfg.IsEdit && cfg.Exists {
		effectiveDefault := cfg.CurrentValue
		if effectiveDefault == "" {
			effectiveDefault = cfg.DefaultValue
		}
		if effectiveDefault != "" {
			if ui.Confirm(ctx, ConfirmOptions{
				Message: fmt.Sprintf("Modify %s? (current: %s)", cfg.Label, effectiveDefault),
			}) {
				return strings.TrimSpace(ui.Text(ctx, TextOptions{
					Message:      fmt.Sprintf("New %s", cfg.Label),
					DefaultValue: effectiveDefault,
					Placeholder:  effectiveDefault,
				}))
			}

			return effectiveDefault
		}

		return strings.TrimSpace(ui.Text(ctx, TextOptions{
			Message:     cfg.Label,
			Placeholder: cfg.DefaultValue,
		}))
	}

	result := ui.Text(ctx, TextOptions{
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

func promptFieldKeepCurrent(cfg testPromptFieldConfig, ui PromptUI) string {
	ctx := context.Background()

	effectiveDefault := cfg.CurrentValue
	if effectiveDefault == "" {
		effectiveDefault = cfg.DefaultValue
	}

	if ui.Confirm(ctx, ConfirmOptions{
		Message: fmt.Sprintf("Modify %s? (current: %s)", cfg.Label, effectiveDefault),
	}) {
		return strings.TrimSpace(ui.Text(ctx, TextOptions{
			Message:      fmt.Sprintf("New %s", cfg.Label),
			DefaultValue: effectiveDefault,
			Placeholder:  effectiveDefault,
		}))
	}

	return effectiveDefault
}

func promptFieldNew(cfg testPromptFieldConfig, ui PromptUI) string {
	ctx := context.Background()

	result := ui.Text(ctx, TextOptions{
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
