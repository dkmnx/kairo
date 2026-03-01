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
