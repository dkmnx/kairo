package cmd

import (
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

var (
	configDir string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: `Kairo is a CLI tool for managing Claude Code API providers with 
encrypted secrets management using age encryption.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			_ = cmd.Help()
			return
		}

		cfg, err := config.LoadConfig(dir)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Println("No providers configured. Run 'kairo setup' to get started.")
				return
			}
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		if cfg.DefaultProvider == "" {
			cmd.Println("No default provider set.")
			cmd.Println()
			cmd.Println("Usage:")
			cmd.Println("  kairo setup        # Configure providers")
			cmd.Println("  kairo default <provider>  # Set default provider")
			cmd.Println("  kairo <provider> [args]   # Switch and run Claude")
			return
		}

		switchCmd.Run(cmd, append([]string{cfg.DefaultProvider}, args...))
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default ~/.config/kairo)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func getConfigDir() string {
	if configDir != "" {
		return configDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.config/kairo"
}
