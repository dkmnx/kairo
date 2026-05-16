package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

const claudeYoloFlag = "--dangerously-skip-permissions"
const qwenYoloFlag = "--yolo"

func yoloModeFlag(harness string) string {
	if harness == harnessQwen {
		return qwenYoloFlag
	}
	if harness == harnessPi {
		return ""
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
		cmd.Printf("    irm %s | iex\n", constants.RawGitHubFileURL("main", "scripts/install.ps1"))
	default:
		cmd.Printf("    curl -sSL %s | sh\n", constants.RawGitHubFileURL("main", "scripts/install.sh"))
	}

	cmd.Println()
	cmd.Println("  For manual installation, see:")
	cmd.Printf("    %s#manual-installation\n", constants.GitHubBlobURL("main", "docs/guides/user-guide.md"))
	cmd.Println()
	if getVerbose() {
		cmd.Printf("Technical details: %v\n", err)
	}
}

func handleSecretsError(err error) {
	ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
	printSecretsRecoveryHelp()
}
