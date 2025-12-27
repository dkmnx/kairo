package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestRotateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	originalSecrets := "ZAI_API_KEY=test-key\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, originalSecrets); err != nil {
		t.Fatal(err)
	}

	err := crypto.RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	if decrypted != originalSecrets {
		t.Errorf("decrypted = %q, want %q", decrypted, originalSecrets)
	}
}

func TestRotateCommandNoSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	err := crypto.RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file should still exist")
	}
}

func TestRotateCommandPreservesProviders(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
			"minimax": {Name: "MiniMax"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	secrets := "ZAI_API_KEY=zai-key\nMINIMAX_API_KEY=minimax-key\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secrets); err != nil {
		t.Fatal(err)
	}

	if err := crypto.RotateKey(tmpDir); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "zai")
	}

	if _, ok := loadedCfg.Providers["zai"]; !ok {
		t.Error("zai provider missing")
	}

	if _, ok := loadedCfg.Providers["minimax"]; !ok {
		t.Error("minimax provider missing")
	}
}
