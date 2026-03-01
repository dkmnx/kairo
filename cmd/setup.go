package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

// providerNamePattern matches valid custom provider names: starting with a letter
// and containing only alphanumeric characters, underscores, and hyphens.
var providerNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Provider names must:
// - Be 1-50 characters long
// - Start with a letter
// - Contain only alphanumeric characters, underscores, and hyphens
// - Not be a reserved built-in provider name (case-insensitive)
func validateCustomProviderName(name string) (string, error) {
	if name == "" {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name is required")
	}
	// Check maximum length
	if len(name) > validate.MaxProviderNameLength {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("provider name must be at most %d characters (got %d)", validate.MaxProviderNameLength, len(name)))
	}
	if !providerNamePattern.MatchString(name) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			"provider name must start with a letter and contain only alphanumeric characters, underscores, and hyphens")
	}
	// Check for reserved provider names (case-insensitive)
	lowerName := strings.ToLower(name)
	if providers.IsBuiltInProvider(lowerName) {
		return "", kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("reserved provider name: %s", lowerName))
	}
	return name, nil
}

// buildProviderConfig creates a Provider configuration from a ProviderDefinition.
func buildProviderConfig(definition providers.ProviderDefinition, baseURL, model string) config.Provider {
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

// addAndSaveProvider adds a provider to the config and saves it to disk.
// If setAsDefault is true and no default provider is set, the provider becomes the default.
func addAndSaveProvider(configDir string, cfg *config.Config, providerName string, provider config.Provider, setAsDefault bool) error {
	cfg.Providers[providerName] = provider
	if setAsDefault && cfg.DefaultProvider == "" {
		cfg.DefaultProvider = providerName
	}
	if err := config.SaveConfig(configDir, cfg); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}
	return nil
}

// ensureConfigDirectory creates the config directory and encryption key if they don't exist.
func ensureConfigDirectory(configDir string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"creating config directory", err)
	}
	if err := crypto.EnsureKeyExists(configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"creating encryption key", err)
	}
	return nil
}

// loadOrInitializeConfig loads an existing config or creates a new empty one.
func loadOrInitializeConfig(configDir string) (*config.Config, error) {
	cfg, err := configCache.Get(configDir)
	if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		return nil, err
	}
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.Provider),
		}
	}
	return cfg, nil
}

// LoadSecrets loads and decrypts secrets from the specified directory.
// Returns the secrets map, secrets file path, key file path, and any error.
// Returns nil map with error if secrets file cannot be decrypted.
// Returns empty map with nil error if secrets file doesn't exist (first-time setup).
// LoadAndDecryptSecrets loads and decrypts secrets from the specified directory.
// Returns secrets map, secrets path, and key path. If secrets file doesn't exist
// or decryption fails, returns empty secrets map with appropriate error handling.
func LoadAndDecryptSecrets(configDir string) (map[string]string, string, string, error) {
	secretsPath := filepath.Join(configDir, config.SecretsFileName)
	keyPath := filepath.Join(configDir, config.KeyFileName)

	secrets := make(map[string]string)

	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		return secrets, secretsPath, keyPath, nil
	}

	existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		return nil, secretsPath, keyPath, err
	}

	secrets = config.ParseSecrets(existingSecrets)
	return secrets, secretsPath, keyPath, nil
}

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

// saveProviderConfiguration saves the provider configuration and secrets.
// Returns audit details for logging.
func saveProviderConfiguration(configDir string, cfg *config.Config, providerName string, provider config.Provider, apiKey string, secrets map[string]string, secretsPath, keyPath string, isEdit, wasExisting bool) (map[string]interface{}, error) {
	setAsDefault := cfg.DefaultProvider == ""
	if err := addAndSaveProvider(configDir, cfg, providerName, provider, setAsDefault); err != nil {
		return nil, err
	}

	// Save secrets
	secrets[apiKeyEnvVarName(providerName)] = apiKey
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving API key", err)
	}

	// Prepare audit details
	details := map[string]any{
		"display_name": provider.Name,
		"base_url":     provider.BaseURL,
		"model":        provider.Model,
	}
	if setAsDefault {
		details["set_as_default"] = "true"
	}
	if isEdit && wasExisting {
		details["action"] = "edit"
	}

	return details, nil
}

