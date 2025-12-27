package cmd

import (
	"time"

	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of Kairo",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("Kairo version: %s\n", version.Version)
		if version.Commit != "unknown" && version.Commit != "" {
			cmd.Printf("Commit: %s\n", version.Commit)
		}
		if version.Date != "" && version.Date != "unknown" {
			if t, err := time.Parse(time.RFC3339, version.Date); err == nil {
				cmd.Printf("Date: %s\n", t.Format("2006-01-02"))
			} else {
				cmd.Printf("Date: %s\n", version.Date)
			}
		}

		if version.Version != "dev" {
			checkForUpdates(cmd)
		}
	},
}

func checkForUpdates(cmd *cobra.Command) {
	latest, err := getLatestRelease()
	if err != nil {
		return
	}

	if versionGreaterThan(version.Version, latest.TagName) {
		cmd.Println()
		cmd.Printf("A new version is available: %s\n", latest.TagName)
		cmd.Printf("You are currently on: %s\n", version.Version)
		cmd.Println()
		cmd.Println("To update, run:")
		cmd.Println("  kairo update")
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func createVersionCommand() *cobra.Command {
	return versionCmd
}
