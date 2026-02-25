package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	resetYesFlag bool
)

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

		cfg, err := configCache.Get(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintWarn("No providers configured")
				return
			}
			handleConfigError(cmd, err)
			return
		}

		if target == "all" {
			if !resetYesFlag {
				ui.PrintWarn("This will remove ALL provider configurations and secrets.")
				confirmed, err := ui.Confirm("Do you want to proceed?")
				if err != nil {
					ui.PrintError(fmt.Sprintf("Failed to read input: %v", err))
					return
				}
				if !confirmed {
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
			keyPath := filepath.Join(dir, "age.key")

			_, err := os.Stat(secretsPath)
			if err == nil {
				err := os.Remove(secretsPath)
				if err != nil {
					ui.PrintWarn(fmt.Sprintf("Warning: Could not remove secrets file: %v", err))
				}
			}

			_, err = os.Stat(keyPath)
			if err == nil {
				err := os.Remove(keyPath)
				if err != nil {
					ui.PrintWarn(fmt.Sprintf("Warning: Could not remove key file: %v", err))
				}
			}

			ui.PrintSuccess("All providers reset successfully")

			if err := logAuditEvent(dir, func(logger *audit.Logger) error {
				return logger.LogReset("all")
			}); err != nil {
				ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
			}
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
			delete(secrets, fmt.Sprintf("%s_API_KEY", strings.ToUpper(target)))

			secretsContent := config.FormatSecrets(secrets)

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

		if err := logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogReset(target)
		}); err != nil {
			ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
		}
	},
}

func init() {
	resetCmd.Flags().BoolVar(&resetYesFlag, "yes", false, "Skip confirmation prompt")
	rootCmd.AddCommand(resetCmd)
}