// resolveProviderName handles custom provider name input and validation.
func resolveProviderName(providerName string) (string, error) {
	if providerName != "custom" {
		return providerName, nil
	}

	customName := tap.Text(context.Background(), tap.TextOptions{
		Message: "Provider name",
	})
	return validateCustomProviderName(customName)
}

// getProviderDefinition retrieves the provider definition, using providerName for custom providers.
func getProviderDefinition(providerName string) providers.ProviderDefinition {
	definition, _ := providers.GetBuiltInProvider(providerName)
	if definition.Name == "" {
		definition.Name = providerName
	}
	return definition
}

// displayProviderHeader shows the appropriate header based on edit/setup mode.
func displayProviderHeader(provider config.Provider, definition providers.ProviderDefinition, isEdit, exists bool) {
	if isEdit && exists {
		tap.Message(fmt.Sprintf("Editing %s", provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	} else {
		ui.PrintHeader(fmt.Sprintf("%s Configuration", definition.Name))
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

// buildProviderConfigFromInput creates a Provider config from user input.
func buildProviderConfigFromInput(definition providers.ProviderDefinition, baseURL, model string, exists bool, existing config.Provider) config.Provider {
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

func configureProvider(configDir string, cfg *config.Config, providerName string, secrets map[string]string, secretsPath, keyPath string, isEdit bool) (string, map[string]interface{}, error) {
	validatedName, err := resolveProviderName(providerName)
	if err != nil {
		return "", nil, err
	}

	definition := getProviderDefinition(validatedName)
	provider, exists := cfg.Providers[validatedName]

	displayProviderHeader(provider, definition, isEdit, exists)

	apiKey := promptForAPIKey(validatedName, secrets, isEdit, exists)
	if err := validate.ValidateAPIKey(apiKey, definition.Name); err != nil {
		return "", nil, err
	}

	baseURL := promptForBaseURL(provider, definition, isEdit, exists)
	if err := validate.ValidateURL(baseURL, definition.Name); err != nil {
		return "", nil, err
	}

	model := promptForModel(provider, definition, isEdit, exists)
	if err := validate.ValidateProviderModel(model, definition.Name); err != nil {
		return "", nil, err
	}

	if !providers.IsBuiltInProvider(validatedName) {
		model = strings.TrimSpace(model)
		if model == "" {
			return "", nil, kairoerrors.NewError(kairoerrors.ValidationError,
				"model name is required for custom providers")
		}
	}

	provider = buildProviderConfigFromInput(definition, baseURL, model, exists, provider)

	details, err := saveProviderConfiguration(configDir, cfg, validatedName, provider, apiKey, secrets, secretsPath, keyPath, isEdit, exists)
	if err != nil {
		return "", nil, err
	}

	tap.Outro(fmt.Sprintf("%s configured successfully", provider.Name), tap.MessageOptions{
		Hint: fmt.Sprintf("Run 'kairo %s' to use this provider", validatedName),
	})
	return validatedName, details, nil
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup and edit wizard",
	Long:  "Run the interactive wizard to configure new providers or edit existing ones. Select a provider to edit or choose 'Setup new provider' to add a new provider.",
	Run: func(cmd *cobra.Command, args []string) {
		configDir := getConfigDir()
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				ui.PrintError("Cannot find home directory")
				return
			}
			configDir = filepath.Join(home, ".config", "kairo")
		}

		if err := ensureConfigDirectory(configDir); err != nil {
			ui.PrintError(err.Error())
			return
		}

		cfg, err := loadOrInitializeConfig(configDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		secrets, secretsPath, keyPath, err := LoadAndDecryptSecrets(configDir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
			ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
			ui.PrintInfo("Use --verbose for more details.")
			return
		}

		providerName := promptForProvider(cfg)
		if providerName == "" {
			ui.PrintInfo("Setup cancelled")
			return
		}

		_, exists := cfg.Providers[providerName]
		if _, _, err := configureProvider(configDir, cfg, providerName, secrets, secretsPath, keyPath, exists); err != nil {
			ui.PrintError(err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
