package recoveryphrase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndRecover(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	// Use a proper 32-byte key (X25519 private key size)
	keyData := make([]byte, 32)
	for i := range keyData {
		keyData[i] = byte(i % 256)
	}
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}

	phrase, err := CreateRecoveryPhrase(keyPath)
	if err != nil {
		t.Fatalf("CreateRecoveryPhrase failed: %v", err)
	}

	if len(phrase) < 10 {
		t.Error("recovery phrase too short")
	}

	os.Remove(keyPath)

	err = RecoverFromPhrase(tmpDir, phrase)
	if err != nil {
		t.Fatalf("RecoverFromPhrase failed: %v", err)
	}

	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key after recovery: %v", err)
	}
	if string(content) != string(keyData) {
		t.Error("key not recovered correctly")
	}
}

func TestGenerateRecoveryPhrase(t *testing.T) {
	phrase, err := GenerateRecoveryPhrase()
	if err != nil {
		t.Fatalf("GenerateRecoveryPhrase failed: %v", err)
	}

	if len(phrase) < 10 {
		t.Error("phrase too short")
	}
}

func TestRecoverFromPhrase_MaxLength(t *testing.T) {
	tmpDir := t.TempDir()

	longPhrase := make([]byte, maxPhraseLength+1)
	for i := range longPhrase {
		longPhrase[i] = 'A'
	}

	err := RecoverFromPhrase(tmpDir, string(longPhrase))
	if err == nil {
		t.Error("expected error for phrase exceeding max length")
	}
}

func TestRecoverFromPhrase_TooShort(t *testing.T) {
	tmpDir := t.TempDir()

	err := RecoverFromPhrase(tmpDir, "singleword")
	if err == nil {
		t.Error("expected error for phrase with too few words")
	}
}
