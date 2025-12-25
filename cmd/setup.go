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

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  "Run the interactive setup wizard to configure providers",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				cmd.Println("Error: cannot find home directory")
				return
			}
			dir = filepath.Join(home, ".config", "kairo")
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			cmd.Printf("Error creating config directory: %v\n", err)
			return
		}

		if err := crypto.EnsureKeyExists(dir); err != nil {
			cmd.Printf("Error creating encryption key: %v\n", err)
			return
		}

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

		fmt.Println("Kairo Setup Wizard")
		fmt.Println("==================")
		fmt.Println()

		for name, def := range providers.BuiltInProviders {
			configure := ui.PromptWithDefault(fmt.Sprintf("Configure %s", name), "y/n")
			if configure != "y" && configure != "Y" {
				continue
			}

			provider := config.Provider{
				Name: def.Name,
			}

			apiKey, err := ui.PromptSecret(fmt.Sprintf("%s API Key", def.Name))
			if err != nil {
				cmd.Printf("Error reading API key: %v\n", err)
				continue
			}
			if err := validate.ValidateAPIKey(apiKey); err != nil {
				cmd.Printf("Error: %v\n", err)
				continue
			}

			if def.BaseURL == "" {
				baseURL := ui.PromptWithDefault("Base URL", "")
				if err := validate.ValidateURL(baseURL); err != nil {
					cmd.Printf("Error: %v\n", err)
					continue
				}
				provider.BaseURL = baseURL
			} else {
				provider.BaseURL = def.BaseURL
				fmt.Printf("Base URL: %s\n", provider.BaseURL)
			}

			if def.Model == "" {
				provider.Model = ui.PromptWithDefault("Model", "")
			} else {
				provider.Model = def.Model
				fmt.Printf("Model: %s\n", provider.Model)
			}

			if len(def.EnvVars) > 0 {
				provider.EnvVars = def.EnvVars
			}

			cfg.Providers[name] = provider

			secretsPath := filepath.Join(dir, "secrets.age")
			secrets := fmt.Sprintf("%s_API_KEY=%s\n", name, apiKey)
			if err := crypto.EncryptSecrets(secretsPath, filepath.Join(dir, "age.key"), secrets); err != nil {
				cmd.Printf("Error saving API key: %v\n", err)
				continue
			}

			ui.PrintSuccess(fmt.Sprintf("%s configured", def.Name))
			fmt.Println()
		}

		if len(cfg.Providers) > 0 {
			defaultProvider := ui.PromptWithDefault("Set default provider", "")
			if defaultProvider != "" {
				if _, ok := cfg.Providers[defaultProvider]; ok {
					cfg.DefaultProvider = defaultProvider
				}
			}
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}

		fmt.Println()
		ui.PrintSuccess("Setup complete!")
		fmt.Println("Run 'kairo list' to see configured providers")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
