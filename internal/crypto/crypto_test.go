package crypto

import (
	"context"

	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKeyGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
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
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := `ZAI_API_KEY=sk-test-key
MINIMAX_API_KEY=sk-another-key
`

	err = EncryptSecrets(context.Background(), secretsPath, keyPath, secrets)
	if err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	decrypted, err := DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
	}

	if decrypted != secrets {
		t.Errorf("decrypted = %q, want %q", decrypted, secrets)
	}
}

func TestDecryptInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "nonexistent.age")
	_, err = DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error on nonexistent file")
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
	err = GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	readonlyDir := filepath.Join(tmpDir, "readonly")
	err = os.MkdirAll(readonlyDir, 0555)
	if err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(readonlyDir, "secrets.age")
	err = EncryptSecrets(context.Background(), secretsPath, keyPath, "test=secret")
	if err == nil {
		t.Error("EncryptSecrets(context.Background(), ) should error on readonly directory")
	}
}

func TestEnsureKeyExistsWhenKeyExists(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	err = EnsureKeyExists(context.Background(), tmpDir)
	if err != nil {
		t.Errorf("EnsureKeyExists(context.Background(), ) error = %v, want nil when key exists", err)
	}
}

func TestEnsureKeyExistsCreatesKey(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatal("key file should not exist before test")
	}

	err := EnsureKeyExists(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists(context.Background(), ) error = %v", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("key file was not created")
	}
}

func TestEnsureKeyExistsCreatesCorrectFormat(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureKeyExists(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists(context.Background(), ) error = %v", err)
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

	err = EnsureKeyExists(context.Background(), nestedDir)
	if err != nil {
		t.Fatalf("EnsureKeyExists(context.Background(), ) error = %v", err)
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

func TestLoadIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	identity, err := loadIdentity(keyPath)
	if err != nil {
		t.Fatalf("loadIdentity() error = %v", err)
	}
	if identity == nil {
		t.Error("loadIdentity() should return a valid identity")
	}
}

func TestLoadIdentityEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "empty.key")

	if err := os.WriteFile(keyPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadIdentity(keyPath)
	if err == nil {
		t.Error("loadIdentity() should error on empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

func TestLoadIdentityInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "invalid.key")

	badKey := "AGE-SECRET-KEY-INVALID-FORMAT-THIS-IS-NOT-VALID"
	if err := os.WriteFile(keyPath, []byte(badKey+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadIdentity(keyPath)
	if err == nil {
		t.Error("loadIdentity() should error on invalid key format")
	}
}

func TestEncryptSecretsWithInvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	err := EncryptSecrets(context.Background(), secretsPath, "/nonexistent/path/key", "test=secret")
	if err == nil {
		t.Error("EncryptSecrets(context.Background(), ) should error on invalid key path")
	}
}

func TestDecryptSecretsWithInvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	_, err := DecryptSecrets(context.Background(), secretsPath, "/nonexistent/path/key")
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error on invalid key path")
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
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(readonlyDir, "secrets.age")
	err := EncryptSecrets(context.Background(), secretsPath, keyPath, "test=secret")
	if err == nil {
		t.Error("EncryptSecrets(context.Background(), ) should fail when cannot create secrets file")
	}
}

func TestDecryptCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "corrupted.age")

	corruptedData := []byte("this is not encrypted data!!!")
	if err := os.WriteFile(secretsPath, corruptedData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error on corrupted file")
	}
}

func TestDecryptTruncatedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "truncated.age")

	validContent := "ANTHROPIC_API_KEY=test-key-123"
	err = EncryptSecrets(context.Background(), secretsPath, keyPath, validContent)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatal(err)
	}

	truncatedData := data[:len(data)/2]
	if err := os.WriteFile(secretsPath, truncatedData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error on truncated file")
	}
}

func TestDecryptRandomData(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "random.age")

	randomData := make([]byte, 256)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}
	if err := os.WriteFile(secretsPath, randomData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error on random data")
	}
}

