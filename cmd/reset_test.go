package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
)

func TestResetCommandNoConfig(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"reset"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("Execute() should return an error for missing args")
	}
}

func TestResetCommandSingleProvider(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: Minimax-M2.1
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"reset", "zai"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if _, ok := cfg.Providers["zai"]; ok {
		t.Error("zai provider should be removed")
	}

	if _, ok := cfg.Providers["minimax"]; !ok {
		t.Error("minimax provider should remain")
	}
}

func TestResetCommandAllProviders(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: Minimax-M2.1
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"reset", "all", "--yes"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(cfg.Providers))
	}
}

func TestResetCommandNonexistentProvider(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"reset", "nonexistent"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(cfg.Providers))
	}
}

func TestResetCommandAllRequiresConfirmation(t *testing.T) {
	setupMockExec(t)
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	configPath := filepath.Join(tmpDir, "config")
	configContent := `providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Reset the --yes flag to ensure test isolation
	resetYes = false

	// Simulate user input "n" for no confirmation
	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	pr, pw, _ := os.Pipe()
	defer pr.Close()
	defer pw.Close()

	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = pw.WriteString("n\n")
		pw.Close()
	}()

	os.Stdin = pr

	rootCmd.SetArgs([]string{"reset", "all"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify providers still exist (operation was cancelled)
	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Providers) == 0 {
		t.Error("expected providers to remain after cancellation, but all were removed")
	}

	if _, ok := cfg.Providers["zai"]; !ok {
		t.Error("zai provider should still exist after cancellation")
	}
}
