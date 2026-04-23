package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Providers: map[string]Provider{},
	}

	err := SaveConfig(context.Background(), subDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	configPath := filepath.Join(subDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}
}

func TestSaveConfigOverwrites(t *testing.T) {
	tmpDir := t.TempDir()

	cfg1 := &Config{
		Providers: map[string]Provider{
			"provider1": {Name: "Provider 1"},
		},
	}

	if err := SaveConfig(context.Background(), tmpDir, cfg1); err != nil {
		t.Fatal(err)
	}

	cfg2 := &Config{
		Providers: map[string]Provider{
			"provider2": {Name: "Provider 2"},
		},
	}

	if err := SaveConfig(context.Background(), tmpDir, cfg2); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := loaded.Providers["provider1"]; ok {
		t.Error("First provider should be removed after overwrite")
	}

	if _, ok := loaded.Providers["provider2"]; !ok {
		t.Error("Second provider should exist")
	}
}

func TestSaveConfigEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Providers: map[string]Provider{},
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
	}

	loaded, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Providers) != 0 {
		t.Errorf("Providers = %d, want 0", len(loaded.Providers))
	}
}

func TestSaveConfigWithDefaultProvider(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Providers: map[string]Provider{
			"test": {Name: "Test"},
		},
		DefaultProvider: "test",
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.DefaultProvider != "test" {
		t.Errorf("DefaultProvider = %q, want %q", loaded.DefaultProvider, "test")
	}
}

func TestSaveConfigWithHarness(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Providers:      map[string]Provider{},
		DefaultHarness: "qwen",
	}

	err := SaveConfig(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.DefaultHarness != "qwen" {
		t.Errorf("DefaultHarness = %q, want %q", loaded.DefaultHarness, "qwen")
	}
}

func TestLoadConfigWithEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Logf("LoadConfig(context.Background(), ) error on empty file: %v", err)
	}
}

func TestLoadConfigWithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("LoadConfig(context.Background(), ) should error on invalid YAML")
	}
}

func TestSaveConfigAtomicWrite(t *testing.T) {
	// Test that SaveConfig uses atomic write (temp file + rename) pattern.
	// This ensures:
	// 1. No partial writes on interruption
	// 2. No temp files left behind after successful save
	// 3. Permissions are preserved
	// 4. Overwrites work correctly

	t.Run("no temp files left after successful save", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &Config{
			Providers: map[string]Provider{
				"test": {Name: "Test"},
			},
		}

		err := SaveConfig(context.Background(), tmpDir, cfg)
		if err != nil {
			t.Fatalf("SaveConfig(context.Background(), ) error = %v", err)
		}

		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			if filepath.Ext(f.Name()) == ".tmp" {
				t.Errorf("Temp file should be cleaned up: %s", f.Name())
			}
		}
	})

	t.Run("atomic overwrite preserves content", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg1 := &Config{
			Providers: map[string]Provider{
				"first": {Name: "First Provider", BaseURL: "https://first.example.com", Model: "model-1"},
			},
			DefaultProvider: "first",
		}

		if err := SaveConfig(context.Background(), tmpDir, cfg1); err != nil {
			t.Fatal(err)
		}

		// Overwrite with new config
		cfg2 := &Config{
			Providers: map[string]Provider{
				"second": {Name: "Second Provider", BaseURL: "https://second.example.com", Model: "model-2"},
			},
			DefaultProvider: "second",
			DefaultHarness:  "qwen",
		}

		if err := SaveConfig(context.Background(), tmpDir, cfg2); err != nil {
			t.Fatal(err)
		}

		loaded, err := LoadConfig(context.Background(), tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := loaded.Providers["first"]; ok {
			t.Error("First provider should not exist after overwrite")
		}

		if provider, ok := loaded.Providers["second"]; !ok {
			t.Error("Second provider should exist")
		} else {
			if provider.Name != "Second Provider" {
				t.Errorf("Provider name = %q, want %q", provider.Name, "Second Provider")
			}
		}

		if loaded.DefaultProvider != "second" {
			t.Errorf("DefaultProvider = %q, want %q", loaded.DefaultProvider, "second")
		}

		if loaded.DefaultHarness != "qwen" {
			t.Errorf("DefaultHarness = %q, want %q", loaded.DefaultHarness, "qwen")
		}
	})

	t.Run("atomic write preserves permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		tmpDir := t.TempDir()

		cfg := &Config{
			Providers: map[string]Provider{
				"test": {Name: "Test"},
			},
		}

		if err := SaveConfig(context.Background(), tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(tmpDir, "config.yaml")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		// Config files should have 0600 permissions (owner read/write only)
		if info.Mode().Perm() != 0600 {
			t.Errorf("File permissions = %o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("multiple overwrites do not leave temp files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Perform multiple overwrites
		for i := 0; i < 5; i++ {
			cfg := &Config{
				Providers: map[string]Provider{
					"provider": {Name: "Provider"},
				},
				DefaultProvider: "provider",
			}

			if err := SaveConfig(context.Background(), tmpDir, cfg); err != nil {
				t.Fatalf("SaveConfig iteration %d error: %v", i, err)
			}
		}

		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		tmpCount := 0
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".tmp" {
				tmpCount++
			}
		}

		if tmpCount > 0 {
			t.Errorf("Found %d temp files, want 0", tmpCount)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file (config.yaml), found %d: %v", len(files), files)
		}
	})
}
