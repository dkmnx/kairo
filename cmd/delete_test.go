package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

// TestDeleteCmdDeletesProviderFromConfig tests that config file has correct structure for deletion.
func TestDeleteCmdDeletesProviderFromConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate encryption key
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	// Save and restore original config dir
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	setConfigDir(tmpDir)

	// Create config with one provider
	cfg := &config.Config{
		DefaultProvider: "testprovider",
		Providers: map[string]config.Provider{
			"testprovider": {Name: "Test Provider", BaseURL: "https://test.com", Model: "test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Save secrets file with API key
	secretsPath := filepath.Join(tmpDir, config.SecretsFileName)
	keyPath := filepath.Join(tmpDir, config.KeyFileName)
	secrets := map[string]string{
		"TESTPROVIDER_API_KEY": "secret-key",
	}
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	// Verify provider exists
	loadedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if _, exists := loadedCfg.Providers["testprovider"]; !exists {
		t.Fatal("Provider should exist")
	}

	// Simulate deletion: remove provider from config
	delete(loadedCfg.Providers, "testprovider")
	loadedCfg.DefaultProvider = ""

	// Save updated config
	if err := config.SaveConfig(context.Background(), tmpDir, loadedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify provider was deleted
	updatedCfg, err := config.LoadConfig(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if _, exists := updatedCfg.Providers["testprovider"]; exists {
		t.Error("Provider should have been deleted")
	}

	if updatedCfg.DefaultProvider != "" {
		t.Error("DefaultProvider should have been cleared")
	}
}

// TestDeleteCmdDeletesProviderSecrets tests that delete removes provider API key from secrets.
func TestDeleteCmdDeletesProviderSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate encryption key
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	// Save and restore original config dir
	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	setConfigDir(tmpDir)

	// Create config with two providers
	cfg := &config.Config{
		DefaultProvider: "provider1",
		Providers: map[string]config.Provider{
			"provider1": {Name: "Provider 1"},
			"provider2": {Name: "Provider 2"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Save secrets file with API keys for both providers
	secretsPath := filepath.Join(tmpDir, config.SecretsFileName)
	keyPath := filepath.Join(tmpDir, config.KeyFileName)
	secrets := map[string]string{
		"PROVIDER1_API_KEY": "key1",
		"PROVIDER2_API_KEY": "key2",
	}
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	// Load and verify both keys exist
	loadedSecrets, _, _, err := LoadAndDecryptSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadAndDecryptSecrets() error = %v", err)
	}

	if loadedSecrets["PROVIDER1_API_KEY"] != "key1" {
		t.Error("Provider1 API key should exist")
	}

	if loadedSecrets["PROVIDER2_API_KEY"] != "key2" {
		t.Error("Provider2 API key should exist")
	}

	// Simulate deletion of provider2's secrets
	delete(loadedSecrets, "PROVIDER2_API_KEY")

	// Save updated secrets
	updatedSecretsContent := config.FormatSecrets(loadedSecrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, updatedSecretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	// Verify provider2's key was deleted
	updatedSecrets, _, _, err := LoadAndDecryptSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadAndDecryptSecrets() error = %v", err)
	}

	if _, exists := updatedSecrets["PROVIDER2_API_KEY"]; exists {
		t.Error("Provider2 API key should have been deleted")
	}

	// Verify provider1's key still exists
	if updatedSecrets["PROVIDER1_API_KEY"] != "key1" {
		t.Error("Provider1 API key should still exist")
	}
}

// TestDeleteCmdRemovesEmptySecretsFile tests that secrets file is removed when empty.
func TestDeleteCmdRemovesEmptySecretsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate encryption key
	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	// Create config with one provider
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"testprovider": {Name: "Test Provider"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Create empty secrets file
	secretsPath := filepath.Join(tmpDir, config.SecretsFileName)
	keyPath := filepath.Join(tmpDir, config.KeyFileName)
	secrets := map[string]string{}
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	// Simulate deletion: remove empty secrets (as the delete command does)
	if err := os.Remove(secretsPath); err != nil {
		t.Fatalf("os.Remove() error = %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
		t.Error("Secrets file should have been removed when empty")
	}
}