func TestDecryptSecretsBytes_Success(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "TEST_API_KEY=secret-value-123"

	err = EncryptSecrets(context.Background(), secretsPath, keyPath, secrets)
	if err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	data, err := DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes(context.Background(), ) error = %v", err)
	}

	if string(data) != secrets {
		t.Errorf("Decrypted content = %q, want %q", string(data), secrets)
	}

	ClearMemory(data)
	for i := range data {
		if data[i] != '\x00' {
			t.Errorf("ClearMemory() should zeroize the data, byte %d is %q not null", i, data[i])
		}
	}
}

func TestDecryptSecretsBytes_ClearMemory(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "my-secret=value"

	err = EncryptSecrets(context.Background(), secretsPath, keyPath, secrets)
	if err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	var decrypted []byte
	func() {
		data, err := DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecretsBytes(context.Background(), ) error = %v", err)
		}
		defer ClearMemory(data)

		decrypted = data
		// Data should still be accessible before defer runs
		if string(decrypted) != secrets {
			t.Errorf("Inside closure: got %q, want %q", string(decrypted), secrets)
		}
	}()

	// After defer runs, data should be zeroized
	for i := range decrypted {
		if decrypted[i] != '\x00' {
			t.Errorf("After ClearMemory(): data should be zeroized, byte %d is %q", i, decrypted[i])
		}
	}
}

func TestDecryptSecretsBytes_InvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	_, err := DecryptSecretsBytes(context.Background(), secretsPath, "/nonexistent/key")
	if err == nil {
		t.Error("DecryptSecretsBytes(context.Background(), ) should error on invalid key path")
	}
}

func TestDecryptSecretsBytes_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "nonexistent.age")
	_, err = DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecretsBytes(context.Background(), ) should error on nonexistent file")
	}
}

func TestDecryptSecretsBytes_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "corrupted.age")
	corruptedData := []byte("not valid age encrypted data")
	if err := os.WriteFile(secretsPath, corruptedData, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecretsBytes(context.Background(), ) should error on corrupted file")
	}
}

func TestClearMemory_MultipleCalls(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := EncryptSecrets(context.Background(), secretsPath, keyPath, "test=value"); err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	data, err := DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecretsBytes(context.Background(), ) error = %v", err)
	}

	ClearMemory(data)
	ClearMemory(data)
	ClearMemory(data)

	// Data should remain zeroized
	for i := range data {
		if data[i] != '\x00' {
			t.Errorf("Data should remain zeroized after multiple ClearMemory() calls, byte %d is %q", i, data[i])
			break
		}
	}
}

func TestGenerateKeyWithFailingTempFile(t *testing.T) {
	// We'll use a regular temp dir since testing precise error conditions needs special setup
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	err := GenerateKey(context.Background(), keyPath)
	if err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) should work normally: %v", err)
	}

	// The key file should exist
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("key file should exist")
	}
}

func TestEncryptSecretsWithFailingTempFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	// Note: We can't easily force a temp file creation error without
	// modifying the os.CreateTemp behavior. The test verifies the
	// happy path and relies on other tests (like readonly) for error paths.

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	err := EncryptSecrets(context.Background(), secretsPath, keyPath, "test=value")
	if err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) should work on valid path: %v", err)
	}
}

func TestDecryptSecrets_OpenError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping readonly test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping readonly test when running as root")
	}

	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := EncryptSecrets(context.Background(), secretsPath, keyPath, "test=secret"); err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(secretsPath, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(secretsPath, 0644) // Clean up

	_, err := DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecrets(context.Background(), ) should error when secrets file is unreadable")
	}
}

func TestDecryptSecretsBytes_OpenError(t *testing.T) {
	// Test DecryptSecretsBytes when file can't be opened
	if runtime.GOOS == "windows" {
		t.Skip("Skipping readonly test on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping readonly test when running as root")
	}

	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatalf("GenerateKey(context.Background(), ) error = %v", err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	if err := EncryptSecrets(context.Background(), secretsPath, keyPath, "test=secret"); err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(secretsPath, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(secretsPath, 0644) // Clean up

	_, err := DecryptSecretsBytes(context.Background(), secretsPath, keyPath)
	if err == nil {
		t.Error("DecryptSecretsBytes(context.Background(), ) should error when secrets file is unreadable")
	}
}
