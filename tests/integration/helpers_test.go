package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// testBinary holds the path to the test binary built by TestMain.
// It is shared across all integration tests to avoid rebuilding.
var testBinary string

// TestMain is the entry point for all integration tests.
// It builds the kairo binary once and makes it available to all tests
// via the testBinary variable. This significantly speeds up test execution
// by avoiding redundant builds.
//
// The binary is automatically cleaned up after all tests complete.
func TestMain(m *testing.M) {
	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find project root: %v\n", err)
		os.Exit(1)
	}

	// Create temporary directory for test binary
	tmpDir, err := os.MkdirTemp("", "kairo-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build test binary
	testBinary = filepath.Join(tmpDir, "kairo_test")
	if runtime.GOOS == "windows" {
		testBinary += ".exe"
	}

	os.Stderr.WriteString("Building test binary...\n")
	cmd := exec.Command("go", "build", "-o", testBinary, ".")
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\noutput: %s\n", err, string(output))
		os.Exit(1)
	}

	// Run tests
	exitCode := m.Run()

	// Cleanup is handled by defer os.RemoveAll(tmpDir)
	os.Exit(exitCode)
}

// findProjectRoot locates the project root by searching for go.mod.
// It starts from the current working directory and searches upward
// through parent directories until it finds go.mod or reaches the
// filesystem root.
//
// Returns:
//   - string: Absolute path to the project root directory
//   - error: os.ErrNotExist if go.mod is not found, or path-related errors
//
// This function is shared across all integration tests to avoid duplication.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding go.mod
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
