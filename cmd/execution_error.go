package cmd

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/dkmnx/kairo/internal/constants"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func yoloModeFlag(h string) string {
	return harness.YoloFlag(h)
}
func handleConfigError(cmd *cobra.Command, err error) {
	if isBinaryOutdatedError(err) {
		promptUpgrade(cmd, err)

		return
	}
	cmd.Printf("Error loading config: %v\n", err)
}

func isBinaryOutdatedError(err error) bool {
	var typeErr *yaml.TypeError

	return errors.Is(err, kairoerrors.ErrBinaryOutdated) || errors.As(err, &typeErr)
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
	if verbose() {
		cmd.Printf("Technical details: %v\n", err)
	}
}

func handleSecretsError(err error) {
	ui.PrintError(fmt.Sprintf("Failed to decrypt secrets file: %v", err))
	printSecretsRecoveryHelp()
}
