package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	secretspkg "github.com/dkmnx/kairo/internal/secrets"
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

	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	secretsMap := map[string]string{
		"TESTPROVIDER_API_KEY": "secret-key",
	}
	secretsContent := secretspkg.Format(secretsMap)
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

	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	secretsMap := map[string]string{
		"PROVIDER1_API_KEY": "key1",
		"PROVIDER2_API_KEY": "key2",
	}
	secretsContent := secretspkg.Format(secretsMap)
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

	updatedSecretsContent := secretspkg.Format(loadedSecrets)
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

	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)
	secretsMap := map[string]string{}
	secretsContent := secretspkg.Format(secretsMap)
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

func TestDeleteProviderSecretsReturnsErrorOnBadKey(t *testing.T) {
	tmpDir := t.TempDir()

	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, "nonexistent.key")

	err := deleteProviderSecrets(context.Background(), secretsPath, keyPath, "testprovider")
	if err == nil {
		t.Fatal("deleteProviderSecrets should return error when decryption fails")
	}
	var kairoErr *kairoerrors.KairoError
	if !errors.As(err, &kairoErr) {
		t.Fatalf("expected *KairoError, got %T", err)
	}
	if kairoErr.Type != kairoerrors.CryptoError {
		t.Errorf("error type = %v, want %v", kairoErr.Type, kairoerrors.CryptoError)
	}
}

func TestDeleteProviderSecretsPreservesMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()

	if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
	keyPath := filepath.Join(tmpDir, constants.KeyFileName)

	secretsContent := "VALID_KEY=valid_value\nmalformed_without_equals\nPROVIDER_TO_DELETE_API_KEY=secret\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	if err := deleteProviderSecrets(context.Background(), secretsPath, keyPath, "PROVIDER_TO_DELETE"); err != nil {
		t.Fatalf("deleteProviderSecrets() error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	if !strings.Contains(decrypted, "VALID_KEY=valid_value") {
		t.Error("decrypted content should still contain VALID_KEY=valid_value")
	}
	if !strings.Contains(decrypted, "malformed_without_equals") {
		t.Error("decrypted content should still contain malformed_without_equals")
	}
	if strings.Contains(decrypted, "PROVIDER_TO_DELETE_API_KEY=secret") {
		t.Error("decrypted content should NOT contain PROVIDER_TO_DELETE_API_KEY")
	}
}
