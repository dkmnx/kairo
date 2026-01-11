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
// a letter and contain only alphanumeric characters.
var validProviderName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

// validateCustomProviderName validates a custom provider name and returns the validated name or an error.
func validateCustomProviderName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("provider name is required")
	}
	if !validProviderName.MatchString(name) {
		return "", fmt.Errorf("provider name must start with a letter and contain only alphanumeric characters")
	}
	if providers.IsBuiltInProvider(name) {
		return "", fmt.Errorf("reserved provider name")
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
	cfg, err := config.LoadConfig(dir)
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

func loadSecrets(dir string) (map[string]string, string, string) {
	secretsPath := filepath.Join(dir, "secrets.age")
	keyPath := filepath.Join(dir, "age.key")

	secrets := make(map[string]string)
	existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		if getVerbose() {
			ui.PrintInfo(fmt.Sprintf("Warning: Could not decrypt existing secrets: %v", err))
		}
	} else {
		secrets = config.ParseSecrets(existingSecrets)
	}
	return secrets, secretsPath, keyPath
}

func promptForProvider() string {
	providerList := providers.GetProviderList()
	ui.PrintHeader("Kairo Setup Wizard\n")
	ui.PrintWhite("Available providers:")
	for i, name := range providerList {
		ui.PrintWhite(fmt.Sprintf("  %d.   %s", i+1, name))
	}
	ui.PrintWhite("  q.   Exit\n")

	selection := ui.PromptWithDefault("Select provider to configure", "")
	return strings.TrimSpace(selection)
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
		customName := ui.Prompt("Provider name")
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

	model := ui.PromptWithDefault("Model", def.Model)

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
	baseURL := ui.PromptWithDefault("Base URL", defaultURL)
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

		secrets, secretsPath, keyPath := loadSecrets(dir)

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
			logAuditEvent(dir, func(logger *audit.Logger) error {
				return logger.LogSuccess("setup", configuredProvider, auditDetails)
			})
		}
	},
}

// parseIntOrZero converts a string to an integer, returning 0 if invalid.
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
