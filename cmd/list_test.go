package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestListCommandNoConfig(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	listCmd.SetErr(buf)

	err := listCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestListCommandWithConfig(t *testing.T) {
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

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	listCmd.SetErr(buf)

	err = listCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
