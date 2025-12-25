package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var defaultCmd = &cobra.Command{
	Use:   "default [provider]",
	Short: "Get or set the default provider",
	Long:  "With no arguments, shows the current default provider. With a provider name, sets it as the default.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				if len(args) == 0 {
					ui.PrintWarn("No default provider configured")
				} else {
					ui.PrintError(fmt.Sprintf("Provider '%s' not found in config", args[0]))
				}
				return
			}
			ui.PrintError(fmt.Sprintf("Error loading config: %v", err))
			return
		}

		if len(args) == 0 {
			if cfg.DefaultProvider == "" {
				ui.PrintWarn("No default provider configured")
				ui.PrintInfo("Run 'kairo default <provider>' to set one")
			} else {
				ui.PrintSuccess(fmt.Sprintf("Default provider: %s", cfg.DefaultProvider))
			}
			return
		}

		providerName := args[0]
		if _, ok := cfg.Providers[providerName]; !ok {
			ui.PrintError(fmt.Sprintf("Provider '%s' not configured", providerName))
			ui.PrintInfo("Run 'kairo config " + providerName + "' to configure")
			return
		}

		cfg.DefaultProvider = providerName
		if err := config.SaveConfig(dir, cfg); err != nil {
			ui.PrintError(fmt.Sprintf("Error saving config: %v", err))
			return
		}

		ui.PrintSuccess(fmt.Sprintf("Default provider set to: %s", providerName))
	},
}

func init() {
	rootCmd.AddCommand(defaultCmd)
}
