package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

// requireConfigDir returns the config directory or prints an error and returns empty.
// Use this when you need the config directory for reading.
func requireConfigDir(cmd *cobra.Command) string {
	if cmd == nil {
		// Fall back to default context for backward compatibility
		return defaultCLIContext.GetConfigDir()
	}
	dir := GetCLIContext(cmd).GetConfigDir()
	if dir == "" {
		ui.PrintError("Config directory not found")
	}
	return dir
}

// requireConfigDirWritable is like requireConfigDir but also ensures the directory exists.
// Use this when you need to write to the config directory.
func requireConfigDirWritable(cmd *cobra.Command) string {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return ""
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		ui.PrintError("Error creating config directory: " + err.Error())
		return ""
	}
	return dir
}

// loadConfigOrExit loads the config from the given directory.
// It prints appropriate error messages and returns nil if config cannot be loaded.
// Use when you need a config and want to exit early on error.
func loadConfigOrExit(cmd *cobra.Command) *config.Config {
	dir := requireConfigDir(cmd)
	if dir == "" {
		return nil
	}

	cliCtx := GetCLIContext(cmd)
	cfg, err := cliCtx.GetConfigCache().Get(cliCtx.GetRootCtx(), dir)
	if err != nil {
		if os.IsNotExist(err) {
			ui.PrintWarn("No providers configured")
			ui.PrintInfo("Run 'kairo setup' to get started")
			return nil
		}
		handleConfigError(cmd, err)
		return nil
	}
	return cfg
}

// printSecretsRecoveryHelp prints guidance for recovering from lost secrets.
func printSecretsRecoveryHelp() {
	ui.PrintInfo("Restore 'age.key' and 'secrets.age' from backup,")
	ui.PrintInfo("or remove both files and run 'kairo setup --reset-secrets' to re-enter API keys.")
	ui.PrintInfo("Use --verbose for more details.")
}

// lookPath is the function used to search for executables in PATH.
// It can be replaced in tests to avoid requiring actual executables.
var lookPath = exec.LookPath

// execCommand is the function used to execute external commands.
// It can be replaced in tests to avoid actual process execution.
var execCommand = exec.Command

// execCommandContext is like execCommand but with context for cancellation.
// It can be replaced in tests to avoid actual process execution.
var execCommandContext = exec.CommandContext

// exitProcess is the function used to terminate the process.
// It can be replaced in tests to avoid actual exit calls.
var exitProcess = os.Exit

// parseIntOrZero parses a string to int, returning 0 on invalid input.
func parseIntOrZero(input string) int {
	var result int
	for _, c := range input {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

// runningWithRaceDetector returns true if the race detector is enabled.
func runningWithRaceDetector() bool {
	// Check for -race flag in build flags
	// This is a simple heuristic - actual detection may vary
	return strings.Contains(os.Getenv("GOFLAGS"), "-race")
}

// mergeEnvVars merges and deduplicates environment variables.
// Later values override earlier values with the same key.
// Uses O(n) algorithm with a single pass and final deduplication.
func mergeEnvVars(envs ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			idx := strings.IndexByte(env, '=')
			if idx <= 0 {
				continue
			}
			key := env[:idx]
			seen[key] = true
			result = append(result, env)
		}
	}

	if len(seen) == len(result) {
		return result
	}

	seen = make(map[string]bool)
	var deduped []string
	for i := len(result) - 1; i >= 0; i-- {
		env := result[i]
		idx := strings.IndexByte(env, '=')
		if idx <= 0 {
			continue
		}
		key := env[:idx]
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, env)
		}
	}

	for i, j := 0, len(deduped)-1; i < j; i, j = i+1, j-1 {
		deduped[i], deduped[j] = deduped[j], deduped[i]
	}

	return deduped
}

// setupSignalHandler sets up a signal handler for cleanup.
func setupSignalHandler(cancel func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		signal.Stop(sigChan)
		if cancel != nil {
			cancel()
		}
	}()
}
