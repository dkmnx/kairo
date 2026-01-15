package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/dkmnx/kairo/internal/audit"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// rotateYesFlag is used by Cobra for flag binding
	rotateYesFlag bool
	// rotateYes provides atomic access for thread safety
	rotateYes atomic.Bool
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate encryption key",
	Long: `Generate a new encryption key and re-encrypt all secrets.

This command is a security best practice that should be performed periodically.
It generates a new age X25519 key and re-encrypts all stored API keys with it.

The old key is replaced with the new key. All secrets remain accessible
after the rotation completes.

Examples:
  kairo rotate`,
	Run: func(cmd *cobra.Command, args []string) {
		// Sync flag value to atomic variable
		rotateYes.Store(rotateYesFlag)

		dir := getConfigDir()
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				ui.PrintError("Cannot find home directory")
				return
			}
			if runtime.GOOS == "windows" {
				dir = filepath.Join(home, "AppData", "Roaming", "kairo")
			} else {
				dir = filepath.Join(home, ".config", "kairo")
			}
		}

		if !rotateYes.Load() {
			ui.PrintWarn("This will rotate your encryption key and re-encrypt all secrets.")
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

		cmd.Printf("Rotating encryption key in %s...\n", dir)

		if err := crypto.RotateKey(dir); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to rotate key: %v", err))
			return
		}

		ui.PrintSuccess("Encryption key rotated successfully")

		logAuditEvent(dir, func(logger *audit.Logger) error {
			return logger.LogRotate("all")
		})
	},
}

func init() {
	rotateCmd.Flags().BoolVar(&rotateYesFlag, "yes", false, "Skip confirmation prompt")
	rootCmd.AddCommand(rotateCmd)
}
