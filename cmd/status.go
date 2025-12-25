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
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if len(cfg.Providers) == 0 {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return
		}

		ui.PrintSection("Provider Status")

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")

		secretsContent := ""
		if _, err := os.Stat(secretsPath); err == nil {
			secretsContent, err = crypto.DecryptSecrets(secretsPath, keyPath)
			if err != nil && verbose {
				ui.PrintInfo(fmt.Sprintf("Warning: Could not decrypt secrets: %v", err))
			}
		}

		for name, provider := range cfg.Providers {
			if name == "anthropic" {
				ui.PrintInfo(fmt.Sprintf("✓ %s", name))
				ui.PrintInfo("    Native Anthropic (no API key required)")
				continue
			}

			if provider.BaseURL == "" {
				ui.PrintWarn(fmt.Sprintf("%s - No base URL configured", name))
				continue
			}

			apiKeyVar := fmt.Sprintf("%s_API_KEY", name)
			hasApiKey := false

			for _, line := range strings.Split(secretsContent, "\n") {
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, apiKeyVar+"=") {
					hasApiKey = true
					break
				}
			}

			if !hasApiKey {
				ui.PrintWarn(fmt.Sprintf("%s - API key not configured", name))
				ui.PrintInfo(fmt.Sprintf("    %s", provider.BaseURL))
				continue
			}

			ui.PrintInfo(fmt.Sprintf("✓ %s", name))
			ui.PrintInfo(fmt.Sprintf("    %s", provider.BaseURL))
		}

		if cfg.DefaultProvider != "" {
			ui.PrintInfo(fmt.Sprintf("Default provider: %s", cfg.DefaultProvider))
		}

		ui.PrintInfo("To configure a provider: kairo config <provider>")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
