package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
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
func buildProviderConfig(def providers.ProviderDefinition, baseURL, model string) config.Provider {
	provider := config.Provider{
		Name:    def.Name,
		BaseURL: baseURL,
		Model:   model,
	}
	if len(def.EnvVars) > 0 {
		provider.EnvVars = def.EnvVars
	}
	return provider
}

// saveProviderConfigFile saves a provider configuration to the config file.
// If setAsDefault is true and no default provider is set, the provider becomes the default.
func saveProviderConfigFile(dir string, cfg *config.Config, providerName string, provider config.Provider, setAsDefault bool) error {
	cfg.Providers[providerName] = provider
	if setAsDefault && cfg.DefaultProvider == "" {
		cfg.DefaultProvider = providerName
	}
	if err := config.SaveConfig(dir, cfg); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}
	return nil
}

// validateAPIKey is a wrapper around validate.ValidateAPIKey for consistency.
func validateAPIKey(key, providerName string) error {
	return validate.ValidateAPIKey(key, providerName)
}

// validateBaseURL is a wrapper around validate.ValidateURL for consistency.
func validateBaseURL(url, providerName string) error {
	return validate.ValidateURL(url, providerName)
}

// providerStatusIcon returns a status indicator for a provider's configuration.
// Note: This function is intentionally used only in tests (setup_test.go) to verify
// provider status display logic. It remains exported for test coverage purposes.
func providerStatusIcon(cfg *config.Config, secrets map[string]string, provider string) string {
	if !providers.RequiresAPIKey(provider) {
		if _, exists := cfg.Providers[provider]; exists {
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
func ensureConfigDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"creating config directory", err)
	}
	if err := crypto.EnsureKeyExists(dir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"creating encryption key", err)
	}
	return nil
}

