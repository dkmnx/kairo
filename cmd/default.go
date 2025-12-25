package cmd

import (
	"os"

	"github.com/dkmnx/kairo/internal/config"
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
			cmd.Println("Error: config directory not found")
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				if len(args) == 0 {
					cmd.Println("No default provider configured")
				} else {
					cmd.Printf("Error: provider '%s' not found in config\n", args[0])
				}
				return
			}
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		if len(args) == 0 {
			if cfg.DefaultProvider == "" {
				cmd.Println("No default provider configured")
			} else {
				cmd.Printf("Default provider: %s\n", cfg.DefaultProvider)
			}
			return
		}

		providerName := args[0]
		if _, ok := cfg.Providers[providerName]; !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			return
		}

		cfg.DefaultProvider = providerName
		if err := config.SaveConfig(dir, cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}

		cmd.Printf("Default provider set to: %s\n", providerName)
	},
}

func init() {
	rootCmd.AddCommand(defaultCmd)
}
