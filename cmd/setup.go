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

// validProviderName validates custom provider names to ensure they start with
// a letter and contain only alphanumeric characters, underscores, and hyphens.
var validProviderName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// validateCustomProviderName validates a custom provider name and returns the validated name or an error.
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
	if !validProviderName.MatchString(name) {
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
func buildProviderConfig(providerDef providers.ProviderDefinition, baseURL, model string) config.Provider {
	provider := config.Provider{
		Name:    providerDef.Name,
		BaseURL: baseURL,
		Model:   model,
	}
	if len(providerDef.EnvVars) > 0 {
		provider.EnvVars = providerDef.EnvVars
	}
	return provider
}

// addAndSaveProvider adds a provider to the config and saves it to disk.
// If setAsDefault is true and no default provider is set, the provider becomes the default.
func addAndSaveProvider(configDir string, appConfig *config.Config, providerName string, provider config.Provider, setAsDefault bool) error {
	appConfig.Providers[providerName] = provider
	if setAsDefault && appConfig.DefaultProvider == "" {
		appConfig.DefaultProvider = providerName
	}
	if err := config.SaveConfig(configDir, appConfig); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}
	return nil
}

// providerStatusIcon returns a status indicator for a provider's configuration.
// Note: This function is intentionally used only in tests (setup_test.go) to verify
// provider status display logic. It remains exported for test coverage purposes.
func providerStatusIcon(appConfig *config.Config, secrets map[string]string, provider string) string {
	if !providers.RequiresAPIKey(provider) {
		if _, exists := appConfig.Providers[provider]; exists {
			return ui.Green + "[x]" + ui.Reset
		}
		return "  "
	}

	apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
	for k := range secrets {
		if k == apiKeyKey {
			return ui.Green + "[x]" + ui.Reset
		}
	}
	return "  "
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
	appConfig, err := configCache.Get(configDir)
	if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		return nil, err
	}
	if err != nil {
		appConfig = &config.Config{
			Providers: make(map[string]config.Provider),
		}
	}
	return appConfig, nil
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
func promptForProvider(appConfig *config.Config) string {
	if len(appConfig.Providers) > 0 {
		// Has configured providers - show them + setup new option
		// Get names of configured providers
		providerNames := make([]string, 0, len(appConfig.Providers))
		for name := range appConfig.Providers {
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
func saveProviderConfiguration(configDir string, appConfig *config.Config, providerName string, provider config.Provider, apiKey string, secrets map[string]string, secretsPath, keyPath string, isEdit, wasExisting bool) (map[string]interface{}, error) {
	setAsDefault := appConfig.DefaultProvider == ""
	if err := addAndSaveProvider(configDir, appConfig, providerName, provider, setAsDefault); err != nil {
		return nil, err
	}

	// Save secrets
	secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey
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

func configureProvider(configDir string, appConfig *config.Config, providerName string, secrets map[string]string, secretsPath, keyPath string, isEdit bool) (string, map[string]interface{}, error) {
	// Handle custom provider name
	if providerName == "custom" {
		customName := tap.Text(context.Background(), tap.TextOptions{
			Message: "Provider name",
		})
		validatedName, err := validateCustomProviderName(customName)
		if err != nil {
			return "", nil, err
		}
		providerName = validatedName
	}

	// Get built-in provider definition if available
	// Error is intentionally ignored - custom providers will return an error,
	// which is expected. We'll use the providerName directly in that case.
	providerDef, _ := providers.GetBuiltInProvider(providerName)
	if providerDef.Name == "" {
		providerDef.Name = providerName
	}

	// Check if provider already exists
	provider, exists := appConfig.Providers[providerName]

	// Prompt for configuration details
	if isEdit && exists {
		// Edit mode - show "Editing" header
		tap.Message(fmt.Sprintf("Editing %s", provider.Name), tap.MessageOptions{
			Hint: "Press Enter to keep current values",
		})
	} else {
		// Setup mode - show "Configuration" header
		ui.PrintHeader(fmt.Sprintf("%s Configuration", providerDef.Name))
	}

	// API Key prompt
	var apiKey string
	if isEdit && exists {
		// Check if existing API key exists in secrets
		existingKey := secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))]
		if existingKey == "" {
			// No existing key - must prompt for new one
			apiKey = tap.Password(context.Background(), tap.PasswordOptions{
				Message: "API Key",
			})
		} else {
			// Has existing key - confirm before prompting
			modifyAPIKey := tap.Confirm(context.Background(), tap.ConfirmOptions{
				Message: "Modify API key?",
			})
			if modifyAPIKey {
				apiKey = tap.Password(context.Background(), tap.PasswordOptions{
					Message: "New API Key",
				})
			} else {
				// Keep existing API key from secrets
				apiKey = existingKey
			}
		}
	} else {
		// Setup mode - always prompt
		apiKey = tap.Password(context.Background(), tap.PasswordOptions{
			Message: "API Key",
		})
	}
	if err := validate.ValidateAPIKey(apiKey, providerDef.Name); err != nil {
		return "", nil, err
	}

	// Base URL prompt
	var baseURLDefault string
	if isEdit && exists {
		// Edit mode - use existing value as default
		if provider.BaseURL != "" {
			baseURLDefault = provider.BaseURL
		} else {
			if providerDef.BaseURL != "" {
				baseURLDefault = providerDef.BaseURL
			}
		}

		if baseURLDefault != "" {
			modifyBaseURL := tap.Confirm(context.Background(), tap.ConfirmOptions{
				Message: fmt.Sprintf("Modify Base URL? (current: %s)", baseURLDefault),
			})
			if modifyBaseURL {
				provider.BaseURL = tap.Text(context.Background(), tap.TextOptions{
					Message:      "New Base URL",
					DefaultValue: baseURLDefault,
					Placeholder:  baseURLDefault,
				})
			} else {
				provider.BaseURL = baseURLDefault
			}
		} else {
			provider.BaseURL = tap.Text(context.Background(), tap.TextOptions{
				Message:     "Base URL",
				Placeholder: providerDef.BaseURL,
			})
			provider.BaseURL = strings.TrimSpace(provider.BaseURL)
			if provider.BaseURL == "" {
				provider.BaseURL = providerDef.BaseURL
			}
		}
	} else {
		// Setup mode - prompt with default from definition
		baseURL := providerDef.BaseURL

		baseURL = tap.Text(context.Background(), tap.TextOptions{
			Message:      "Base URL",
			DefaultValue: baseURL,
			Placeholder:  baseURL,
		})
		// Use default value if user pressed Enter (empty input)
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			baseURL = providerDef.BaseURL
		}

		// Build provider config for URL
		if !exists {
			provider = config.Provider{
				Name:    providerDef.Name,
				BaseURL: baseURL,
			}
		} else {
			provider.BaseURL = baseURL
		}
	}

	if err := validate.ValidateURL(provider.BaseURL, providerDef.Name); err != nil {
		return "", nil, err
	}

	// Model prompt
	var modelDefault string
	if isEdit && exists {
		// Edit mode - use existing value as default
		if provider.Model != "" {
			modelDefault = provider.Model
		} else {
			if providerDef.Model != "" {
				modelDefault = providerDef.Model
			}
		}

		if modelDefault != "" {
			modifyModel := tap.Confirm(context.Background(), tap.ConfirmOptions{
				Message: fmt.Sprintf("Modify Model? (current: %s)", modelDefault),
			})
			if modifyModel {
				provider.Model = tap.Text(context.Background(), tap.TextOptions{
					Message:      "New Model",
					DefaultValue: modelDefault,
					Placeholder:  modelDefault,
				})
				provider.Model = strings.TrimSpace(provider.Model)
				if provider.Model == "" {
					provider.Model = modelDefault
				}
			}
			// If not modified, keep existing value
		} else {
			provider.Model = tap.Text(context.Background(), tap.TextOptions{
				Message:     "Model",
				Placeholder: providerDef.Model,
			})
			provider.Model = strings.TrimSpace(provider.Model)
			if provider.Model == "" {
				provider.Model = providerDef.Model
			}
		}
	} else {
		// Setup mode - prompt with default from definition
		model := providerDef.Model

		model = tap.Text(context.Background(), tap.TextOptions{
			Message:      "Model",
			DefaultValue: model,
			Placeholder:  model,
		})
		// Use default value if user pressed Enter (empty input)
		model = strings.TrimSpace(model)
		if model == "" {
			model = providerDef.Model
		}

		if !exists {
			provider = config.Provider{
				Name:  providerDef.Name,
				Model: model,
			}
		} else {
			provider.Model = model
		}
	}

	if err := validate.ValidateProviderModel(provider.Model, providerDef.Name); err != nil {
		return "", nil, err
	}

	// Validate model is non-empty for custom providers
	// Built-in providers like anthropic may use empty values
	if !providers.IsBuiltInProvider(providerName) {
		provider.Model = strings.TrimSpace(provider.Model)
		if provider.Model == "" {
			return "", nil, kairoerrors.NewError(kairoerrors.ValidationError,
				"model name is required for custom providers")
		}
	}

	// Build and save provider configuration
	details, err := saveProviderConfiguration(configDir, appConfig, providerName, provider, apiKey, secrets, secretsPath, keyPath, isEdit, exists)
	if err != nil {
		return "", nil, err
	}

	tap.Outro(fmt.Sprintf("%s configured successfully", provider.Name), tap.MessageOptions{
		Hint: fmt.Sprintf("Run 'kairo %s' to use this provider", providerName),
	})
	return providerName, details, nil
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

		appConfig, err := loadOrInitializeConfig(configDir)
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

		providerName := promptForProvider(appConfig)
		if providerName == "" {
			ui.PrintInfo("Setup cancelled")
			return
		}

		_, exists := appConfig.Providers[providerName]
		if _, _, err := configureProvider(configDir, appConfig, providerName, secrets, secretsPath, keyPath, exists); err != nil {
			ui.PrintError(err.Error())
			return
		}
	},
}

// promptForAPIKey prompts user for an API key and validates it.
// NOTE: This function is kept for backwards compatibility with tests.
// Production code now uses Tap TUI.
func promptForAPIKey(providerName string) (string, error) {
	apiKey, err := ui.PromptSecret("API Key")
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.ValidationError,
			"reading API key", err)
	}
	if err := validate.ValidateAPIKey(apiKey, providerName); err != nil {
		return "", err
	}
	return apiKey, nil
}

// promptForBaseURL prompts user for a base URL and validates it.
// NOTE: This function is kept for backwards compatibility with tests.
// Production code now uses Tap TUI.
func promptForBaseURL(defaultURL, providerName string) (string, error) {
	baseURL, err := ui.PromptWithDefault("Base URL", defaultURL)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.ValidationError,
			"reading base URL", err)
	}
	if err := validate.ValidateURL(baseURL, providerName); err != nil {
		return "", err
	}
	return baseURL, nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
