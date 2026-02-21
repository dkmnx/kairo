package recoveryphrase

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func CreateRecoveryPhrase(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"read key", err)
	}

	phrase := base64.RawStdEncoding.EncodeToString(keyData)

	words := strings.Fields(phrase)

	return strings.Join(words, "-"), nil
}

func RecoverFromPhrase(configDir, phrase string) error {
	words := strings.Split(phrase, "-")
	encoded := strings.Join(words, "")

	keyData, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"decode phrase", err)
	}

	keyPath := filepath.Join(configDir, "age.key")
	return os.WriteFile(keyPath, keyData, 0600)
}

func GenerateRecoveryPhrase() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"generate key", err)
	}

	phrase := base64.RawStdEncoding.EncodeToString(key)
	return strings.Join(strings.Fields(phrase), "-"), nil
}
