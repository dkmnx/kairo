package crypto

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"filippo.io/age"
	"github.com/dkmnx/kairo/internal/constants"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func GenerateKey(ctx context.Context, keyPath string) error {
	if err := kairoerrors.CheckContext(ctx); err != nil {
		return err
	}

	key, err := age.GenerateX25519Identity()
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate encryption key", err).
			WithContext("path", keyPath)
	}

	tempKeyPath := keyPath + ".tmp"

	keyFile, err := os.OpenFile(tempKeyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temporary key file", err).
			WithContext("path", tempKeyPath)
	}

	_, err = fmt.Fprintf(keyFile, "%s\n%s\n", key.String(), key.Recipient().String())
	if err != nil {
		keyFile.Close()
		_ = os.Remove(tempKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write key to file", err).
			WithContext("path", tempKeyPath)
	}

	if err := keyFile.Close(); err != nil {
		_ = os.Remove(tempKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temporary key file", err).
			WithContext("path", tempKeyPath)
	}

	if err := os.Rename(tempKeyPath, keyPath); err != nil {
		_ = os.Remove(tempKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to rename temporary key file", err).
			WithContext("temp_path", tempKeyPath).
			WithContext("key_path", keyPath)
	}

	return nil
}

func EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error {
	if err := kairoerrors.CheckContext(ctx); err != nil {
		return err
	}

	recipient, err := loadRecipient(keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load encryption key", err).
			WithContext("key_path", keyPath).
			WithContext("secrets_path", secretsPath)
	}

	tempPath := secretsPath + ".tmp"

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create temporary secrets file", err).
			WithContext("path", tempPath)
	}

	encryptor, err := age.Encrypt(file, recipient)
	if err != nil {
		file.Close()
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to initialize encryption", err).
			WithContext("secrets_path", secretsPath)
	}

	_, err = encryptor.Write([]byte(secrets))
	if err != nil {
		encryptor.Close()
		file.Close()
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to encrypt secrets", err)
	}

	if err := encryptor.Close(); err != nil {
		file.Close()
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to finalize encryption", err).
			WithContext("secrets_path", secretsPath)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temporary file", err).
			WithContext("path", tempPath)
	}

	if err := os.Rename(tempPath, secretsPath); err != nil {
		_ = os.Remove(tempPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to replace secrets file", err).
			WithContext("temp_path", tempPath).
			WithContext("secrets_path", secretsPath)
	}

	return nil
}

// DecryptSecrets decrypts the secrets file and returns the content as a string.
//
// Deprecated: Use DecryptSecretsBytes instead. Strings are immutable in Go and
// cannot be cleared from memory, which means secret key material remains in
// memory until garbage collection. DecryptSecretsBytes returns a []byte that
// can be cleared with crypto.ClearMemory after use.
func DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error) {
	if err := kairoerrors.CheckContext(ctx); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := decryptToBuffer(ctx, secretsPath, keyPath, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ClearMemory(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

func DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error) {
	var buf bytes.Buffer
	if err := decryptToBuffer(ctx, secretsPath, keyPath, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decryptToBuffer(ctx context.Context, secretsPath, keyPath string, buf *bytes.Buffer) error {
	if err := kairoerrors.CheckContext(ctx); err != nil {
		return err
	}

	identity, err := loadIdentity(keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load decryption key", err).
			WithContext("key_path", keyPath).
			WithContext("hint", "Ensure your encryption key file exists and is valid")
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to open secrets file", err).
			WithContext("path", secretsPath)
	}
	defer file.Close()

	decryptor, err := age.Decrypt(file, identity)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption")
	}

	_, err = buf.ReadFrom(decryptor)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to read decrypted content", err)
	}

	return nil
}

func loadRecipient(keyPath string) (age.Recipient, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to open key file", err).
			WithContext("path", keyPath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, kairoerrors.NewError(kairoerrors.CryptoError,
			"key file is empty").
			WithContext("path", keyPath)
	}
	if !scanner.Scan() {
		return nil, kairoerrors.NewError(kairoerrors.CryptoError,
			"key file is missing recipient line").
			WithContext("path", keyPath).
			WithContext("hint", "key file should contain identity and recipient lines")
	}

	recipient, err := age.ParseX25519Recipient(scanner.Text())
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to parse recipient from key file", err).
			WithContext("path", keyPath).
			WithContext("hint", "key file may be corrupted or malformed")
	}

	return recipient, nil
}

func loadIdentity(keyPath string) (age.Identity, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to open key file", err).
			WithContext("path", keyPath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, kairoerrors.NewError(kairoerrors.CryptoError,
			"key file is empty").
			WithContext("path", keyPath)
	}

	identity, err := age.ParseX25519Identity(scanner.Text())
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to parse identity from key file", err).
			WithContext("path", keyPath).
			WithContext("hint", "key file may be corrupted or invalid format")
	}

	return identity, nil
}

func EnsureKeyExists(ctx context.Context, configDir string) error {
	keyPath := filepath.Join(configDir, constants.KeyFileName)
	_, err := os.Stat(keyPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check key file status", err).
			WithContext("path", keyPath)
	}

	return GenerateKey(ctx, keyPath)
}
