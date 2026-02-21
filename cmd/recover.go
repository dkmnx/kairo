package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/recoveryphrase"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Generate or use recovery phrases for lost keys",
	Long: `Generate recovery phrases to backup your encryption key,
or recover your key from a previously generated phrase.

Warning: Store the recovery phrase securely. Anyone with access
to it can recover your encryption key and access your secrets.`,
}

var generatePhraseCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a recovery phrase for your current key",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		keyPath := dir + "/age.key"
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			// No existing key, generate new phrase
			phrase, err := recoveryphrase.GenerateRecoveryPhrase()
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to generate phrase: %v", err))
				return
			}
			ui.PrintInfo("New recovery phrase (save this!):")
			ui.PrintWarn(phrase)
			return
		}

		phrase, err := recoveryphrase.CreateRecoveryPhrase(keyPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to create phrase: %v", err))
			return
		}

		ui.PrintInfo("Recovery phrase (save this securely!):")
		ui.PrintWarn(phrase)
	},
}

var restorePhraseCmd = &cobra.Command{
	Use:   "restore <phrase>",
	Short: "Restore your encryption key from a recovery phrase",
	Long: `Restore your encryption key from a recovery phrase.
Example: kairo recover restore word1-word2-word3...`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		phrase := strings.TrimSpace(args[0])
		err := recoveryphrase.RecoverFromPhrase(dir, phrase)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to restore: %v", err))
			return
		}

		ui.PrintSuccess("Key restored from recovery phrase")
		ui.PrintWarn("If you have secrets.age, they may need to be re-encrypted with the new key.")
	},
}

func init() {
	recoverCmd.AddCommand(generatePhraseCmd)
	recoverCmd.AddCommand(restorePhraseCmd)
	rootCmd.AddCommand(recoverCmd)
}
