package recoveryphrase

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

const maxPhraseLength = 65536

func CreateRecoveryPhrase(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"read key", err)
	}

	phrase := base64.RawStdEncoding.EncodeToString(keyData)

	words := strings.Fields(phrase)

	// Add MAC for validation
	mac := generateMAC(keyData)
	words = append(words, mac)

	return strings.Join(words, "-"), nil
}

func RecoverFromPhrase(configDir, phrase string) error {
	if len(phrase) > maxPhraseLength {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"validate phrase", kairoerrors.ErrRecoveryPhraseTooLong)
	}

	words := strings.Split(phrase, "-")

	// Validate min word count: base64 words + 1 MAC word
	if len(words) < 2 {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"validate phrase", kairoerrors.ErrRecoveryPhraseTooShort)
	}

	// Extract checksum (last word) and validate
	providedMAC := words[len(words)-1]
	words = words[:len(words)-1]

	encoded := strings.Join(words, "")

	keyData, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"decode phrase", err)
	}

	// Validate HMAC
	expectedMAC := generateMAC(keyData)
	if !hmac.Equal([]byte(providedMAC), []byte(expectedMAC)) {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"validate phrase", kairoerrors.ErrRecoveryPhraseInvalid)
	}

	keyPath := filepath.Join(configDir, "age.key")
	return os.WriteFile(keyPath, keyData, 0600)
}

func generateMAC(data []byte) string {
	// Use HMAC-SHA256 for integrity verification
	// HMAC key is derived from the key data (self-contained verification)
	h := hmac.New(sha256.New, data)
	h.Write(data)
	mac := h.Sum(nil)
	return strings.ToUpper(formatHex(uint32(mac[0])<<24 | uint32(mac[1])<<16 | uint32(mac[2])<<8 | uint32(mac[3])))
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