// loadOrInitializeConfig loads an existing config or creates a new empty one.
func loadOrInitializeConfig(dir string) (*config.Config, error) {
	cfg, err := configCache.Get(dir)
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
func LoadAndDecryptSecrets(dir string) (map[string]string, string, string, error) {
	secretsPath := filepath.Join(dir, config.SecretsFileName)
	keyPath := filepath.Join(dir, config.KeyFileName)

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

// promptForProvider displays interactive provider selection menu using Tap TUI.
//
// This function presents a list of available providers to the user using Tap
// and returns the selected provider name, or empty string if user cancels.
//
// Parameters:
//   - none (function uses providers.GetProviderList() internally)
//
// Returns:
//   - string: Selected provider name, or empty string if user chose to exit
//
// Error conditions: None (returns empty string on cancellation)
//
// Thread Safety: Not thread-safe (uses Tap which reads from stdin)
// Security Notes: This is a user-facing interactive function. Input is validated by Tap.
func promptForProvider() string {
	providerList := providers.GetProviderList()

	// Convert to tap.SelectOption format
	options := make([]tap.SelectOption[string], len(providerList))
	for i, name := range providerList {
		options[i] = tap.SelectOption[string]{Value: name, Label: name}
	}

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

// configureAnthropic configures the Native Anthropic provider with default settings.
//
// This function sets up the Anthropic provider with empty base URL and model,
// indicating it will use Anthropic's default endpoints. It saves the
// configuration and displays a success message to the user.
//
// Parameters:
//   - dir: Configuration directory where config.yaml should be saved
//   - cfg: Existing configuration object to update
//   - providerName: Name of provider to configure (should be "anthropic")
//
// Returns:
//   - error: Returns error if configuration cannot be saved
//
// Error conditions:
//   - Returns error when config file cannot be written (e.g., permissions, disk full)
//
// Thread Safety: Not thread-safe (modifies global config, file I/O)
// Security Notes: No sensitive data handled. Uses default Anthropic endpoints (no custom URL needed).
func configureAnthropic(dir string, cfg *config.Config, providerName string) error {
	def, _ := providers.GetBuiltInProvider(providerName)
	cfg.Providers[providerName] = config.Provider{
		Name:    def.Name,
		BaseURL: "",
		Model:   "",
	}
	if err := config.SaveConfig(dir, cfg); err != nil {
		return err
	}
	ui.PrintSuccess("Native Anthropic is ready to use!")
	ui.PrintInfo(fmt.Sprintf("Run 'kairo %s' or just 'kairo' to use it.", providerName))
	return nil
}

func configureProvider(dir string, cfg *config.Config, providerName string, secrets map[string]string, secretsPath, keyPath string) (string, map[string]interface{}, error) {
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

	def, _ := providers.GetBuiltInProvider(providerName)
	if def.Name == "" {
		def.Name = providerName
	}

	// Prompt for configuration details
	ui.PrintInfo("")
	ui.PrintHeader(fmt.Sprintf("%s Configuration", def.Name))

	apiKey := tap.Password(context.Background(), tap.PasswordOptions{
		Message: "API Key",
	})
	if err := validateAPIKey(apiKey, def.Name); err != nil {
		return "", nil, err
	}

	baseURL := tap.Text(context.Background(), tap.TextOptions{
		Message:     "Base URL",
		Placeholder: def.BaseURL,
	})
	if err := validateBaseURL(baseURL, def.Name); err != nil {
		return "", nil, err
	}

	model := tap.Text(context.Background(), tap.TextOptions{
		Message:     "Model",
		Placeholder: def.Model,
	})
	if err := validate.ValidateProviderModel(model, def.Name); err != nil {
		return "", nil, err
	}

	// Validate model is non-empty for custom providers
	// Built-in providers like anthropic may use empty values
	if !providers.IsBuiltInProvider(providerName) {
		model = strings.TrimSpace(model)
		if model == "" {
			return "", nil, kairoerrors.NewError(kairoerrors.ValidationError,
				"model name is required for custom providers")
		}
	}

	// Build and save provider configuration
	provider := buildProviderConfig(def, baseURL, model)
	setAsDefault := cfg.DefaultProvider == ""
	if err := saveProviderConfigFile(dir, cfg, providerName, provider, setAsDefault); err != nil {
		return "", nil, err
	}

	// Save secrets
	secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		return "", nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving API key", err)
	}

	// Prepare audit details
	details := map[string]interface{}{
		"display_name": def.Name,
		"base_url":     baseURL,
		"model":        model,
	}
	if setAsDefault {
		details["set_as_default"] = "true"
	}

	ui.PrintSuccess(fmt.Sprintf("%s configured successfully", def.Name))
	ui.PrintInfo(fmt.Sprintf("Run 'kairo %s' to use this provider", providerName))
	return providerName, details, nil
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  "Run the interactive setup wizard to configure providers",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				ui.PrintError("Cannot find home directory")
				return
			}
			dir = filepath.Join(home, ".config", "kairo")
		}

		if err := ensureConfigDirectory(dir); err != nil {
			ui.PrintError(err.Error())
			return
		}

		cfg, err := loadOrInitializeConfig(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		secrets, secretsPath, keyPath, err := LoadAndDecryptSecrets(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
			ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
			ui.PrintInfo("Use --verbose for more details.")
			return
		}

		providerName := promptForProvider()
		if providerName == "" {
			ui.PrintInfo("Setup cancelled")
			return
		}

		var configuredProvider string
		var auditDetails map[string]interface{}
		if !providers.RequiresAPIKey(providerName) {
			if err := configureAnthropic(dir, cfg, providerName); err != nil {
				ui.PrintError(err.Error())
				return
			}
			configuredProvider = providerName
			auditDetails = map[string]interface{}{
				"display_name": "Native Anthropic",
				"type":         "builtin_no_api_key",
			}
			if cfg.DefaultProvider == providerName {
				auditDetails["set_as_default"] = "true"
			}
		} else {
			provider, details, err := configureProvider(dir, cfg, providerName, secrets, secretsPath, keyPath)
			if err != nil {
				ui.PrintError(err.Error())
				return
			}
			configuredProvider = provider
			auditDetails = details
		}

		if configuredProvider != "" {
			if err := logAuditEvent(dir, func(logger *audit.Logger) error {
				return logger.LogSuccess("setup", configuredProvider, auditDetails)
			}); err != nil {
				ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
			}
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
	if err := validateAPIKey(apiKey, providerName); err != nil {
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
	if err := validateBaseURL(baseURL, providerName); err != nil {
		return "", err
	}
	return baseURL, nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
