package cmd

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of Kairo",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintSuccess(fmt.Sprintf("Kairo version: %s", version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
