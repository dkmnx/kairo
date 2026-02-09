package cmd

import (
	"os"
	"path/filepath"
	"testing"

	recoverpkg "github.com/dkmnx/kairo/internal/recover"
)

func TestRecoverGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	os.WriteFile(keyPath, []byte("test-key-data-32-bytes-long!"), 0600)

	rootCmd.SetArgs([]string{"--config", tmpDir, "recover", "generate"})
	err := rootCmd.Execute()

	if err != nil {
		t.Errorf("recover generate failed: %v", err)
	}
}

func TestRecoverGenerateNoKey(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd.SetArgs([]string{"--config", tmpDir, "recover", "generate"})
	err := rootCmd.Execute()

	if err != nil {
		t.Errorf("recover generate with no key failed: %v", err)
	}
}

func TestRecoverRestore(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	// First, generate a phrase from a known key
	knownKey := []byte("test-key-for-recovery-32bytes!")
	os.WriteFile(keyPath, knownKey, 0600)

	// Create phrase using the actual recovery package
	phrase, err := recoverpkg.CreateRecoveryPhrase(keyPath)
	if err != nil {
		t.Fatalf("failed to create phrase: %v", err)
	}

	// Delete key file
	os.Remove(keyPath)

	rootCmd.SetArgs([]string{"--config", tmpDir, "recover", "restore", phrase})
	err = rootCmd.Execute()

	if err != nil {
		t.Errorf("recover restore failed: %v", err)
	}

	// Verify key was restored
	restoredKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Errorf("key file not restored: %v", err)
	}
	if string(restoredKey) != string(knownKey) {
		t.Errorf("restored key does not match original")
	}
}
