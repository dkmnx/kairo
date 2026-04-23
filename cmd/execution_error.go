package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

const harnessQwen = "qwen"
const claudeYoloFlag = "--dangerously-skip-permissions"

func yoloModeFlag(harness string) string {
	if harness == harnessQwen {
		return "--yolo"
	}

	return claudeYoloFlag
}

func handleConfigError(cmd *cobra.Command, err error) {
	errStr := err.Error()
	if isOutdatedBinaryError(errStr) {
		promptUpgrade(cmd, err)

		return
	}
	cmd.Printf("Error loading config: %v\n", err)
}

func isOutdatedBinaryError(errStr string) bool {
	return (strings.Contains(errStr, "field") && strings.Contains(errStr, "not found in type")) ||
		strings.Contains(errStr, "configuration file contains field(s) not recognized") ||
		strings.Contains(errStr, "your installed kairo binary is outdated")
}

func promptUpgrade(cmd *cobra.Command, err error) {
	cmd.Println("Error: Your kairo binary is outdated and cannot read your configuration file.")
	cmd.Println()
	cmd.Println("The configuration file contains newer fields that this version doesn't recognize.")
	cmd.Println()
	cmd.Println("How to fix:")
	cmd.Println("  Run the installation script for your platform:")
	cmd.Println()

	switch runtime.GOOS {
	case constants.WindowsGOOS:
		cmd.Println("    irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex")
	default:
		cmd.Println("    curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh")
	}

	cmd.Println()
	cmd.Println("  For manual installation, see:")
	cmd.Println("    https://github.com/dkmnx/kairo/blob/main/docs/guides/user-guide.md#manual-installation")
	cmd.Println()
	if getVerbose() {
		cmd.Printf("Technical details: %v\n", err)
	}
}

func handleSecretsError(err error) {
	ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
	printSecretsRecoveryHelp()
}
