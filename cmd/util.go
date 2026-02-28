package cmd

import (
	"os"
	"os/exec"
	"strings"
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
