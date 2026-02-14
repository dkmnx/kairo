package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
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
		return "", fmt.Errorf("provider name is required")
	}
	// Check maximum length (50 characters)
	if len(name) > 50 {
		return "", fmt.Errorf("provider name must be at most 50 characters (got %d)", len(name))
	}
	if !validProviderName.MatchString(name) {
		return "", fmt.Errorf("provider name must start with a letter and contain only alphanumeric characters, underscores, and hyphens")
	}
	// Check for reserved provider names (case-insensitive)
	lowerName := strings.ToLower(name)
	if providers.IsBuiltInProvider(lowerName) {
		return "", fmt.Errorf("reserved provider name: %s", lowerName)
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

// getSortedSecretsKeys returns a sorted slice of keys from a secrets map.
func getSortedSecretsKeys(secrets map[string]string) []string {
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// formatSecretsFileContent formats a secrets map into a string suitable for file storage.
func formatSecretsFileContent(secrets map[string]string) string {
	var builder strings.Builder
	keys := getSortedSecretsKeys(secrets)
	for _, key := range keys {
		value := secrets[key]
		if key != "" && value != "" {
			builder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}
	return builder.String()
}

// saveProviderConfigFile saves a provider configuration to the config file.
// If setAsDefault is true and no default provider is set, the provider becomes the default.
func saveProviderConfigFile(dir string, cfg *config.Config, providerName string, provider config.Provider, setAsDefault bool) error {
	cfg.Providers[providerName] = provider
	if setAsDefault && cfg.DefaultProvider == "" {
		cfg.DefaultProvider = providerName
	}
	if err := config.SaveConfig(dir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
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
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := crypto.EnsureKeyExists(dir); err != nil {
		return fmt.Errorf("creating encryption key: %w", err)
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
func LoadSecrets(dir string) (map[string]string, string, string, error) {
	secretsPath := filepath.Join(dir, "secrets.age")
	keyPath := filepath.Join(dir, "age.key")

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

// promptForProvider displays interactive provider selection menu and reads user choice.
//
// This function presents a numbered list of available providers to the user,
// prompts for selection, and returns the trimmed provider name.
// Special options 'q', 'exit', or 'done' return empty string.
//
// Parameters:
//   - none (function uses providers.GetProviderList() internally)
//
// Returns:
//   - string: Selected provider name, or empty string if user chose to exit
//
// Error conditions: None (returns empty string on input errors, but does not error)
//
// Thread Safety: Not thread-safe (uses ui.PromptWithDefault which reads from stdin)
// Security Notes: This is a user-facing interactive function. Input is trimmed but not validated here (validation happens in caller).
func promptForProvider() string {
	providerList := providers.GetProviderList()
	ui.PrintHeader("Kairo Setup Wizard\n")
	ui.PrintWhite("Available providers:")
	for i, name := range providerList {
		ui.PrintWhite(fmt.Sprintf("  %d.   %s", i+1, name))
	}
	ui.PrintWhite("  q.   Exit\n")

	selection, err := ui.PromptWithDefault("Select provider to configure", "")
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
		return ""
	}
	return strings.TrimSpace(selection)
}

// promptForHarness prompts user to select a CLI harness (claude or qwen).
func promptForHarness() string {
	ui.PrintHeader("CLI Harness Selection\n")
	ui.PrintWhite("Select CLI harness:")
	ui.PrintWhite("  1.   Claude Code (default)")
	ui.PrintWhite("  2.   Qwen Code")
	ui.PrintWhite("")

	selection, err := ui.PromptWithDefault("Selection [1-2]", "1")
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
		return ""
	}

	num := parseIntOrZero(selection)
	if num < 1 || num > 2 {
		return "claude"
	}

	if num == 2 {
		return "qwen"
	}
	return "claude"
}

// parseProviderSelection converts user input to a provider name.
func parseProviderSelection(selection string) (string, bool) {
	if selection == "" || selection == "done" || selection == "q" || selection == "exit" {
		return "", false
	}

	providerList := providers.GetProviderList()
	num := parseIntOrZero(selection)
	if num < 1 || num > len(providerList) {
		ui.PrintError(fmt.Sprintf("Invalid selection. Please enter a number 1-%d, or 'q' to exit.", len(providerList)))
		return "", false
	}

	return providerList[num-1], true
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
		customName, err := ui.Prompt("Provider name")
		if err != nil {
			return "", nil, fmt.Errorf("reading provider name: %w", err)
		}
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

	apiKey, err := promptForAPIKey(def.Name)
	if err != nil {
		return "", nil, err
	}

	baseURL, err := promptForBaseURL(def.BaseURL, def.Name)
	if err != nil {
		return "", nil, err
	}

	model, err := ui.PromptWithDefault("Model", def.Model)
	if err != nil {
		return "", nil, fmt.Errorf("reading model: %w", err)
	}

	// Validate model is non-empty for custom providers
	// Built-in providers like anthropic may use empty values
	if !providers.IsBuiltInProvider(providerName) {
		model = strings.TrimSpace(model)
		if model == "" {
			return "", nil, fmt.Errorf("model name is required for custom providers")
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
	secretsContent := formatSecretsFileContent(secrets)
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		return "", nil, fmt.Errorf("saving API key: %w", err)
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

// promptForAPIKey prompts user for an API key and validates it.
func promptForAPIKey(providerName string) (string, error) {
	apiKey, err := ui.PromptSecret("API Key")
	if err != nil {
		return "", fmt.Errorf("reading API key: %w", err)
	}
	if err := validateAPIKey(apiKey, providerName); err != nil {
		return "", err
	}
	return apiKey, nil
}

// promptForBaseURL prompts user for a base URL and validates it.
func promptForBaseURL(defaultURL, providerName string) (string, error) {
	baseURL, err := ui.PromptWithDefault("Base URL", defaultURL)
	if err != nil {
		return "", fmt.Errorf("reading base URL: %w", err)
	}
	if err := validateBaseURL(baseURL, providerName); err != nil {
		return "", err
	}
	return baseURL, nil
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
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			return
		}

		cfg, err := loadOrInitializeConfig(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		harnessSelection := promptForHarness()
		if harnessSelection != "" {
			cfg.DefaultHarness = harnessSelection
			if err := config.SaveConfig(dir, cfg); err != nil {
				ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
				return
			}
			ui.PrintSuccess(fmt.Sprintf("Harness set to: %s", harnessSelection))
		}

		secrets, secretsPath, keyPath, err := LoadSecrets(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
			ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
			ui.PrintInfo("Use --verbose for more details.")
			return
		}

		selection := promptForProvider()
		providerName, ok := parseProviderSelection(selection)
		if !ok {
			return
		}

		var configuredProvider string
		var auditDetails map[string]interface{}
		if !providers.RequiresAPIKey(providerName) {
			if err := configureAnthropic(dir, cfg, providerName); err != nil {
				ui.PrintError(fmt.Sprintf("Error: %v", err))
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
				ui.PrintError(fmt.Sprintf("Error: %v", err))
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

// parseIntOrZero converts a string to an integer, returning 0 if invalid.
//
// This function parses a string character by character, building an integer
// from ASCII digits. If any non-digit character is encountered, the
// function immediately returns 0. Used for parsing user-provided
// numeric selections in setup wizard.
//
// Parameters:
//   - s: String to parse as integer
//
// Returns:
//   - int: Parsed integer value, or 0 if string contains non-digit characters
//
// Error conditions: None (returns 0 for invalid input instead of error)
//
// Thread Safety: Thread-safe (pure function, no shared state)
// Performance Notes: O(n) where n is string length, returns early on first invalid character
func parseIntOrZero(s string) int {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
