package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/update"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kairo to the latest version",
	Long: `Check for a new release and update kairo to the latest version.

This command will:
1. Check GitHub for the latest release
2. Download the install script and its SHA256 checksum
3. Verify script integrity before execution
4. Run the verified install script

Security considerations:
- The install script is downloaded from the specific release tag to ensure
  the script matches the version being installed.
- SHA256 checksums are verified before execution to ensure script integrity.
- You will be prompted for confirmation before installation.
- The script is executed with your current user permissions.

For manual verification, you can download and inspect the install script and checksums from:
https://github.com/dkmnx/kairo/blob/<tag>/scripts/install.sh (Unix)
https://github.com/dkmnx/kairo/blob/<tag>/scripts/install.ps1 (Windows)
https://github.com/dkmnx/kairo/blob/<tag>/scripts/checksums.txt`,
	Run: func(cmd *cobra.Command, args []string) {
		deps := CLIContextFromCmd(cmd).Deps()

		currentVersion := version.Version
		if currentVersion == "dev" {
			cmd.Println("Cannot update development version")

			return
		}

		latest, err := deps.Update.GetLatestRelease()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error checking for updates: %v", err))

			return
		}

		if !update.VersionGreaterThan(currentVersion, latest.TagName) {
			cmd.Printf("You are already on the latest version: %s\n", currentVersion)

			return
		}

		cmd.Printf("Updating to %s...\n", latest.TagName)

		installScriptURL := update.GetInstallScriptURL(runtime.GOOS, latest.TagName)

		confirmed, err := deps.Update.ConfirmUpdate("Do you want to proceed with installation?")
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error reading input: %v", err))

			return
		}
		if !confirmed {
			cmd.Println("Installation canceled.")

			return
		}

		cmd.Printf("\nDownloading install script from: %s\n", installScriptURL)

		tempFile, err := deps.Update.DownloadToTempFile(installScriptURL)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error downloading install script: %v", err))

			return
		}
		defer os.Remove(tempFile)

		scriptName := update.GetScriptNameForChecksums(runtime.GOOS)
		checksumsURL := update.GetChecksumsURL(latest.TagName)

		cmd.Printf("Downloading checksums from: %s\n", checksumsURL)

		checksums, err := deps.Update.DownloadAndParseChecksums(checksumsURL)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Error downloading checksums: %v", err))

			return
		}

		expectedHash, ok := checksums[scriptName]
		if !ok {
			ui.PrintError(fmt.Sprintf("Checksum for %s not found in checksums file", scriptName))

			return
		}

		cmd.Printf("Verifying script integrity...\n")

		if err := deps.Update.VerifyCosignBundle(latest.TagName); err != nil {
			cmd.Printf("Warning: cosign verification skipped or failed: %v\n", err)
		}

		if err := deps.Update.VerifyChecksum(tempFile, expectedHash); err != nil {
			ui.PrintError(fmt.Sprintf("Security verification failed: %v", err))
			cmd.Println("Downloaded script has been removed. Please try again later or report this issue.")

			return
		}

		cmd.Printf("Running install script...\n\n")

		if err := deps.Update.RunInstallScript(tempFile); err != nil {
			ui.PrintError(fmt.Sprintf("Error during installation: %v", err))

			return
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
