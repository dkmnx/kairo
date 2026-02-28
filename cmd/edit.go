package cmd

import (
	"context"
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
	"github.com/yarlson/tap"
)

var editCmd = &cobra.Command{
	Use:   "edit [provider]",
	Short: "Edit provider configuration",
	Long:  "Add or update a provider configuration with API key, base URL, and model. If no provider is specified, shows an interactive list.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		var providerName string
		if len(args) == 0 {
			// Interactive selection using Tap
			providerList := providers.GetProviderList()
			providerList = append(providerList, "custom")

			// Convert to tap.SelectOption format
			options := make([]tap.SelectOption[string], len(providerList))
			for i, name := range providerList {
				options[i] = tap.SelectOption[string]{Value: name, Label: name}
			}

			selected := tap.Select(context.Background(), tap.SelectOptions[string]{
				Message: "Select provider to configure",
				Options: options,
			})
			providerName = selected
			if providerName == "" {
				ui.PrintInfo("Operation cancelled")
				return
			}
		} else {
			providerName = args[0]
		}

		isCustom := providerName == "custom"
		isBuiltIn := providers.IsBuiltInProvider(providerName)
		if !isCustom && !isBuiltIn {
			ui.PrintError(fmt.Sprintf("Unknown provider: '%s'", providerName))
			ui.PrintInfo("Available: anthropic, zai, minimax, kimi, deepseek, custom")
			return
		}

		builtinDef, _ := providers.GetBuiltInProvider(providerName)

		provider, exists := cfg.Providers[providerName]
		if !exists {
			provider = config.Provider{
				Name: builtinDef.Name,
			}
		}

		ui.PrintHeader(fmt.Sprintf("Configuring %s", provider.Name))

		apiKey := tap.Password(context.Background(), tap.PasswordOptions{
			Message: "API Key",
		})
		if err := validate.ValidateAPIKey(apiKey, provider.Name); err != nil {
			ui.PrintError(err.Error())
			return
		}

		baseURLDefault := provider.BaseURL
		if baseURLDefault == "" && builtinDef.BaseURL != "" {
			baseURLDefault = builtinDef.BaseURL
		}

		provider.BaseURL = tap.Text(context.Background(), tap.TextOptions{
			Message:     "Base URL",
			Placeholder: baseURLDefault,
		})

		if err := validate.ValidateURL(provider.BaseURL, provider.Name); err != nil {
			ui.PrintError(err.Error())
			return
		}

		modelDefault := provider.Model
		if modelDefault == "" && builtinDef.Model != "" {
			modelDefault = builtinDef.Model
		}

		provider.Model = tap.Text(context.Background(), tap.TextOptions{
			Message:     "Model",
			Placeholder: modelDefault,
		})

		if err := validate.ValidateProviderModel(provider.Model, provider.Name); err != nil {
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

		// Encrypt secrets FIRST to prevent inconsistent state
		// If this fails, the config won't be saved with a reference to non-existent secrets
		secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey

		if err := crypto.EncryptSecrets(secretsPath, keyPath, config.FormatSecrets(secrets)); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving API key: %v", err))
			return
		}

		// Validate cross-provider config before saving
		if err := validate.ValidateCrossProviderConfig(cfg); err != nil {
			ui.PrintError(err.Error())
			// Rollback: remove the just-encrypted secret
			delete(secrets, fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName)))
			if rollbackErr := crypto.EncryptSecrets(secretsPath, keyPath, config.FormatSecrets(secrets)); rollbackErr != nil {
				ui.PrintError(fmt.Sprintf("Rollback failed: %v", rollbackErr))
				ui.PrintInfo("Config saved but secrets may be outdated. Run 'kairo " + providerName + "' to reconfigure.")
			}
			return
		}

		// Now save config AFTER secrets are successfully encrypted
		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			// Rollback: remove the just-encrypted secret
			delete(secrets, fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName)))
			if rollbackErr := crypto.EncryptSecrets(secretsPath, keyPath, config.FormatSecrets(secrets)); rollbackErr != nil {
				ui.PrintError(fmt.Sprintf("Rollback failed: %v", rollbackErr))
				ui.PrintInfo("Secrets may be outdated. Run 'kairo " + providerName + "' to reconfigure.")
			}
			return
		}

		configCache.Invalidate(dir)

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
	rootCmd.AddCommand(editCmd)
}
