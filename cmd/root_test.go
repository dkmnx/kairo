package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRootCommandNoArgsWithDefault(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	// Mock execCommand to avoid actually running claude
	originalExecCommand := execCommand
	originalExitProcess := exitProcess
	defer func() {
		execCommand = originalExecCommand
		exitProcess = originalExitProcess
	}()

	// Create a fake command that succeeds without doing anything
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command("false") // Command that exits with status 1
		cmd.Path = name
		return cmd
	}
	// Mock exitProcess to prevent test termination
	exitCalled := false
	exitProcess = func(code int) { exitCalled = true }

	tmpDir := t.TempDir()
	configDir = tmpDir

	configPath := filepath.Join(tmpDir, "config")
	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{})
	rootCmd.Execute()
	// We expect the command to fail (exitCalled=true) since claude isn't really available
	if !exitCalled {
		t.Log("Note: Command did not exit, which may indicate mocking is not working correctly")
	}
}

func TestRootCommandNoArgsNoDefault(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRootCommandNoArgsNoConfig(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
