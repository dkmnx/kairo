package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [provider]",
	Short: "Remove a provider configuration",
	Long:  "Remove a provider from Kairo. If no provider is specified, shows an interactive list of configured providers.",
	Run: func(cmd *cobra.Command, args []string) {
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
			handleConfigError(cmd, err)
			return
		}

		var target string
		if len(args) == 0 {
			// Interactive selection using tap
			if len(cfg.Providers) == 0 {
				ui.PrintWarn("No providers configured")
				ui.PrintInfo("Run 'kairo setup' to get started")
				return
			}

			providerNames := make([]string, 0, len(cfg.Providers))
			for name := range cfg.Providers {
				providerNames = append(providerNames, name)
			}

			// Convert to tap.SelectOption format
			options := make([]tap.SelectOption[string], len(providerNames))
			for i, name := range providerNames {
				options[i] = tap.SelectOption[string]{Value: name, Label: name}
			}

			fmt.Println()

			tap.Intro("Delete Provider", tap.MessageOptions{
				Hint: "Remove a configured provider from Kairo",
			})

			selected := tap.Select(context.Background(), tap.SelectOptions[string]{
				Message: "Select provider to delete",
				Options: options,
			})
			target = selected
			if target == "" {
				ui.PrintInfo("Operation cancelled")
				return
			}
		} else {
			target = args[0]
		}

		_, ok := cfg.Providers[target]
		if !ok {
			ui.PrintError(fmt.Sprintf("Provider '%s' not configured", target))
			ui.PrintInfo("Run 'kairo list' to see configured providers")
			return
		}

		// Confirmation
		confirmed := tap.Confirm(context.Background(), tap.ConfirmOptions{
			Message: fmt.Sprintf("Are you sure you want to delete '%s'?", target),
		})
		if !confirmed {
			ui.PrintInfo("Operation cancelled")
			return
		}

		delete(cfg.Providers, target)

		if cfg.DefaultProvider == target {
			cfg.DefaultProvider = ""
		}

		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Saving config: %v", err))
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

		tap.Outro(fmt.Sprintf("Provider '%s' deleted successfully", target))

		if err := logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogReset(target)
		}); err != nil {
			ui.PrintWarn(fmt.Sprintf("Audit logging failed: %v", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
