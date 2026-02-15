package crypto

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKeyGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file not created")
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("key file is empty")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := `ZAI_API_KEY=sk-test-key
MINIMAX_API_KEY=sk-another-key
`

	err = EncryptSecrets(secretsPath, keyPath, secrets)
	if err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	decrypted, err := DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	if decrypted != secrets {
		t.Errorf("decrypted = %q, want %q", decrypted, secrets)
	}
}

func TestDecryptInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "nonexistent.age")
	_, err = DecryptSecrets(secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets() should error on nonexistent file")
	}
}

func TestEncryptToReadonlyDir(t *testing.T) {
	// Skip on Windows (readonly directories work differently)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping readonly test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping readonly test when running as root (permissions don't apply)")
	}

	tmpDir := t.TempDir()
	writableDir := filepath.Join(tmpDir, "writable")
	err := os.MkdirAll(writableDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(writableDir, "age.key")
	err = GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	readonlyDir := filepath.Join(tmpDir, "readonly")
	err = os.MkdirAll(readonlyDir, 0555)
	if err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(readonlyDir, "secrets.age")
	err = EncryptSecrets(secretsPath, keyPath, "test=secret")
	if err == nil {
		t.Error("EncryptSecrets() should error on readonly directory")
	}
}

func TestEnsureKeyExistsWhenKeyExists(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	err = EnsureKeyExists(tmpDir)
	if err != nil {
		t.Errorf("EnsureKeyExists() error = %v, want nil when key exists", err)
	}
}

func TestEnsureKeyExistsCreatesKey(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatal("key file should not exist before test")
	}

	err := EnsureKeyExists(tmpDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file was not created")
	}
}

func TestEnsureKeyExistsCreatesCorrectFormat(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureKeyExists(tmpDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("key file is empty")
	}

	if content[len(content)-1] != '\n' {
		t.Error("key file should end with newline")
	}
}

func TestEnsureKeyExistsWithNestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "dir")

	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	keyPath := filepath.Join(nestedDir, "age.key")
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Fatal("nested directory should exist before test")
	}

	err = EnsureKeyExists(nestedDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists() error = %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file was not created in nested directory")
	}
}

func TestLoadRecipientEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "empty.key")

	if err := os.WriteFile(keyPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadRecipient(keyPath)
	if err == nil {
		t.Error("loadRecipient() should error on empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

func TestLoadRecipientMissingRecipient(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "missing-recipient.key")

	if err := os.WriteFile(keyPath, []byte("identity-line-only\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadRecipient(keyPath)
	if err == nil {
		t.Error("loadRecipient() should error on file missing recipient line")
	}
	if !strings.Contains(err.Error(), "recipient") {
		t.Errorf("error should mention 'recipient', got: %v", err)
	}
}

func TestEncryptSecretsWithInvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	err := EncryptSecrets(secretsPath, "/nonexistent/path/key", "test=secret")
	if err == nil {
		t.Error("EncryptSecrets() should error on invalid key path")
	}
}

func TestDecryptSecretsWithInvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	_, err := DecryptSecrets(secretsPath, "/nonexistent/path/key")
	if err == nil {
		t.Error("DecryptSecrets() should error on invalid key path")
	}
}

func TestEncryptSecretsFileError(t *testing.T) {
	// Skip on Windows (readonly directories work differently)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping readonly test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping readonly test when running as root")
	}

	tmpDir := t.TempDir()
	writableDir := filepath.Join(tmpDir, "writable")
	if err := os.MkdirAll(writableDir, 0755); err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(writableDir, "age.key")
	if err := GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(readonlyDir, "secrets.age")
	err := EncryptSecrets(secretsPath, keyPath, "test=secret")
	if err == nil {
		t.Error("EncryptSecrets() should fail when cannot create secrets file")
	}
}

func TestRotateKey(t *testing.T) {
	tmpDir := t.TempDir()

	oldKeyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(oldKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	oldKeyContent, err := os.ReadFile(oldKeyPath)
	if err != nil {
		t.Fatalf("failed to read old key: %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	originalSecrets := `ZAI_API_KEY=sk-test-key
MINIMAX_API_KEY=sk-another-key
DEEPSEEK_API_KEY=sk-deepseek-key
`
	err = EncryptSecrets(secretsPath, oldKeyPath, originalSecrets)
	if err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	err = RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	newKeyPath := filepath.Join(tmpDir, "age.key")

	_, err = os.Stat(newKeyPath)
	if err != nil {
		t.Errorf("new key file was not created: %v", err)
	}

	newKeyContent, err := os.ReadFile(newKeyPath)
	if err != nil {
		t.Fatalf("failed to read new key: %v", err)
	}

	if string(oldKeyContent) == string(newKeyContent) {
		t.Error("key content should have changed after rotation")
	}

	decrypted, err := DecryptSecrets(secretsPath, newKeyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() with new key error = %v", err)
	}

	if decrypted != originalSecrets {
		t.Errorf("decrypted secrets = %q, want %q", decrypted, originalSecrets)
	}
}

func TestRotateKeyWithEmptySecrets(t *testing.T) {
	tmpDir := t.TempDir()

	oldKeyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(oldKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	emptySecrets := ""

	err = EncryptSecrets(secretsPath, oldKeyPath, emptySecrets)
	if err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	err = RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	newKeyPath := filepath.Join(tmpDir, "age.key")
	decrypted, err := DecryptSecrets(secretsPath, newKeyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() with new key error = %v", err)
	}

	if decrypted != emptySecrets {
		t.Errorf("decrypted secrets = %q, want empty string", decrypted)
	}
}

func TestRotateKeyPreservesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai
    model: zai-model
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	oldKeyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(oldKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	err = EncryptSecrets(secretsPath, oldKeyPath, "ZAI_API_KEY=sk-key\n")
	if err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	err = RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file was lost: %v", err)
	}

	if string(data) != configContent {
		t.Errorf("config content changed after key rotation")
	}
}

func TestRotateKeyNoSecretsFile(t *testing.T) {
	tmpDir := t.TempDir()

	oldKeyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(oldKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	err = RotateKey(tmpDir)
	if err != nil {
		t.Fatalf("RotateKey() should succeed when no secrets file exists, error = %v", err)
	}

	newKeyPath := filepath.Join(tmpDir, "age.key")
	_, err = os.Stat(newKeyPath)
	if err != nil {
		t.Errorf("new key file should still be created: %v", err)
	}
}

func TestRotateKeyInvalidOldKey(t *testing.T) {
	tmpDir := t.TempDir()

	validKeyPath := filepath.Join(tmpDir, "valid.key")
	err := GenerateKey(validKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	validKeyContent, err := os.ReadFile(validKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "TEST_KEY=secret\n"

	if err := EncryptSecrets(secretsPath, validKeyPath, secrets); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	oldKeyPath := filepath.Join(tmpDir, "age.key")
	if err := os.WriteFile(oldKeyPath, []byte("AGE-SECRET-KEY-INVALID1234567890abcdef\nAGE-SECRET-KEY-RECIPIENT123456789012345678901234567890\n"), 0600); err != nil {
		t.Fatal(err)
	}

	err = RotateKey(tmpDir)
	if err == nil {
		t.Error("RotateKey() should error when old key cannot decrypt secrets")
	}

	if err := os.WriteFile(oldKeyPath, validKeyContent, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(secretsPath, oldKeyPath)
	if err != nil {
		t.Errorf("should be able to decrypt with valid key: %v", err)
	}
}

func TestDecryptCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "corrupted.age")

	// Write corrupted data (not valid age encryption)
	corruptedData := []byte("this is not encrypted data!!!")
	if err := os.WriteFile(secretsPath, corruptedData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets() should error on corrupted file")
	}
}

func TestDecryptTruncatedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "truncated.age")

	// First encrypt some valid data
	validContent := "ANTHROPIC_API_KEY=test-key-123"
	err = EncryptSecrets(secretsPath, keyPath, validContent)
	if err != nil {
		t.Fatal(err)
	}

	// Read and truncate the file
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatal(err)
	}

	// Write truncated data (half of original)
	truncatedData := data[:len(data)/2]
	if err := os.WriteFile(secretsPath, truncatedData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets() should error on truncated file")
	}
}

func TestDecryptRandomData(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(keyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "random.age")

	// Write random bytes (not valid age encryption)
	randomData := make([]byte, 256)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}
	if err := os.WriteFile(secretsPath, randomData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets() should error on random data")
	}
}
