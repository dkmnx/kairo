package cmd

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/backup"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of your configuration and secrets",
	Long: `Create a timestamped backup of your kairo configuration,
encryption key, and encrypted secrets.

Backups are stored in: ~/.config/kairo/backups/`,
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		backupPath, err := backup.CreateBackup(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to create backup: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Backup created: %s", backupPath))
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup file",
	Long: `Restore your kairo configuration from a backup file.
Warning: This will overwrite your current configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		err := backup.RestoreBackup(dir, args[0])
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to restore backup: %v", err))
			return
		}

		ui.PrintSuccess("Backup restored successfully")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
