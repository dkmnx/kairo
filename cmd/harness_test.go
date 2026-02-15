package cmd

import (
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestHarnessGetNoConfig(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "get"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestHarnessGetWithConfig(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers:      make(map[string]config.Provider),
		DefaultModels:  make(map[string]string),
		DefaultHarness: "qwen",
	}
	err := config.SaveConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	rootCmd.SetArgs([]string{"harness", "get"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestHarnessSetClaude(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "claude"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DefaultHarness != "claude" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "claude")
	}
}

func TestHarnessSetQwen(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "qwen"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DefaultHarness != "qwen" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "qwen")
	}
}

func TestHarnessSetInvalid(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "invalid"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestHarnessSetCaseInsensitive(t *testing.T) {
	tests := []struct {
		name        string
		harnessName string
		expected    string
	}{
		{"uppercase CLAUDE", "CLAUDE", "claude"},
		{"uppercase QWEN", "QWEN", "qwen"},
		{"mixed case Claude", "Claude", "claude"},
		{"mixed case Qwen", "Qwen", "qwen"},
		{"lowercase claude", "claude", "claude"},
		{"lowercase qwen", "qwen", "qwen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfigDir := getConfigDir()
			defer func() { setConfigDir(originalConfigDir) }()

			tmpDir := t.TempDir()
			setConfigDir(tmpDir)

			rootCmd.SetArgs([]string{"harness", "set", tt.harnessName})
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			cfg, err := config.LoadConfig(tmpDir)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if cfg.DefaultHarness != tt.expected {
				t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, tt.expected)
			}
		})
	}
}

func TestGetHarness(t *testing.T) {
	tests := []struct {
		name          string
		flagHarness   string
		configHarness string
		expected      string
	}{
		{"flag takes precedence", "qwen", "claude", "qwen"},
		{"uses config when flag empty", "", "qwen", "qwen"},
		{"defaults to claude when both empty", "", "", "claude"},
		{"defaults to claude when config invalid", "", "invalid", "claude"},
		{"defaults to claude when flag invalid", "invalid", "", "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHarness(tt.flagHarness, tt.configHarness)
			if result != tt.expected {
				t.Errorf("getHarness() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetHarnessBinary(t *testing.T) {
	tests := []struct {
		name     string
		harness  string
		expected string
	}{
		{"claude returns claude", "claude", "claude"},
		{"qwen returns qwen", "qwen", "qwen"},
		{"unknown returns claude", "unknown", "claude"},
		{"empty returns claude", "", "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHarnessBinary(tt.harness)
			if result != tt.expected {
				t.Errorf("getHarnessBinary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetHarnessWithExistingConfig(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers:      make(map[string]config.Provider),
		DefaultModels:  make(map[string]string),
		DefaultHarness: "qwen",
	}
	err := config.SaveConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	result := getHarness("", loadedCfg.DefaultHarness)
	if result != "qwen" {
		t.Errorf("getHarness() = %q, want %q", result, "qwen")
	}
}
