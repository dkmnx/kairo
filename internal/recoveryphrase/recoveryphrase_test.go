package recoveryphrase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndRecover(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")

	_ = os.WriteFile(keyPath, []byte("test-key-content"), 0600)

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

	content, _ := os.ReadFile(keyPath)
	if string(content) != "test-key-content" {
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
