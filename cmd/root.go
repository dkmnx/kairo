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

func getVerbose() bool {
	return verbose
}

func setVerbose(v bool) {
	verbose = v
}

func setConfigDir(dir string) {
	configDir = dir
}

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
			// If args provided, try to use the first arg as provider name
			if len(args) > 0 {
				// Let switchCmd.Run handle provider validation and errors
				switchCmd.Run(cmd, args)
				return
			}

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
	args := os.Args[1:]

	// Check if the first non-flag argument is a provider name (not a subcommand)
	firstArg := findFirstNonFlagArg(args)
	finalArgs := args

	// Allow Cobra's completion hidden commands to pass through
	if firstArg == "__complete" || firstArg == "__completeNoDesc" {
		// Do nothing, let Cobra handle completion
	} else if firstArg != "" && !isKnownSubcommand(firstArg) {
		// This looks like a provider name - convert to switch command
		// Let switchCmd handle validation and error messages
		finalArgs = append([]string{"switch"}, args...)
	}

	// Clean up args after execution to prevent test pollution
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs(finalArgs)
	return rootCmd.Execute()
}

// findFirstNonFlagArg returns the first argument that's not a flag
func findFirstNonFlagArg(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Skip flags
		if len(arg) > 0 && arg[0] == '-' {
			// Skip flag value if it's a separate argument
			if arg == "--config" && i+1 < len(args) {
				i++
			}
			continue
		}
		return arg
	}
	return ""
}

// isKnownSubcommand checks if the given name is a known subcommand
func isKnownSubcommand(name string) bool {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return true
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return true
			}
		}
	}
	// Allow Cobra's completion hidden commands to pass through
	if name == "__complete" || name == "__completeNoDesc" {
		return true
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func getConfigDir() string {
	if configDir != "" {
		return configDir
	}
	return env.GetConfigDir()
}
