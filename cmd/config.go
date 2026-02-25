package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config <provider>",
	Short: "Configure a provider",
	Long:  "Configure a provider with API key, base URL, and model",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]

		if !providers.IsBuiltInProvider(providerName) {
			ui.PrintError(fmt.Sprintf("Unknown provider: '%s'", providerName))
			ui.PrintInfo("Available: anthropic, zai, minimax, kimi, deepseek, custom")
			return
		}

		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating config directory: %v", err))
			return
		}

		if err := crypto.EnsureKeyExists(dir); err != nil {
			ui.PrintError(fmt.Sprintf("Error creating encryption key: %v", err))
			return
		}

		builtinDef, _ := providers.GetBuiltInProvider(providerName)

		cfg, err := configCache.Get(dir)
		if err != nil && !os.IsNotExist(err) {
			handleConfigError(cmd, err)
			return
		}
		if err != nil {
			cfg = &config.Config{
				Providers: make(map[string]config.Provider),
			}
		}

		provider, exists := cfg.Providers[providerName]
		if !exists {
			provider = config.Provider{
				Name: builtinDef.Name,
			}
		}

		ui.PrintHeader(fmt.Sprintf("Configuring %s", provider.Name))

		apiKey, err := ui.PromptSecret("API Key")
		if err != nil {
			if errors.Is(err, ui.ErrUserCancelled) {
				cmd.Println("\nConfiguration cancelled.")
				return
			}
			ui.PrintError(fmt.Sprintf("Error reading API key: %v", err))
			return
		}
		if err := validate.ValidateAPIKey(apiKey, provider.Name); err != nil {
			ui.PrintError(err.Error())
			return
		}

		provider.BaseURL, err = promptWithDefaultAndValidate("Base URL", provider.BaseURL, builtinDef.BaseURL, func(value string) error {
			return validate.ValidateURL(value, provider.Name)
		})
		if err != nil {
			ui.PrintError(err.Error())
			return
		}

		provider.Model, err = promptWithDefaultAndValidate("Model", provider.Model, builtinDef.Model, nil)
		if err != nil {
			ui.PrintError(err.Error())
			return
		}

		// Always refresh EnvVars from provider definition
		// This ensures new env vars from updated definitions are merged
		if len(builtinDef.EnvVars) > 0 {
			provider.EnvVars = builtinDef.EnvVars
		}

		secrets, secretsPath, keyPath, err := LoadAndDecryptSecrets(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
			ui.PrintInfo("Your encryption key may be corrupted. Try 'kairo rotate' to fix.")
			ui.PrintInfo("Use --verbose for more details.")
			return
		}

		oldProvider := cfg.Providers[providerName]
		cfg.Providers[providerName] = provider
		if cfg.DefaultProvider == "" {
			cfg.DefaultProvider = providerName
		}

		if cfg.DefaultModels == nil {
			cfg.DefaultModels = make(map[string]string)
		}
		cfg.DefaultModels[providerName] = provider.Model

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		configCache.Invalidate(dir)

		secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey

		if err := crypto.EncryptSecrets(secretsPath, keyPath, config.FormatSecrets(secrets)); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving API key: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Provider '%s' configured successfully", providerName))

		action := "add"
		if exists {
			action = "update"
		}

		var changes []audit.Change
		if provider.BaseURL != "" && provider.BaseURL != oldProvider.BaseURL {
			old := oldProvider.BaseURL
			if old == "" && builtinDef.BaseURL != "" {
				old = builtinDef.BaseURL
			}
			changes = append(changes, audit.Change{Field: "base_url", Old: old, New: provider.BaseURL})
		}
		if provider.Model != "" && provider.Model != oldProvider.Model {
			old := oldProvider.Model
			if old == "" && builtinDef.Model != "" {
				old = builtinDef.Model
			}
			changes = append(changes, audit.Change{Field: "model", Old: old, New: provider.Model})
		}

		if err := logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogConfig(providerName, action, changes)
		}); err != nil {
			ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func promptWithDefaultAndValidate(fieldName, currentValue, defaultValue string, validator func(string) error) (string, error) {
	promptValue := currentValue
	if defaultValue != "" && currentValue == "" {
		promptValue = defaultValue
	}

	value, err := ui.PromptWithDefault(fieldName, promptValue)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	if validator != nil {
		if err := validator(value); err != nil {
			return "", err
		}
	}

	return value, nil
}
