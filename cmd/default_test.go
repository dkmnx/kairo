package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestDefaultCommandNoArgs(t *testing.T) {
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

	rootCmd.SetArgs([]string{"default"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestDefaultCommandSetProvider(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	t.Logf("tmpDir: %s", tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	configContent := `default_provider: anthropic
providers:
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Before: configDir=%s", configDir)
	rootCmd.SetArgs([]string{"default", "zai"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	t.Logf("After: configDir=%s", configDir)

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	t.Logf("config content: %s", string(content))
}

func TestDefaultCommandProviderNotFound(t *testing.T) {
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

	rootCmd.SetArgs([]string{"default", "nonexistent"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
