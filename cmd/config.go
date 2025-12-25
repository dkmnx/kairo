package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
			cmd.Printf("Error: unknown provider '%s'\n", providerName)
			cmd.Println("Available providers: anthropic, zai, minimax, kimi, deepseek, custom")
			return
		}

		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			return
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			cmd.Printf("Error creating config directory: %v\n", err)
			return
		}

		if err := crypto.EnsureKeyExists(dir); err != nil {
			cmd.Printf("Error creating encryption key: %v\n", err)
			return
		}

		builtinDef, _ := providers.GetBuiltInProvider(providerName)

		cfg, err := config.LoadConfig(dir)
		if err != nil && !os.IsNotExist(err) {
			cmd.Printf("Error loading config: %v\n", err)
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

		fmt.Printf("Configuring %s\n\n", provider.Name)

		apiKey, err := ui.PromptSecret("API Key")
		if err != nil {
			cmd.Printf("Error reading API key: %v\n", err)
			return
		}
		if err := validate.ValidateAPIKey(apiKey); err != nil {
			cmd.Printf("Error: %v\n", err)
			return
		}

		if builtinDef.BaseURL == "" {
			baseURL := ui.PromptWithDefault("Base URL", provider.BaseURL)
			if err := validate.ValidateURL(baseURL); err != nil {
				cmd.Printf("Error: %v\n", err)
				return
			}
			provider.BaseURL = baseURL
		} else {
			provider.BaseURL = builtinDef.BaseURL
			cmd.Printf("Base URL: %s\n", provider.BaseURL)
		}

		if builtinDef.Model == "" {
			provider.Model = ui.PromptWithDefault("Model", provider.Model)
		} else {
			provider.Model = builtinDef.Model
			cmd.Printf("Model: %s\n", provider.Model)
		}

		if len(builtinDef.EnvVars) > 0 && len(provider.EnvVars) == 0 {
			provider.EnvVars = builtinDef.EnvVars
		}

		cfg.Providers[providerName] = provider
		if cfg.DefaultProvider == "" {
			cfg.DefaultProvider = providerName
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}

		secretsPath := filepath.Join(dir, "secrets.age")
		secrets := fmt.Sprintf("%s_API_KEY=%s\n", providerName, apiKey)
		if err := crypto.EncryptSecrets(secretsPath, filepath.Join(dir, "age.key"), secrets); err != nil {
			cmd.Printf("Error saving API key: %v\n", err)
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Provider '%s' configured successfully", providerName))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
