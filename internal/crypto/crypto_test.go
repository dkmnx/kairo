package crypto

import (
	"os"
	"path/filepath"
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
