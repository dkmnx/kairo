package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/pkg/env"
	"github.com/spf13/cobra"
)

var (
	configDir string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "kairo",
	Short: "Kairo - Manage Claude Code API providers",
	Long: fmt.Sprintf(`Kairo is a CLI tool for managing Claude Code API providers with 
encrypted secrets management using age encryption.

Version: %s (commit: %s, date: %s)`, kairoversion.Version, kairoversion.Commit, kairoversion.Date),
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			cmd.Println("Error: config directory not found")
			_ = cmd.Help() // Ignoring error - Help() rarely fails and we're exiting anyway
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

// getConfigDir returns the configuration directory path.
// It uses the --config flag if provided, otherwise falls back to the default.
func getConfigDir() string {
	if configDir != "" {
		return configDir
	}
	return env.GetConfigDir()
}
