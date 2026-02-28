package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// lookPath is the function used to search for executables in PATH.
// It can be replaced in tests to avoid requiring actual executables.
var lookPath = exec.LookPath

// execCommand is the function used to execute external commands.
// It can be replaced in tests to avoid actual process execution.
var execCommand = exec.Command

// exitProcess is the function used to terminate the process.
// It can be replaced in tests to avoid actual exit calls.
var exitProcess = os.Exit

// parseIntOrZero parses a string to int, returning 0 on invalid input.
func parseIntOrZero(s string) int {
	var result int
	for _, c := range s {
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
func mergeEnvVars(envs ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			// Extract key (everything before first '=')
			idx := strings.IndexByte(env, '=')
			if idx <= 0 {
				// Invalid format, skip
				continue
			}
			key := env[:idx]

			// Remove any previous occurrence of this key
			if seen[key] {
				// Find and remove previous entry with this key
				for i, e := range result {
					if strings.HasPrefix(e, key+"=") {
						result = append(result[:i], result[i+1:]...)
						break
					}
				}
			}

			// Add new entry
			result = append(result, env)
			seen[key] = true
		}
	}

	return result
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
