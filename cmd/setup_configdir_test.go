package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestGetConfigDirWithEnv(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	// Reset configDir to empty so testCLI.ConfigDir() falls back to env.GetConfigDir()
	originalConfigDir := testCLI.ConfigDir()
	defer testCLI.SetConfigDir(originalConfigDir)
	testCLI.SetConfigDir("")

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(tmpDir, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(tmpDir, ".config", "kairo")
	}
	dir := testCLI.ConfigDir()
	if dir != expectedDir {
		t.Errorf("testCLI.ConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestGetConfigDirWithFlag(t *testing.T) {
	originalConfigDir := testCLI.ConfigDir()
	testCLI.SetConfigDir("/custom/path")
	defer testCLI.SetConfigDir(originalConfigDir)

	dir := testCLI.ConfigDir()
	if dir != "/custom/path" {
		t.Errorf("testCLI.ConfigDir() = %q, want %q", dir, "/custom/path")
	}
}

func TestGetConfigDirWithFlagAndEnv(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalConfigDir := testCLI.ConfigDir()
	defer func() {
		os.Setenv("HOME", originalHome)
		testCLI.SetConfigDir(originalConfigDir)
	}()

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	testCLI.SetConfigDir("/custom/path")

	dir := testCLI.ConfigDir()
	if dir != "/custom/path" {
		t.Errorf("testCLI.ConfigDir() = %q, want %q (flag should take precedence)", dir, "/custom/path")
	}
}

func TestGetConfigDirEmptyConfigDir(t *testing.T) {
	originalConfigDir := testCLI.ConfigDir()
	testCLI.SetConfigDir("")
	defer testCLI.SetConfigDir(originalConfigDir)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home directory")
	}

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(home, ".config", "kairo")
	}
	dir := testCLI.ConfigDir()
	if dir != expectedDir {
		t.Errorf("testCLI.ConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestEnsureConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cliCtx := NewCLIContext()
	err := EnsureConfigDir(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("EnsureConfigDir() error = %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("age.key was not created: %v", err)
	}
}

func TestLoadOrInitializeConfigExisting(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
		DefaultProvider: "zai",
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "zai")
	}
	if _, ok := loadedCfg.Providers["zai"]; !ok {
		t.Errorf("Provider zai not found in loaded config")
	}
}

func TestLoadOrInitializeConfigNew(t *testing.T) {
	tmpDir := t.TempDir()

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("LoadConfig() returned nil for non-existent config, want empty config")
	}
	if loadedCfg.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty string", loadedCfg.DefaultProvider)
	}
	if loadedCfg.Providers == nil {
		t.Error("Providers map is nil, want empty map")
	}
}

func TestLoadOrInitializeConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"test": {Name: "Test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if loadedCfg.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty", loadedCfg.DefaultProvider)
	}
}

func TestEnsureConfigDirectory_ErrorPaths(t *testing.T) {
	t.Run("invalid path with permission issue", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "notadir")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Try to create directory inside a file path (should fail)
		invalidPath := filepath.Join(tmpFile.Name(), "config")
		cliCtx := NewCLIContext()
		err = EnsureConfigDir(cliCtx, invalidPath)
		if err == nil {
			t.Error("expected error for invalid config directory path")
		}
	})
}
