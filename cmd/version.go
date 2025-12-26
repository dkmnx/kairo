package cmd

import (
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
		if version.Date != "" {
			cmd.Printf("Date: %s\n", version.Date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
