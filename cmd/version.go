package cmd

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of Kairo",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintSuccess(fmt.Sprintf("Kairo version: %s", version.Version))
		if version.Commit != "unknown" && version.Commit != "" {
			ui.PrintInfo(fmt.Sprintf("Commit: %s", version.Commit))
		}
		if version.Date != "" {
			ui.PrintInfo(fmt.Sprintf("Date: %s", version.Date))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
