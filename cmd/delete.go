package cmd

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/secrets"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
	"github.com/yarlson/tap"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [provider]",
	Short: "Remove a provider configuration",
	Long:  "Remove a provider from Kairo. If no provider is specified, shows an interactive list of configured providers.",
	Run: func(cmd *cobra.Command, args []string) {
		cliCtx := CLIContextFromCmd(cmd)
		dir := requireConfigDir(cmd)
		if dir == "" {
			return
		}

		cfg, err := config.LoadConfig(cliCtx.RootCtx(), dir)
		if err != nil {
			if stderrors.Is(err, fs.ErrNotExist) {
				printNoProvidersMessage()

				return
			}
			handleConfigError(cmd, err)

			return
		}

		var target string
		if len(args) == 0 {
			if len(cfg.Providers) == 0 {
				printNoProvidersMessage()

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

			selected := tap.Select(cliCtx.RootCtx(), tap.SelectOptions[string]{
				Message: "Select provider to delete",
				Options: options,
			})
			target = selected
			if target == "" {
				tap.Cancel("Operation canceled")

				return
			}
		} else {
			target = args[0]
		}

		_, ok := cfg.Providers[target]
		if !ok {
			tap.Cancel(fmt.Sprintf("Provider '%s' not configured", target))

			return
		}

		confirmed := tap.Confirm(cliCtx.RootCtx(), tap.ConfirmOptions{
			Message: fmt.Sprintf("Are you sure you want to delete '%s'?", target),
		})
		if !confirmed {
			tap.Cancel("Operation canceled")

			return
		}

		delete(cfg.Providers, target)

		if cfg.DefaultProvider == target {
			cfg.DefaultProvider = ""
		}

		if err := config.SaveConfig(cliCtx.RootCtx(), dir, cfg); err != nil {
			tap.Cancel(fmt.Sprintf("Saving config: %v", err))

			return
		}

		cliCtx.InvalidateCache(dir)

		secretsPath := filepath.Join(dir, constants.SecretsFileName)
		keyPath := filepath.Join(dir, constants.KeyFileName)

		if err := deleteProviderSecrets(cliCtx.RootCtx(), cliCtx.Crypto(), secretsPath, keyPath, target); err != nil {
			tap.Cancel(fmt.Sprintf("Failed to clean up secrets for '%s': %v", target, err))

			return
		}

		tap.Outro(fmt.Sprintf("Provider '%s' deleted successfully", target))
	},
}

func deleteProviderSecrets(ctx context.Context, svc crypto.Service, secretsPath, keyPath, providerName string) error {
	existingSecrets, err := svc.DecryptSecretsBytes(ctx, secretsPath, keyPath)
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to decrypt secrets for cleanup", err).
			WithContext("provider", providerName)
	}
	defer crypto.ClearMemory(existingSecrets)

	parsed := secrets.ParseWithStats(string(existingSecrets))

	apiKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
	delete(parsed.Secrets, apiKey)

	secretsContent := secrets.Format(parsed.Secrets)
	if len(parsed.RawLines) > 0 {
		secretsContent += strings.Join(parsed.RawLines, "\n") + "\n"
		ui.PrintWarn(
			fmt.Sprintf("%d malformed entries preserved (unparseable)",
				len(parsed.RawLines),
			),
		)
	}

	if secretsContent == "" {
		if removeErr := os.Remove(secretsPath); removeErr != nil {
			return errors.WrapError(errors.FileSystemError,
				"could not remove empty secrets file", removeErr).
				WithContext("path", secretsPath)
		}

		return nil
	}

	if err := svc.EncryptSecrets(ctx, secretsPath, keyPath, secretsContent); err != nil {
		return errors.WrapError(errors.CryptoError,
			"could not update secrets", err).
			WithContext("path", secretsPath)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
