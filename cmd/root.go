package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/pkg/env"
	"github.com/spf13/cobra"
)

var (
	configDir  string
	verbose    bool
	configLock sync.RWMutex
)

func getVerbose() bool {
	configLock.RLock()
	defer configLock.RUnlock()
	return verbose
}

func setVerbose(v bool) {
	configLock.Lock()
	verbose = v
	configLock.Unlock()
}

func setConfigDir(dir string) {
	configLock.Lock()
	configDir = dir
	configLock.Unlock()
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
			if len(args) > 0 {
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

	// Check if to first non-flag argument is a provider name (not a subcommand)
	firstArg := findFirstNonFlagArg(args)
	finalArgs := args

	// Allow Cobra's completion hidden commands to pass through
	if firstArg == "__complete" || firstArg == "__completeNoDesc" {
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
	// Allow Cobra's completion hidden commands
	if name == "__complete" || name == "__completeNoDesc" {
		return true
	}
	return false
}

func init() {
	// NOTE: Cobra's BoolVarP writes directly to verbose variable during flag parsing.
	// This is acceptable because:
	// 1. Flag parsing happens in the main thread before any concurrent access
	// 2. In normal CLI execution, there is no concurrent access to flags
	// 3. Tests should use setVerbose() to avoid potential race conditions
	// 4. Thread-safe access is guaranteed through getVerbose() which uses configLock.RLock()
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default ~/.config/kairo)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func getConfigDir() string {
	configLock.RLock()
	dir := configDir
	configLock.RUnlock()

	if dir != "" {
		return dir
	}
	return env.GetConfigDir()
}

// getConfigDirRaw returns the raw configDir value without fallback logic.
// Useful for testing scenarios where you want to access the variable directly.
// nolint:unused // Reserved for future use and testing scenarios
func getConfigDirRaw() string {
	configLock.RLock()
	defer configLock.RUnlock()
	return configDir
}
