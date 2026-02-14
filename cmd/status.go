package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Test all configured providers",
	Long:  "Test connectivity for all configured providers",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintWarn("Config directory not found")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintWarn("No providers configured")
				ui.PrintInfo("Run 'kairo setup' to get started")
				return
			}
			handleConfigError(cmd, err)
			return
		}

		if len(cfg.Providers) == 0 {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return
		}

		ui.PrintWhite("Checking provider status...")
		fmt.Println()

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")

		secrets := make(map[string]string)
		if _, err := os.Stat(secretsPath); err == nil {
			secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
			if err != nil {
				ui.PrintWarn(fmt.Sprintf("Could not decrypt secrets file: %v", err))
				ui.PrintInfo("API key status will not be shown. Use --verbose for more details.")
				return
			}
			secrets = config.ParseSecrets(secretsContent)
		}

		names := sortProviderNames(cfg.Providers, cfg.DefaultProvider)

		for _, name := range names {
			provider := cfg.Providers[name]
			isDefault := (name == cfg.DefaultProvider)

			if provider.BaseURL == "" {
				ui.PrintWarn(fmt.Sprintf("%s - No base URL configured", name))
				continue
			}

			apiKeyVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(name))
			_, hasApiKey := secrets[apiKeyVar]

			if !hasApiKey {
				ui.PrintWarn(fmt.Sprintf("%s - API key not configured", name))
				continue
			}

			if isDefault {
				fmt.Printf("%s%s:%s: %s %s(default)%s - ✓ Good\n", ui.White, name, provider.Model, provider.BaseURL, ui.Gray, ui.Reset)
			} else {
				ui.PrintWhite(fmt.Sprintf("%s:%s: %s - ✓ Good", name, provider.Model, provider.BaseURL))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
