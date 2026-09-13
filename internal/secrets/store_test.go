package secrets

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
)

type failingGenerateKeyService struct {
	crypto.DefaultService
}

func (failingGenerateKeyService) GenerateKey(context.Context, string) error {
	return stderrors.New("injected key generation failure")
}

func TestReset(t *testing.T) {
	t.Run("deletes old files and regenerates key", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		oldKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("failed to read old key: %v", err)
		}

		if err := Reset(context.Background(), svc, tmpDir, secretsPath, keyPath); err != nil {
			t.Fatalf("Reset() error = %v", err)
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

		if err := Reset(context.Background(), crypto.DefaultService{}, tmpDir, secretsPath, keyPath); err != nil {
			t.Fatalf("Reset() should succeed when files don't exist, got: %v", err)
		}

		if _, err := os.Stat(keyPath); err != nil {
			t.Errorf("new key file should exist after reset: %v", err)
		}
	})

	t.Run("preserves old key and secrets when key generation fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		oldKeyContent, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("failed to read old key: %v", err)
		}

		err = Reset(context.Background(), failingGenerateKeyService{}, tmpDir, secretsPath, keyPath)
		if err == nil {
			t.Fatal("Reset() should fail when key generation fails")
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
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		keyPath := filepath.Join(tmpDir, constants.KeyFileName)
		secretsPath := filepath.Join(tmpDir, constants.SecretsFileName)

		if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, "TEST_KEY=value\n"); err != nil {
			t.Fatalf("EncryptSecrets() error = %v", err)
		}

		if err := Reset(context.Background(), svc, tmpDir, secretsPath, keyPath); err != nil {
			t.Fatalf("Reset() error = %v", err)
		}

		for _, stray := range []string{keyPath + ".new", keyPath + ".backup"} {
			if _, err := os.Stat(stray); !os.IsNotExist(err) {
				t.Errorf("stray file %s should not exist after reset", stray)
			}
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("encrypts and saves secrets", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		secretsPath, keyPath := Paths(tmpDir)

		err := Save(context.Background(), svc, secretsPath, keyPath, map[string]string{
			"ZAI_API_KEY": "sk-test-123",
		})
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if _, err := os.Stat(secretsPath); err != nil {
			t.Errorf("secrets file should exist: %v", err)
		}

		decrypted, err := svc.DecryptSecrets(context.Background(), secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets() error = %v", err)
		}

		if decrypted != "ZAI_API_KEY=sk-test-123\n" {
			t.Errorf("decrypted content = %q, want %q", decrypted, "ZAI_API_KEY=sk-test-123\n")
		}
	})

	t.Run("error with invalid key path", func(t *testing.T) {
		err := Save(context.Background(), crypto.DefaultService{},
			"/nonexistent/secrets", "/nonexistent/key", map[string]string{"K": "V"})
		if err == nil {
			t.Error("Save() should fail with invalid key path")
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("decrypts existing store", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatal(err)
		}
		secretsPath, keyPath := Paths(tmpDir)
		if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		result, err := Load(context.Background(), svc, tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if result.SecretsPath != secretsPath || result.KeyPath != keyPath {
			t.Errorf("paths = %q/%q, want %q/%q",
				result.SecretsPath, result.KeyPath, secretsPath, keyPath)
		}
		if result.Secrets["ZAI_API_KEY"] != "test-key" {
			t.Errorf("ZAI_API_KEY = %q, want %q", result.Secrets["ZAI_API_KEY"], "test-key")
		}
	})

	t.Run("missing secrets file is empty success", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := Load(context.Background(), crypto.DefaultService{}, tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(result.Secrets) != 0 {
			t.Errorf("got %d secrets, want 0", len(result.Secrets))
		}
		if result.SecretsPath == "" || result.KeyPath == "" {
			t.Error("paths should be set even when the store is missing")
		}
	})

	t.Run("corrupted secrets file keeps paths on error", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := crypto.DefaultService{}

		if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatal(err)
		}
		secretsPath, _ := Paths(tmpDir)
		if err := os.WriteFile(secretsPath, []byte("corrupted invalid encrypted data"), 0o600); err != nil {
			t.Fatal(err)
		}

		result, err := Load(context.Background(), svc, tmpDir)
		if err == nil {
			t.Fatal("expected error for corrupted secrets file")
		}
		if result.SecretsPath != secretsPath {
			t.Errorf("paths must survive decrypt failure for --reset-secrets, got %q", result.SecretsPath)
		}
	})
}
