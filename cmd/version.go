package cmd

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = ""
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of Kairo",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintSuccess(fmt.Sprintf("Kairo version: %s", version))
		if commit != "unknown" && commit != "" {
			ui.PrintInfo(fmt.Sprintf("Commit: %s", commit))
		}
		if date != "" {
			ui.PrintInfo(fmt.Sprintf("Date: %s", date))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
