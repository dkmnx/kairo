package crypto

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(ctx, keyPath)
	if err == nil {
		t.Error("GenerateKey with canceled context should return error")
	}

	if _, statErr := os.Stat(keyPath); statErr == nil {
		t.Error("key file should not be created with canceled context")
	}
}

func TestEncryptSecrets_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	err := EncryptSecrets(ctx, secretsPath, keyPath, "test=secret")
	if err == nil {
		t.Error("EncryptSecrets with canceled context should return error")
	}
}

func TestDecryptSecrets_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	_, err := DecryptSecrets(ctx, secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets with canceled context should return error")
	}
}

func TestDecryptSecretsBytes_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	_, err := DecryptSecretsBytes(ctx, secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecretsBytes with canceled context should return error")
	}
}

func TestEnsureKeyExists_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpDir := t.TempDir()
	err := EnsureKeyExists(ctx, tmpDir)
	if err == nil {
		t.Error("EnsureKeyExists with canceled context should return error")
	}
}

func TestLoadRecipient_MalformedRecipient(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	content := "AGE-SECRET-KEY-VALIDLINE\nnot-a-valid-recipient\n"
	if err := os.WriteFile(keyPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadRecipient(keyPath)
	if err == nil {
		t.Error("loadRecipient with malformed recipient should return error")
	}
}

func TestEncryptDecrypt_EmptySecrets(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	err := EncryptSecrets(context.Background(), secretsPath, keyPath, "")
	if err != nil {
		t.Fatalf("EncryptSecrets with empty string should succeed, got: %v", err)
	}

	decrypted, err := DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets should succeed, got: %v", err)
	}
	if decrypted != "" {
		t.Errorf("decrypted = %q, want empty string", decrypted)
	}
}

func TestEncryptDecrypt_LargeSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	largeSecret := ""
	for i := 0; i < 10000; i++ {
		largeSecret += "API_KEY_XYZ=test_value_with_long_content_1234567890\n"
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	err := EncryptSecrets(context.Background(), secretsPath, keyPath, largeSecret)
	if err != nil {
		t.Fatalf("EncryptSecrets with large payload should succeed, got: %v", err)
	}

	decrypted, err := DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets should succeed, got: %v", err)
	}
	if decrypted != largeSecret {
		t.Error("decrypted content should match original large payload")
	}
}

func TestEncryptDecrypt_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	secrets := "API_KEY=key-with-$pecial-chars!@#$%^&*()\nANOTHER=\"quoted 'value'\""
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	err := EncryptSecrets(context.Background(), secretsPath, keyPath, secrets)
	if err != nil {
		t.Fatalf("EncryptSecrets should succeed, got: %v", err)
	}

	decrypted, err := DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets should succeed, got: %v", err)
	}
	if decrypted != secrets {
		t.Errorf("decrypted mismatch: got %q, want %q", decrypted, secrets)
	}
}

func TestDecryptSecrets_WrongKey(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath1 := filepath.Join(tmpDir, "key1.age")
	keyPath2 := filepath.Join(tmpDir, "key2.age")
	if err := GenerateKey(context.Background(), keyPath1); err != nil {
		t.Fatal(err)
	}
	if err := GenerateKey(context.Background(), keyPath2); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := EncryptSecrets(context.Background(), secretsPath, keyPath1, "secret=data"); err != nil {
		t.Fatal(err)
	}

	_, err := DecryptSecrets(context.Background(), secretsPath, keyPath2)
	if err == nil {
		t.Error("DecryptSecrets with wrong key should return error")
	}
}

func TestGenerateKey_OverwritesExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	firstKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	secondKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(firstKey) == string(secondKey) {
		t.Error("regenerating key should produce different key content")
	}
}
