package recover

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateRecoveryPhrase generates a recovery phrase from the key file
func CreateRecoveryPhrase(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("read key: %w", err)
	}

	// Encode key as base64 phrase
	phrase := base64.RawStdEncoding.EncodeToString(keyData)

	// Split into words for readability
	words := strings.Fields(phrase)

	return strings.Join(words, "-"), nil
}

// RecoverFromPhrase recovers the key file from a recovery phrase
func RecoverFromPhrase(configDir, phrase string) error {
	// Decode phrase back to key
	words := strings.Split(phrase, "-")
	encoded := strings.Join(words, "")

	keyData, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode phrase: %w", err)
	}

	keyPath := filepath.Join(configDir, "age.key")
	return os.WriteFile(keyPath, keyData, 0600)
}

// GenerateRecoveryPhrase generates a fresh recovery phrase (for new keys)
func GenerateRecoveryPhrase() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	phrase := base64.RawStdEncoding.EncodeToString(key)
	return strings.Join(strings.Fields(phrase), "-"), nil
}
