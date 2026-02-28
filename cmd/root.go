// Package cmd implements the Kairo CLI application using the Cobra framework.
//
// Architecture:
//   - Commands are defined in individual files (root.go, setup.go, switch.go, etc.)
//   - Global state (configDir, verbose) is managed via getter/setter functions
//   - Command execution is orchestrated by rootCmd.Execute()
//
// Testing:
//   - Most commands have corresponding *_test.go files
//   - Integration tests verify end-to-end workflows
//   - External process execution can be mocked via execCommand variable
//
// Design principles:
//   - Minimal business logic in command handlers
//   - Delegation to internal packages for core functionality
//   - Consistent error handling with user-friendly messages
//
// Security:
//   - All user input is read securely using ui package
//   - No secrets are logged to stdout/stderr
//   - API keys are managed via encrypted secrets file
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	kairoversion "github.com/dkmnx/kairo/internal/version"
	"github.com/dkmnx/kairo/pkg/env"
	"github.com/spf13/cobra"
)

var (
	configDir   string
	configDirMu sync.RWMutex // Protects configDir
	verbose     bool
	verboseMu   sync.RWMutex // Protects verbose
	configCache *config.ConfigCache
)

func getVerbose() bool {
	verboseMu.RLock()
	defer verboseMu.RUnlock()
	return verbose
}

func setVerbose(v bool) {
	verboseMu.Lock()
	defer verboseMu.Unlock()
	verbose = v
}

func setConfigDir(dir string) {
	configDirMu.Lock()
	defer configDirMu.Unlock()
	configDir = dir
}

func getConfigDir() string {
	configDirMu.RLock()
	defer configDirMu.RUnlock()
	if configDir != "" {
		return configDir
	}
	return env.GetConfigDir()
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
			_ = cmd.Help()
			return
		}

		cfg, err := configCache.Get(dir)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Println("No providers configured. Run 'kairo setup' to get started.")
				return
			}
			handleConfigError(cmd, err)
			return
		}

		if len(cfg.Providers) == 0 {
			cmd.Println("No providers configured. Run 'kairo setup' to get started.")
			return
		}

		// If no arguments, list providers
		if len(args) == 0 {
			if cfg.DefaultProvider == "" {
				cmd.Println("No default provider set.")
				cmd.Println()
				cmd.Println("Usage:")
				cmd.Println("  kairo setup            # Configure providers")
				cmd.Println("  kairo edit <provider> # Configure a provider")
				cmd.Println("  kairo list             # List providers")
				cmd.Println("  kairo <provider>       # Use specific provider")
				return
			}
			// For now, just show help message
			cmd.Printf("Default provider: %s\n", cfg.DefaultProvider)
			cmd.Println("Usage:")
			cmd.Println("  kairo <provider> [args]  # Use specific provider")
			return
		}

		// First argument should be a provider name
		providerName := args[0]
		provider, ok := cfg.Providers[providerName]
		if !ok {
			cmd.Printf("Error: provider '%s' not configured\n", providerName)
			cmd.Println("Run 'kairo list' to see configured providers")
			return
		}

		// Execute with the specified provider
		cmd.Printf("Provider '%s' (%s) - execution not yet implemented\n", providerName, provider.Name)
	},
}

// Execute runs the kairo CLI application.
// It processes command-line arguments and executes the appropriate Cobra command.
// Returns an error if command execution fails.
func Execute() error {
	args := os.Args[1:]

	// Clean up args after execution to prevent test pollution
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func init() {
	// Initialize cache with 5 minute TTL
	configCache = config.NewConfigCache(5 * time.Minute)

	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default is platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

// handleConfigError provides user-friendly guidance for config errors.
func handleConfigError(cmd *cobra.Command, err error) {
	errStr := err.Error()

	// Check for unknown field error (outdated binary)
	// This can appear in two forms:
	// 1. Raw YAML error: "field X not found in type config.Config"
	// 2. Wrapped error: "configuration file contains field(s) not recognized"
	if (strings.Contains(errStr, "field") && strings.Contains(errStr, "not found in type")) ||
		strings.Contains(errStr, "configuration file contains field(s) not recognized") ||
		strings.Contains(errStr, "your installed kairo binary is outdated") {
		cmd.Println("Error: Your kairo binary is outdated and cannot read your configuration file.")
		cmd.Println()
		cmd.Println("The configuration file contains newer fields that this version doesn't recognize.")
		cmd.Println()
		cmd.Println("How to fix:")
		cmd.Println("  Run the installation script for your platform:")
		cmd.Println()

		// Display platform-specific installation script
		switch runtime.GOOS {
		case "windows":
			cmd.Println("    irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex")
		default: // linux, darwin (macOS)
			cmd.Println("    curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh")
		}

		cmd.Println()
		cmd.Println("  For manual installation, see:")
		cmd.Println("    https://github.com/dkmnx/kairo/blob/main/docs/guides/user-guide.md#manual-installation")
		cmd.Println()
		if verbose {
			cmd.Printf("Technical details: %v\n", err)
		}
		return
	}

	// Default error handling
	cmd.Printf("Error loading config: %v\n", err)
}
