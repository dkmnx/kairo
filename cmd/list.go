package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Long:  "Display all configured providers and their status",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			ui.PrintInfo("Run 'kairo setup' to configure providers")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintWarn("No providers configured")
				ui.PrintInfo("Run 'kairo setup' to get started")
				return
			}
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if len(cfg.Providers) == 0 {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return
		}

		ui.PrintHeader("Configured Providers")
		fmt.Println()

		for name, p := range cfg.Providers {
			if name == cfg.DefaultProvider {
				fmt.Printf("  %s%s%s %s %s\n", ui.Green, "✓", ui.Reset, name, p.BaseURL)
			} else {
				fmt.Printf("  %s %s %s\n", " ", name, p.BaseURL)
			}
		}

		if cfg.DefaultProvider != "" {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("Default provider: %s", cfg.DefaultProvider))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
