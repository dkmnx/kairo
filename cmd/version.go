package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of Kairo",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kairo version: %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
