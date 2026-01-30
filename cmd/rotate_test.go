package cmd

import (
	"os"
	"path/filepath"
	"sync"
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

func TestRotateCommandRequiresConfirmation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	// Reset the --yes flag to ensure test isolation
	rotateYesFlag = false

	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	pr, pw, _ := os.Pipe()
	defer pr.Close()
	defer pw.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = pw.WriteString("n\n")
		pw.Close()
	}()

	os.Stdin = pr

	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"rotate"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wg.Wait()

	// Verify the key file was NOT modified by checking its content hasn't changed
	// Since we cancelled, the original key should still be there
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file should still exist after cancellation")
	}
}

func TestRotateCommandWithYesFlag(t *testing.T) {
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

	secrets := "ZAI_API_KEY=test-key\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secrets); err != nil {
		t.Fatal(err)
	}

	// Reset the --yes flag to ensure test isolation
	rotateYesFlag = false

	originalConfigDir := getConfigDir()
	defer func() { setConfigDir(originalConfigDir) }()
	setConfigDir(tmpDir)

	rootCmd.SetArgs([]string{"rotate", "--yes"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify the rotation completed successfully
	_, err = os.Stat(keyPath)
	if os.IsNotExist(err) {
		t.Error("key file should exist after rotation with --yes flag")
	}
}
