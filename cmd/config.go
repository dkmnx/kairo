package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

		cfg, err := config.LoadConfig(dir)
		if err != nil && !os.IsNotExist(err) {
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
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
			ui.PrintError(fmt.Sprintf("Error reading API key: %v", err))
			return
		}
		if err := validate.ValidateAPIKey(apiKey, provider.Name); err != nil {
			ui.PrintError(err.Error())
			return
		}

		if builtinDef.BaseURL == "" {
			baseURL := ui.PromptWithDefault("Base URL", provider.BaseURL)
			if err := validate.ValidateURL(baseURL, provider.Name); err != nil {
				ui.PrintError(err.Error())
				return
			}
			provider.BaseURL = baseURL
		} else {
			currentBaseURL := provider.BaseURL
			if currentBaseURL == "" {
				currentBaseURL = builtinDef.BaseURL
			}
			baseURL := ui.PromptWithDefault("Base URL", currentBaseURL)
			if err := validate.ValidateURL(baseURL, provider.Name); err != nil {
				ui.PrintError(err.Error())
				return
			}
			provider.BaseURL = baseURL
		}

		if builtinDef.Model == "" {
			provider.Model = ui.PromptWithDefault("Model", provider.Model)
		} else {
			currentModel := provider.Model
			if currentModel == "" {
				currentModel = builtinDef.Model
			}
			provider.Model = ui.PromptWithDefault("Model", currentModel)
		}

		if len(builtinDef.EnvVars) > 0 && len(provider.EnvVars) == 0 {
			provider.EnvVars = builtinDef.EnvVars
		}

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

		oldProvider := cfg.Providers[providerName]
		cfg.Providers[providerName] = provider
		if cfg.DefaultProvider == "" {
			cfg.DefaultProvider = providerName
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		secrets[fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))] = apiKey

		var secretsBuilder strings.Builder
		for key, value := range secrets {
			if key != "" && value != "" {
				secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
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

		logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogConfig(providerName, action, changes)
		})
	},
}

func truncateKey(key string) string {
	if len(key) <= 9 {
		return "***"
	}
	return key[:5] + "********" + key[len(key)-4:]
}

func init() {
	rootCmd.AddCommand(configCmd)
}
