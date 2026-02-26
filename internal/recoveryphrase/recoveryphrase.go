package recoveryphrase

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

const checksumLength = 8

func CreateRecoveryPhrase(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"read key", err)
	}

	phrase := base64.RawStdEncoding.EncodeToString(keyData)

	words := strings.Fields(phrase)

	// Add checksum for validation
	checksum := generateChecksum(keyData)
	words = append(words, checksum)

	return strings.Join(words, "-"), nil
}

func RecoverFromPhrase(configDir, phrase string) error {
	words := strings.Split(phrase, "-")

	// Validate minimum word count: base64 words + 1 checksum word
	if len(words) < 2 {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"validate phrase", errors.New("recovery phrase too short"))
	}

	// Extract checksum (last word) and validate
	providedChecksum := words[len(words)-1]
	words = words[:len(words)-1]

	encoded := strings.Join(words, "")

	keyData, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"decode phrase", err)
	}

	// Validate checksum
	expectedChecksum := generateChecksum(keyData)
	if providedChecksum != expectedChecksum {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"validate phrase", errors.New("recovery phrase is invalid or contains typos"))
	}

	keyPath := filepath.Join(configDir, "age.key")
	return os.WriteFile(keyPath, keyData, 0600)
}

func generateChecksum(data []byte) string {
	crc := crc32.ChecksumIEEE(data)
	// Convert to 8-char hex string
	return strings.ToUpper(formatHex(crc))
}

func formatHex(n uint32) string {
	hexChars := "0123456789ABCDEF"
	result := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		result[i] = hexChars[n&0xF]
		n >>= 4
	}
	return string(result)
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
