package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var resetYes bool

var resetCmd = &cobra.Command{
	Use:   "reset <provider | all>",
	Short: "Reset provider configuration",
	Long:  "Remove a provider's configuration. Use 'all' to reset all providers.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintWarn("No providers configured")
				return
			}
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if target == "all" {
			if !resetYes {
				ui.PrintWarn("This will remove ALL provider configurations and secrets.")
				if !ui.Confirm("Do you want to proceed?") {
					ui.PrintInfo("Operation cancelled")
					return
				}
			}

			for name := range cfg.Providers {
				delete(cfg.Providers, name)
			}
			cfg.DefaultProvider = ""

			if err := config.SaveConfig(dir, cfg); err != nil {
				ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
				return
			}

			secretsPath := filepath.Join(dir, "secrets.age")

			_, err := os.Stat(secretsPath)
			if err == nil {
				err := os.Remove(secretsPath)
				if err != nil {
					ui.PrintWarn(fmt.Sprintf("Warning: Could not remove secrets file: %v", err))
				}
			}

			ui.PrintSuccess("All providers reset successfully")

			logAuditEvent(dir, func(logger *audit.Logger) error {
				return logger.LogReset("all")
			})
			return
		}

		_, ok := cfg.Providers[target]
		if !ok {
			ui.PrintError(fmt.Sprintf("Provider '%s' not configured", target))
			ui.PrintInfo("Run 'kairo list' to see configured providers")
			return
		}

		delete(cfg.Providers, target)

		if cfg.DefaultProvider == target {
			cfg.DefaultProvider = ""
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		secretsPath := filepath.Join(dir, "secrets.age")
		keyPath := filepath.Join(dir, "age.key")

		existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			secrets := config.ParseSecrets(existingSecrets)
			delete(secrets, fmt.Sprintf("%s_API_KEY", target))

			var secretsContent string
			for key, value := range secrets {
				if key != "" && value != "" {
					secretsContent += fmt.Sprintf("%s=%s\n", key, value)
				}
			}

			if secretsContent == "" {
				err := os.Remove(secretsPath)
				if err != nil {
					ui.PrintWarn(fmt.Sprintf("Warning: Could not remove empty secrets file: %v", err))
				}
			} else {
				if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
					ui.PrintWarn(fmt.Sprintf("Warning: Could not update secrets: %v", err))
				}
			}
		}

		ui.PrintSuccess(fmt.Sprintf("Provider '%s' reset successfully", target))

		logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogReset(target)
		})
	},
}

func init() {
	resetCmd.Flags().BoolVar(&resetYes, "yes", false, "Skip confirmation prompt")
	rootCmd.AddCommand(resetCmd)
}
