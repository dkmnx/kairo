package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func setupMockExec(t *testing.T) {
	originalExecCommand := execCommand
	originalExitProcess := exitProcess
	t.Cleanup(func() {
		execCommand = originalExecCommand
		exitProcess = originalExitProcess
	})

	// Mock exitProcess to prevent test termination
	exitProcess = func(code int) {}
}

func TestTestCommandNoConfig(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestTestCommandProviderNotFound(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	configPath := filepath.Join(tmpDir, "config")
	configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	testCmd.SetArgs([]string{"zai"})
	err = testCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestSwitchCommandNoConfig(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	switchCmd.SetArgs([]string{"zai"})
	err := switchCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestSwitchCommandProviderNotFound(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := configDir
	t.Cleanup(func() { configDir = originalConfigDir })

	tmpDir := t.TempDir()
	configDir = tmpDir

	configPath := filepath.Join(tmpDir, "config")
	configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	switchCmd.SetArgs([]string{"zai"})
	err = switchCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
