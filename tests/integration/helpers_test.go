package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var testBinary string

func TestMain(m *testing.M) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find project root: %v\n", err)
		os.Exit(1)
	}

	tmpDir, err := os.MkdirTemp("", "kairo-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

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

	exitCode := m.Run()
	os.Exit(exitCode)
}

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
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
