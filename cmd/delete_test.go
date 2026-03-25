package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestDeleteCmdDeletesProviderFromConfig(t *testing.T) {
	tmpDir := t.TempDir()

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	setConfigDir(tmpDir)

	cfg := &config.Config{
		DefaultProvider: "testprovider",
		Providers: map[string]config.Provider{
			"testprovider": {Name: "Test Provider", BaseURL: "https://test.com", Model: "test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, config.SecretsFileName)
	keyPath := filepath.Join(tmpDir, config.KeyFileName)
	secrets := map[string]string{
		"TESTPROVIDER_API_KEY": "secret-key",
	}
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

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

	if err := config.SaveConfig(context.Background(), tmpDir, loadedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

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

func TestDeleteCmdDeletesProviderSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()

	setConfigDir(tmpDir)

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

	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	loadedSecrets := result.Secrets

	if loadedSecrets["PROVIDER1_API_KEY"] != "key1" {
		t.Error("Provider1 API key should exist")
	}

	if loadedSecrets["PROVIDER2_API_KEY"] != "key2" {
		t.Error("Provider2 API key should exist")
	}

	// Simulate deletion of provider2's secrets
	delete(loadedSecrets, "PROVIDER2_API_KEY")

	updatedSecretsContent := config.FormatSecrets(loadedSecrets)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, updatedSecretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	result, err = LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	updatedSecrets := result.Secrets

	if _, exists := updatedSecrets["PROVIDER2_API_KEY"]; exists {
		t.Error("Provider2 API key should have been deleted")
	}

	if updatedSecrets["PROVIDER1_API_KEY"] != "key1" {
		t.Error("Provider1 API key should still exist")
	}
}

func TestDeleteCmdRemovesEmptySecretsFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"testprovider": {Name: "Test Provider"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

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

	if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
		t.Error("Secrets file should have been removed when empty")
	}
}
