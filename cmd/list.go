package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
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

		ui.PrintSection("Configured Providers")

		for name, p := range cfg.Providers {
			isDefault := (name == cfg.DefaultProvider)

			// Print provider name
			if isDefault {
				ui.PrintDefault(fmt.Sprintf("✓ %s", name))
			} else {
				ui.PrintWhite(fmt.Sprintf("  %s", name))
			}

			// Handle Native Anthropic providers
			if !providers.RequiresAPIKey(name) {
				def, _ := providers.GetBuiltInProvider(name)
				if isDefault {
					ui.PrintDefault(fmt.Sprintf("    %s", def.Name))
					ui.PrintDefault("    Native Anthropic (no API key required)")
				} else {
					ui.PrintWhite(fmt.Sprintf("    %s", def.Name))
					ui.PrintWhite("    Native Anthropic (no API key required)")
				}
				continue
			}

			// Print base URL
			if p.BaseURL != "" {
				if isDefault {
					ui.PrintDefault(fmt.Sprintf("    %s", p.BaseURL))
				} else {
					ui.PrintWhite(fmt.Sprintf("    %s", p.BaseURL))
				}
			}

			// Print model if available
			if p.Model != "" {
				if isDefault {
					ui.PrintDefault(fmt.Sprintf("    %s", p.Model))
				} else {
					ui.PrintWhite(fmt.Sprintf("    %s", p.Model))
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
