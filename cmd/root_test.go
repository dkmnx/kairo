package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCommandNoArgsWithDefault(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

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
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
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
