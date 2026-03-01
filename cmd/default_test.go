package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestDefaultCommandNoArgs(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
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
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	t.Logf("tmpDir: %s", tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
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

	t.Logf("Before: configDir=%s", getConfigDir())
	rootCmd.SetArgs([]string{"default", "zai"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	t.Logf("After: configDir=%s", getConfigDir())

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	t.Logf("config content: %s", string(content))
}

func TestDefaultCommandProviderNotFound(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
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

func TestDefaultCommandUpdatesConfigFile(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
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

	rootCmd.SetArgs([]string{"default", "zai"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify the config file was updated correctly
	configContentStr := string(content)
	if !containsString(configContentStr, "default_provider: zai") {
		t.Error("config file should contain 'default_provider: zai'")
	}
	if containsString(configContentStr, "default_provider: anthropic") {
		t.Error("config file should not contain 'default_provider: anthropic'")
	}
}

