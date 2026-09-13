package crypto

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultService_GenerateKey(t *testing.T) {
	svc := DefaultService{}
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	if err := svc.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key file: %v", err)
	}
	if len(data) == 0 {
		t.Error("key file is empty")
	}
}

func TestDefaultService_EncryptDecryptSecretsRoundtrip(t *testing.T) {
	svc := DefaultService{}
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := svc.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secrets := "ZAI_API_KEY=sk-test-key\nMINIMAX_API_KEY=sk-another-key\n"

	if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, secrets); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	decrypted, err := svc.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}
	if decrypted != secrets {
		t.Errorf("DecryptSecrets() = %q, want %q", decrypted, secrets)
	}

	raw, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("reading encrypted file: %v", err)
	}
	if len(raw) == 0 || string(raw) == secrets {
		t.Error("secrets file should contain ciphertext, not plaintext")
	}
}

func TestDefaultService_DecryptSecretsBytes(t *testing.T) {
	svc := DefaultService{}
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := svc.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secrets := "ZAI_API_KEY=sk-test-key\n"
	if err := svc.EncryptSecrets(context.Background(), secretsPath, keyPath, secrets); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	got, err := svc.DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes() error = %v", err)
	}
	if string(got) != secrets {
		t.Errorf("DecryptSecretsBytes() = %q, want %q", string(got), secrets)
	}
}

func TestDefaultService_DecryptSecretsErrors(t *testing.T) {
	svc := DefaultService{}
	tmpDir := t.TempDir()

	if _, err := svc.DecryptSecrets(context.Background(), filepath.Join(tmpDir, "missing.age"), filepath.Join(tmpDir, "missing.key")); err == nil {
		t.Error("DecryptSecrets() should error when key file is missing")
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := svc.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if _, err := svc.DecryptSecrets(context.Background(), filepath.Join(tmpDir, "missing.age"), keyPath); err == nil {
		t.Error("DecryptSecrets() should error when secrets file is missing")
	}
}

func TestDefaultService_EnsureKeyExists(t *testing.T) {
	svc := DefaultService{}
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() on empty dir error = %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("EnsureKeyExists() should create key file: %v", err)
	}

	first, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key file: %v", err)
	}

	if err := svc.EnsureKeyExists(context.Background(), tmpDir); err != nil {
		t.Fatalf("EnsureKeyExists() on existing key error = %v", err)
	}

	second, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key file after second call: %v", err)
	}
	if string(first) != string(second) {
		t.Error("EnsureKeyExists() should not regenerate an existing key")
	}
}
