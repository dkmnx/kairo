package cmd

import (
	"context"

	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestHarnessGetNoConfig(t *testing.T) {
	originalConfigDir := configDir()
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
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers:      make(map[string]config.Provider),
		DefaultModels:  make(map[string]string),
		DefaultHarness: "qwen",
	}
	err := config.SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	rootCmd.SetArgs([]string{"harness", "get"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestHarnessSetClaude(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "claude"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), %q) error = %v", tmpDir, err)
	}
	if cfg.DefaultHarness != "claude" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "claude")
	}
}

func TestHarnessSetQwen(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "qwen"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}
	if cfg.DefaultHarness != "qwen" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "qwen")
	}
}

func TestHarnessSetInvalid(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	// Pre-create a config so we can verify it wasn't modified
	initialCfg := &config.Config{
		Providers:      map[string]config.Provider{},
		DefaultModels:  map[string]string{},
		DefaultHarness: "claude",
	}
	if err := config.SaveConfig(context.Background(), tmpDir, initialCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	rootCmd.SetArgs([]string{"harness", "set", "invalid"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify invalid harness name was not persisted
	cfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DefaultHarness == "invalid" {
		t.Error("DefaultHarness should not be set to 'invalid'")
	}
}

func TestHarnessSetPi(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "pi"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}
	if cfg.DefaultHarness != "pi" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "pi")
	}
}

func TestHarnessSetCrush(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"harness", "set", "crush"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}
	if cfg.DefaultHarness != "crush" {
		t.Errorf("DefaultHarness = %q, want %q", cfg.DefaultHarness, "crush")
	}
}

func TestGetHarnessWithPi(t *testing.T) {
	tests := []struct {
		name          string
		flagHarness   string
		configHarness string
		expected      string
	}{
		{"flag pi takes precedence over config claude", "pi", "claude", "pi"},
		{"config pi used when flag empty", "", "pi", "pi"},
		{"flag pi takes precedence over config qwen", "pi", "qwen", "pi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveHarness(tt.flagHarness, tt.configHarness)
			if result != tt.expected {
				t.Errorf("resolveHarness() = %q, want %q", result, tt.expected)
			}
		})
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
		{"uppercase PI", "PI", "pi"},
		{"uppercase CRUSH", "CRUSH", "crush"},
		{"mixed case Claude", "Claude", "claude"},
		{"mixed case Qwen", "Qwen", "qwen"},
		{"mixed case Pi", "Pi", "pi"},
		{"mixed case Crush", "Crush", "crush"},
		{"lowercase claude", "claude", "claude"},
		{"lowercase qwen", "qwen", "qwen"},
		{"lowercase pi", "pi", "pi"},
		{"lowercase crush", "crush", "crush"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfigDir := configDir()
			defer func() { setConfigDir(originalConfigDir) }()

			tmpDir := t.TempDir()
			setConfigDir(tmpDir)

			rootCmd.SetArgs([]string{"harness", "set", tt.harnessName})
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			cfg, err := config.LoadConfig(context.Background(), tmpDir)
			if err != nil {
				t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
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
		{"flag pi takes precedence", "pi", "claude", "pi"},
		{"config pi used", "", "pi", "pi"},
		{"flag crush takes precedence", "crush", "claude", "crush"},
		{"config crush used", "", "crush", "crush"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveHarness(tt.flagHarness, tt.configHarness)
			if result != tt.expected {
				t.Errorf("resolveHarness() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetHarnessWithExistingConfig(t *testing.T) {
	originalConfigDir := configDir()
	defer func() { setConfigDir(originalConfigDir) }()

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers:      make(map[string]config.Provider),
		DefaultModels:  make(map[string]string),
		DefaultHarness: "qwen",
	}
	err := config.SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig(context.Background(), ) error = %v", err)
	}

	result := resolveHarness("", loadedCfg.DefaultHarness)
	if result != "qwen" {
		t.Errorf("resolveHarness() = %q, want %q", result, "qwen")
	}
}
