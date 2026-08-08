package cmd

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
)

// failingGenerateKeyService injects a GenerateKey failure to test that
// ResetSecretsFiles leaves the old key and secrets untouched on error.
type failingGenerateKeyService struct {
	crypto.DefaultService
}

func (failingGenerateKeyService) GenerateKey(context.Context, string) error {
	return stderrors.New("injected key generation failure")
}

func TestResetSecretsFiles(t *testing.T) {
	t.Run("deletes old files and regenerates key", func(t *testing.T) {
		tmpDir := t.TempDir()
		cliCtx := NewCLIContext()

		if err := cliCtx.Crypto().EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := cliCtx.Crypto().EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		oldKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("failed to read old key: %v", err)
		}

		if _, err := os.Stat(secretsPath); err != nil {
			t.Fatalf("secrets file should exist before reset: %v", err)
		}

		if err := ResetSecretsFiles(context.Background(), cliCtx, tmpDir, secretsPath, keyPath); err != nil {
			t.Fatalf("ResetSecretsFiles() error = %v", err)
		}

		if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
			t.Error("old secrets file should be deleted after reset")
		}

		newKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("new key file should exist: %v", err)
		}

		if string(oldKeyContent) == string(newKeyContent) {
			t.Error("new key should differ from old key")
		}
	})

	t.Run("succeeds when files do not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		cliCtx := NewCLIContext()
		err := ResetSecretsFiles(context.Background(), cliCtx, tmpDir, secretsPath, keyPath)
		if err != nil {
			t.Fatalf("ResetSecretsFiles() should succeed when files don't exist, got: %v", err)
		}

		if _, err := os.Stat(keyPath); err != nil {
			t.Errorf("new key file should exist after reset: %v", err)
		}
	})

	t.Run("preserves old key and secrets when key generation fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		cliCtx := NewCLIContext()

		if err := cliCtx.Crypto().EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := cliCtx.Crypto().EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		oldKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("failed to read old key: %v", err)
		}

		cliCtx.SetDeps(&Deps{Crypto: failingGenerateKeyService{}})

		err = ResetSecretsFiles(context.Background(), cliCtx, tmpDir, secretsPath, keyPath)
		if err == nil {
			t.Fatal("ResetSecretsFiles() should fail when key generation fails")
		}

		newKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("old key should be preserved: %v", err)
		}
		if string(oldKeyContent) != string(newKeyContent) {
			t.Error("old key was modified despite generation failure")
		}

		if _, err := os.Stat(secretsPath); err != nil {
			t.Errorf("secrets file should be preserved: %v", err)
		}
	})

	t.Run("leaves no temp or backup files after success", func(t *testing.T) {
		tmpDir := t.TempDir()
		cliCtx := NewCLIContext()

		if err := cliCtx.Crypto().EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := cliCtx.Crypto().EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		if err := ResetSecretsFiles(context.Background(), cliCtx, tmpDir, secretsPath, keyPath); err != nil {
			t.Fatalf("ResetSecretsFiles() error = %v", err)
		}

		for _, stray := range []string{keyPath + ".new", keyPath + ".backup"} {
			if _, err := os.Stat(stray); !os.IsNotExist(err) {
				t.Errorf("stray file %s should not exist after reset", stray)
			}
		}
	})
}

func TestEnsureConfigDir(t *testing.T) {
	t.Run("creates directory and key", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, "kairo-config")

		cliCtx := NewCLIContext()
		err := EnsureConfigDir(cliCtx, configDir)
		if err != nil {
			t.Fatalf("EnsureConfigDir() error = %v", err)
		}

		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("config dir should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("config dir should be a directory")
		}

		keyPath := filepath.Join(configDir, constants.KeyFileName)
		if _, err := os.Stat(keyPath); err != nil {
			t.Errorf("encryption key should exist: %v", err)
		}
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		cliCtx := NewCLIContext()
		err := EnsureConfigDir(cliCtx, tmpDir)
		if err != nil {
			t.Fatalf("EnsureConfigDir() should succeed with existing dir, got: %v", err)
		}
	})
}

func TestSaveSecrets(t *testing.T) {
	t.Run("encrypts and saves secrets", func(t *testing.T) {
		tmpDir := t.TempDir()
		cliCtx := NewCLIContext()

		if err := cliCtx.Crypto().EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)
		keyPath := filepath.Join(tmpDir, constants.KeyFileName)

		secrets := map[string]string{
			"ZAI_API_KEY": "sk-test-123",
		}

		err := SaveSecrets(cliCtx, secretsPath, keyPath, secrets)
		if err != nil {
			t.Fatalf("SaveSecrets() error = %v", err)
		}

		if _, err := os.Stat(secretsPath); err != nil {
			t.Errorf("secrets file should exist: %v", err)
		}

		decrypted, err := cliCtx.Crypto().DecryptSecrets(context.Background(), secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets() error = %v", err)
		}

		if decrypted != "ZAI_API_KEY=sk-test-123\n" {
			t.Errorf("decrypted content = %q, want %q", decrypted, "ZAI_API_KEY=sk-test-123\n")
		}
	})

	t.Run("error with invalid key path", func(t *testing.T) {
		cliCtx := NewCLIContext()
		err := SaveSecrets(cliCtx, "/nonexistent/secrets", "/nonexistent/key", map[string]string{"K": "V"})
		if err == nil {
			t.Error("SaveSecrets() should fail with invalid key path")
		}
	})
}
