// Package crypto provides X25519-based encryption and decryption using the age library.
package crypto

import (
	"bufio"
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"filippo.io/age"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/fsutil"
)

// GenerateKey creates a new X25519 keypair and writes it to keyPath atomically.
func GenerateKey(ctx context.Context, keyPath string) error {
	if err := errors.CheckContext(ctx); err != nil {
		return err
	}

	key, err := age.GenerateX25519Identity()
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to generate encryption key", err).
			WithContext("path", keyPath)
	}

	if err := fsutil.WriteAtomic(keyPath, func(f *os.File) error {
		_, writeErr := fmt.Fprintf(f, "%s\n%s\n", key.String(), key.Recipient().String())

		return writeErr
	}); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to write key file", err).
			WithContext("path", keyPath)
	}

	return nil
}

// EncryptSecrets encrypts the given secrets string and writes the ciphertext to secretsPath.
func EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error {
	if err := errors.CheckContext(ctx); err != nil {
		return err
	}

	recipient, err := loadRecipient(keyPath)
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to load encryption key", err).
			WithContext("key_path", keyPath).
			WithContext("secrets_path", secretsPath)
	}

	if err := fsutil.WriteAtomic(secretsPath, func(f *os.File) error {
		encryptor, encErr := age.Encrypt(f, recipient)
		if encErr != nil {
			return errors.WrapError(errors.CryptoError,
				"failed to initialize encryption", encErr)
		}

		if _, writeErr := encryptor.Write([]byte(secrets)); writeErr != nil {
			return errors.WrapError(errors.CryptoError,
				"failed to encrypt secrets", writeErr)
		}

		if closeErr := encryptor.Close(); closeErr != nil {
			return errors.WrapError(errors.CryptoError,
				"failed to finalize encryption", closeErr)
		}

		return nil
	}); err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to write encrypted secrets file", err).
			WithContext("path", secretsPath)
	}

	return nil
}

// DecryptSecrets decrypts the encrypted secrets file and returns the plaintext as a string.
func DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error) {
	if err := errors.CheckContext(ctx); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := decryptToBuffer(ctx, secretsPath, keyPath, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ClearMemory zeroes out the given byte slice to prevent sensitive data from
// remaining in memory.
func ClearMemory(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

// DecryptSecretsBytes decrypts the encrypted secrets file and returns the plaintext as bytes.
func DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error) {
	var buf bytes.Buffer
	if err := decryptToBuffer(ctx, secretsPath, keyPath, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decryptToBuffer(ctx context.Context, secretsPath, keyPath string, buf *bytes.Buffer) error {
	if err := errors.CheckContext(ctx); err != nil {
		return err
	}

	identity, err := loadIdentity(keyPath)
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to load decryption key", err).
			WithContext("key_path", keyPath).
			WithContext("hint", "Ensure your encryption key file exists and is valid")
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return errors.WrapError(errors.FileSystemError,
			"failed to open secrets file", err).
			WithContext("path", secretsPath)
	}
	defer file.Close()

	decryptor, err := age.Decrypt(file, identity)
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption")
	}

	_, err = buf.ReadFrom(decryptor)
	if err != nil {
		return errors.WrapError(errors.CryptoError,
			"failed to read decrypted content", err)
	}

	return nil
}

// readKeyFileScanner opens a key file and returns a scanner for reading its lines.
// The caller must close the returned file when done.
func readKeyFileScanner(keyPath string) (*os.File, *bufio.Scanner, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, nil, errors.WrapError(errors.FileSystemError,
			"failed to open key file", err).
			WithContext("path", keyPath)
	}

	return file, bufio.NewScanner(file), nil
}

func loadRecipient(keyPath string) (age.Recipient, error) {
	file, scanner, err := readKeyFileScanner(keyPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if !scanner.Scan() {
		return nil, errors.NewError(errors.CryptoError,
			"key file is empty").
			WithContext("path", keyPath)
	}
	if !scanner.Scan() {
		return nil, errors.NewError(errors.CryptoError,
			"key file is missing recipient line").
			WithContext("path", keyPath).
			WithContext("hint", "key file should contain identity and recipient lines")
	}

	recipient, err := age.ParseX25519Recipient(scanner.Text())
	if err != nil {
		return nil, errors.WrapError(errors.CryptoError,
			"failed to parse recipient from key file", err).
			WithContext("path", keyPath).
			WithContext("hint", "key file may be corrupted or malformed")
	}

	return recipient, nil
}

func loadIdentity(keyPath string) (age.Identity, error) {
	file, scanner, err := readKeyFileScanner(keyPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if !scanner.Scan() {
		return nil, errors.NewError(errors.CryptoError,
			"key file is empty").
			WithContext("path", keyPath)
	}

	identity, err := age.ParseX25519Identity(scanner.Text())
	if err != nil {
		return nil, errors.WrapError(errors.CryptoError,
			"failed to parse identity from key file", err).
			WithContext("path", keyPath).
			WithContext("hint", "key file may be corrupted or invalid format")
	}

	return identity, nil
}

// EnsureKeyExists generates a new keypair if one does not already exist in configDir.
func EnsureKeyExists(ctx context.Context, configDir string) error {
	keyPath := filepath.Join(configDir, constants.KeyFileName)
	_, err := os.Stat(keyPath)
	if err == nil {
		return nil
	}
	if !stderrors.Is(err, fs.ErrNotExist) {
		return errors.WrapError(errors.FileSystemError,
			"failed to check key file status", err).
			WithContext("path", keyPath)
	}

	return GenerateKey(ctx, keyPath)
}
