package crypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKeyErrorMessages(t *testing.T) {
	t.Run("invalid path returns descriptive error", func(t *testing.T) {
		// Test with an invalid path (e.g., non-existent directory)
		invalidPath := "/nonexistent/directory/age.key"

		err := GenerateKey(invalidPath)
		if err == nil {
			t.Fatal("GenerateKey() should error for invalid path")
		}

		// Error should mention the path that failed
		errMsg := err.Error()
		if !strings.Contains(errMsg, "key file") && !strings.Contains(errMsg, "create") {
			t.Errorf("Error message should mention key file creation, got: %s", errMsg)
		}
	})
}

func TestEncryptSecretsErrorMessages(t *testing.T) {
	t.Run("invalid key path returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretsPath := filepath.Join(tmpDir, "secrets.age")

		err := EncryptSecrets(secretsPath, "/nonexistent/key", "test=secret")
		if err == nil {
			t.Fatal("EncryptSecrets() should error for invalid key path")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "key file") && !strings.Contains(errMsg, "open") {
			t.Errorf("Error message should mention key file, got: %s", errMsg)
		}
	})

	t.Run("readonly directory returns permission error context", func(t *testing.T) {
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
			t.Fatal("EncryptSecrets() should error for readonly directory")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "secrets file") && !strings.Contains(errMsg, "create") {
			t.Errorf("Error message should mention secrets file creation, got: %s", errMsg)
		}
	})
}

func TestDecryptSecretsErrorMessages(t *testing.T) {
	t.Run("nonexistent file returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "nonexistent.age")
		_, err := DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			t.Fatal("DecryptSecrets() should error for nonexistent file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "secrets file") && !strings.Contains(errMsg, "open") {
			t.Errorf("Error message should mention secrets file, got: %s", errMsg)
		}
	})

	t.Run("invalid key path returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretsPath := filepath.Join(tmpDir, "secrets.age")

		_, err := DecryptSecrets(secretsPath, "/nonexistent/key")
		if err == nil {
			t.Fatal("DecryptSecrets() should error for invalid key path")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "key file") && !strings.Contains(errMsg, "open") {
			t.Errorf("Error message should mention key file, got: %s", errMsg)
		}
	})

	t.Run("corrupted secrets file returns decryption error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		if err := os.WriteFile(secretsPath, []byte("corrupted data"), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			t.Fatal("DecryptSecrets() should error for corrupted secrets")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "decrypt") {
			t.Errorf("Error message should mention decryption failure, got: %s", errMsg)
		}
	})
}

func TestRotateKeyErrorMessages(t *testing.T) {
	t.Run("invalid old key returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		validKeyPath := filepath.Join(tmpDir, "valid.key")
		if err := GenerateKey(validKeyPath); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		if err := EncryptSecrets(secretsPath, validKeyPath, "TEST_KEY=secret\n"); err != nil {
			t.Fatal(err)
		}

		// Write invalid key
		oldKeyPath := filepath.Join(tmpDir, "age.key")
		if err := os.WriteFile(oldKeyPath, []byte("INVALID-KEY\n"), 0600); err != nil {
			t.Fatal(err)
		}

		err := RotateKey(tmpDir)
		if err == nil {
			t.Fatal("RotateKey() should error with invalid key")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "decrypt") && !strings.Contains(errMsg, "secret") {
			t.Errorf("Error message should mention decryption or secret, got: %s", errMsg)
		}
	})
}

func TestLoadRecipientErrorMessages(t *testing.T) {
	t.Run("empty file returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "empty.key")
		if err := os.WriteFile(keyPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := loadRecipient(keyPath)
		if err == nil {
			t.Fatal("loadRecipient() should error for empty file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "empty") && !strings.Contains(errMsg, "key file") {
			t.Errorf("Error message should mention empty or key file, got: %s", errMsg)
		}
	})

	t.Run("missing recipient returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "missing.key")
		if err := os.WriteFile(keyPath, []byte("identity-only\n"), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := loadRecipient(keyPath)
		if err == nil {
			t.Fatal("loadRecipient() should error for missing recipient")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "recipient") && !strings.Contains(errMsg, "missing") {
			t.Errorf("Error message should mention recipient, got: %s", errMsg)
		}
	})
}

func TestLoadIdentityErrorMessages(t *testing.T) {
	t.Run("empty file returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "empty.key")
		if err := os.WriteFile(keyPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := loadIdentity(keyPath)
		if err == nil {
			t.Fatal("loadIdentity() should error for empty file")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "empty") && !strings.Contains(errMsg, "key file") {
			t.Errorf("Error message should mention empty or key file, got: %s", errMsg)
		}
	})

	t.Run("invalid identity returns descriptive error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "invalid.key")
		if err := os.WriteFile(keyPath, []byte("INVALID-IDENTITY\n"), 0600); err != nil {
			t.Fatal(err)
		}

		_, err := loadIdentity(keyPath)
		if err == nil {
			t.Fatal("loadIdentity() should error for invalid identity")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "identity") {
			t.Errorf("Error message should mention parse or identity, got: %s", errMsg)
		}
	})
}

func TestErrorWrapping(t *testing.T) {
	t.Run("GenerateKey wraps underlying errors", func(t *testing.T) {
		// This test verifies that errors are properly wrapped with context
		invalidPath := "/nonexistent/path/age.key"
		err := GenerateKey(invalidPath)
		if err == nil {
			t.Fatal("Expected error")
		}

		// Error should contain context about what operation failed
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such file or directory") && !strings.Contains(errMsg, "key") {
			t.Errorf("Error should be wrapped with context, got raw error: %s", errMsg)
		}
	})
}
