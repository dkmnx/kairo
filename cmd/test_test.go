package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTestCommandNoConfig(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestTestCommandProviderNotFound(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

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
