package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		cliCtx := GetCLIContext(cmd)
		dir := requireConfigDir(cmd)
		if dir == "" {
			return
		}

		cfg, err := config.LoadConfig(cliCtx.GetRootCtx(), dir)
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
			if len(cfg.Providers) == 0 {
				ui.PrintWarn("No providers configured")
				ui.PrintInfo("Run 'kairo setup' to get started")
				return
			}

			providerNames := make([]string, 0, len(cfg.Providers))
			for name := range cfg.Providers {
				providerNames = append(providerNames, name)
			}

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

		if err := config.SaveConfig(cliCtx.GetRootCtx(), dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Saving config: %v", err))
			return
		}

		cliCtx.InvalidateCache(dir)

		secretsPath := filepath.Join(dir, config.SecretsFileName)
		keyPath := filepath.Join(dir, config.KeyFileName)

		if err := deleteProviderSecrets(cliCtx.GetRootCtx(), secretsPath, keyPath, target); err != nil {
			ui.PrintWarn(fmt.Sprintf("Warning: %v", err))
		}

		tap.Outro(fmt.Sprintf("Provider '%s' deleted successfully", target))
	},
}

func deleteProviderSecrets(ctx context.Context, secretsPath, keyPath, providerName string) error {
	existingSecrets, err := crypto.DecryptSecretsBytes(ctx, secretsPath, keyPath)
	if err != nil {
		return nil
	}
	defer crypto.ClearMemory(existingSecrets)

	parsed := config.ParseSecretsWithStats(string(existingSecrets))
	if parsed.SkippedCount > 0 {
		ui.PrintWarn(fmt.Sprintf(
			"Warning: %d malformed secret entries were skipped during parsing",
			parsed.SkippedCount))
	}

	apiKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
	delete(parsed.Secrets, apiKey)

	secretsContent := config.FormatSecrets(parsed.Secrets)

	if secretsContent == "" {
		if removeErr := os.Remove(secretsPath); removeErr != nil {
			return fmt.Errorf("could not remove empty secrets file: %w", removeErr)
		}
		return nil
	}

	if err := crypto.EncryptSecrets(ctx, secretsPath, keyPath, secretsContent); err != nil {
		return fmt.Errorf("could not update secrets: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
