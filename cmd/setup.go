package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
)

// validProviderName validates custom provider names to ensure they start with
// a letter and contain only alphanumeric characters, underscores, or hyphens.
var validProviderName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

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
	if err != nil && !os.IsNotExist(err) {
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
		if verbose {
			ui.PrintInfo(fmt.Sprintf("Warning: Could not decrypt existing secrets: %v", err))
		}
	} else {
		secrets = config.ParseSecrets(existingSecrets)
	}
	return secrets, secretsPath, keyPath
}

func promptForProvider() string {
	providerList := providers.GetProviderList()
	ui.PrintHeader("Kairo Setup Wizard")
	ui.PrintInfo("Available providers:")
	for i, name := range providerList {
		ui.PrintInfo(fmt.Sprintf("  %d.   %s", i+1, name))
	}
	ui.PrintInfo("  q.   Exit")
	ui.PrintInfo("")

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

func configureProvider(dir string, cfg *config.Config, providerName string, secrets map[string]string, secretsPath, keyPath string) error {
	if providerName == "custom" {
		customName := ui.Prompt("Provider name")
		if customName == "" {
			return fmt.Errorf("provider name is required")
		}
		if !validProviderName.MatchString(customName) {
			return fmt.Errorf("provider name must start with a letter and contain only alphanumeric characters, underscores, or hyphens")
		}
		if providers.IsBuiltInProvider(customName) {
			return fmt.Errorf("reserved provider name")
		}
		providerName = customName
	}

	def, _ := providers.GetBuiltInProvider(providerName)
	if def.Name == "" {
		def.Name = providerName
	}

	ui.PrintInfo("")
	ui.PrintHeader(fmt.Sprintf("%s Configuration", def.Name))

	apiKey, err := ui.PromptSecret("API Key")
	if err != nil {
		return fmt.Errorf("reading API key: %w", err)
	}
	if err := validate.ValidateAPIKey(apiKey, def.Name); err != nil {
		return err
	}

	baseURL := ui.PromptWithDefault("Base URL", def.BaseURL)
	if err := validate.ValidateURL(baseURL, def.Name); err != nil {
		return err
	}

	model := ui.PromptWithDefault("Model", def.Model)

	provider := config.Provider{
		Name:    def.Name,
		BaseURL: baseURL,
		Model:   model,
	}
	if len(def.EnvVars) > 0 {
		provider.EnvVars = def.EnvVars
	}

	cfg.Providers[providerName] = provider
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = providerName
	}

	if err := config.SaveConfig(dir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	secrets[fmt.Sprintf("%s_API_KEY", providerName)] = apiKey

	var secretsBuilder strings.Builder
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := secrets[key]
		if key != "" && value != "" {
			secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
		return fmt.Errorf("saving API key: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s configured successfully", def.Name))
	ui.PrintInfo(fmt.Sprintf("Run 'kairo %s' to use this provider", providerName))
	return nil
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

		if !providers.RequiresAPIKey(providerName) {
			if err := configureAnthropic(dir, cfg, providerName); err != nil {
				ui.PrintError(fmt.Sprintf("Error: %v", err))
			}
			return
		}

		if err := configureProvider(dir, cfg, providerName, secrets, secretsPath, keyPath); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
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
