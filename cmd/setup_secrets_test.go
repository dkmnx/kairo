package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	secretspkg "github.com/dkmnx/kairo/internal/secrets"
)

func TestParseSecretsForIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
			"minimax": {Name: "MiniMax"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	secretsContent := "ZAI_API_KEY=zai-key\nMINIMAX_API_KEY=minimax-key\nDEEPSEEK_API_KEY=deepseek-key\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	secretsMap := secretspkg.Parse(decrypted)

	if len(secretsMap) != 3 {
		t.Errorf("ParseSecrets() returned %d entries, want 3", len(secretsMap))
	}

	expectedKeys := []string{"ZAI_API_KEY", "MINIMAX_API_KEY", "DEEPSEEK_API_KEY"}
	for _, key := range expectedKeys {
		if _, ok := secretsMap[key]; !ok {
			t.Errorf("ParseSecrets() missing key %q", key)
		}
	}
}

func TestSecretsPreservationWhenAddingProvider(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
			"minimax": {Name: "MiniMax"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	existingSecrets := "ZAI_API_KEY=zai-secret-123\nMINIMAX_API_KEY=minimax-secret-456\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, existingSecrets); err != nil {
		t.Fatal(err)
	}

	secretsContent, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	secretsMap := secretspkg.Parse(secretsContent)
	if len(secretsMap) != 2 {
		t.Errorf("ParseSecrets() returned %d entries, want 2", len(secretsMap))
	}

	newApiKey := "deepseek-secret-789"
	secretsMap["DEEPSEEK_API_KEY"] = newApiKey

	var secretsBuilder strings.Builder
	keys := make([]string, 0, len(secretsMap))
	for key := range secretsMap {
		keys = append(keys, key)
	}
	for _, key := range keys {
		value := secretsMap[key]
		if key != "" && value != "" {
			secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	secretsMap = secretspkg.Parse(decrypted)
	if len(secretsMap) != 3 {
		t.Errorf("After adding provider, expected 3 secrets, got %d", len(secretsMap))
	}

	if secretsMap["ZAI_API_KEY"] != "zai-secret-123" {
		t.Errorf("ZAI_API_KEY was lost, got %q", secretsMap["ZAI_API_KEY"])
	}
	if secretsMap["MINIMAX_API_KEY"] != "minimax-secret-456" {
		t.Errorf("MINIMAX_API_KEY was lost, got %q", secretsMap["MINIMAX_API_KEY"])
	}
	if secretsMap["DEEPSEEK_API_KEY"] != "deepseek-secret-789" {
		t.Errorf("DEEPSEEK_API_KEY not saved correctly, got %q", secretsMap["DEEPSEEK_API_KEY"])
	}
}

func TestLoadSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	secretsOut := result.SecretsPath
	keyOut := result.KeyPath
	secrets := result.Secrets
	if secretsOut != secretsPath {
		t.Errorf("secretsPath = %q, want %q", secretsOut, secretsPath)
	}
	if keyOut != keyPath {
		t.Errorf("keyPath = %q, want %q", keyOut, keyPath)
	}
	if secrets["ZAI_API_KEY"] != "test-key" {
		t.Errorf("ZAI_API_KEY = %q, want %q", secrets["ZAI_API_KEY"], "test-key")
	}
}

func TestLoadSecretsNoSecretsFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	secretsPath := result.SecretsPath
	keyPath := result.KeyPath
	secrets := result.Secrets
	if len(secrets) != 0 {
		t.Errorf("got %d secrets, want 0", len(secrets))
	}
	if !strings.HasSuffix(secretsPath, "secrets.age") {
		t.Errorf("secretsPath = %q, expected to end with secrets.age", secretsPath)
	}
	if !strings.HasSuffix(keyPath, "age.key") {
		t.Errorf("keyPath = %q, expected to end with age.key", keyPath)
	}
}

func TestLoadSecretsWithCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(secretsPath, []byte("corrupted invalid encrypted data"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSecrets(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("Expected error for corrupted secrets file, got nil")
	}
}

func TestLoadSecretsWithCorruptedKey(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath, []byte("invalid-key-content"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSecrets(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("Expected error for corrupted key file, got nil")
	}
}
