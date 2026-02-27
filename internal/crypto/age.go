// Package crypto provides encryption and key management operations using the age library.
//
// This package handles:
//   - X25519 key generation (public/private key pairs)
//   - Secret encryption/decryption for secure API key storage
//   - Key rotation for periodic security best practices
//   - Atomic key replacement to prevent partial state
//
// Thread Safety:
//   - Key file operations are not thread-safe (file I/O)
//   - Functions should not be called concurrently on same key files
//
// Security:
//   - All key files use 0600 permissions (owner only)
//   - Temporary files are created with secure defaults
//   - Key rotation uses atomic operations to prevent data loss
//   - Private key material is never logged or printed
//
// Memory Safety Limitation:
//   - Decrypted secrets are returned as strings which are immutable in Go
//   - This means decrypted data remains in memory until garbage collected
//   - For applications requiring secure memory handling, consider using
//     DecryptSecretsBytes which returns []byte that can be explicitly zeroed
//
// Performance:
//   - Key generation uses X25519 (fast, secure curve)
//   - Encryption uses age's efficient streaming API
//   - Temporary key files are cleaned up on failure
package crypto

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// GenerateKey generates a new X25519 encryption key and saves it to the specified path.
func GenerateKey(keyPath string) error {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate encryption key", err).
			WithContext("path", keyPath)
	}

	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create key file", err).
			WithContext("path", keyPath)
	}
	defer keyFile.Close()

	_, err = fmt.Fprintf(keyFile, "%s\n%s\n", key.String(), key.Recipient().String())
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write key to file", err).
			WithContext("path", keyPath)
	}

	return nil
}

// EncryptSecrets encrypts the given secrets string using age encryption and saves to the specified path.
func EncryptSecrets(secretsPath, keyPath, secrets string) error {
	recipient, err := loadRecipient(keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load encryption key", err).
			WithContext("key_path", keyPath).
			WithContext("secrets_path", secretsPath)
	}

	file, err := os.OpenFile(secretsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to create secrets file", err).
			WithContext("path", secretsPath)
	}
	defer file.Close()

	w, err := age.Encrypt(file, recipient)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to initialize encryption", err).
			WithContext("secrets_path", secretsPath)
	}

	_, err = w.Write([]byte(secrets))
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to encrypt secrets", err)
	}

	if err := w.Close(); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to finalize encryption", err)
	}

	return nil
}

// DecryptSecrets decrypts the secrets file and returns the plaintext content.
func DecryptSecrets(secretsPath, keyPath string) (string, error) {
	identity, err := loadIdentity(keyPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load decryption key", err).
			WithContext("key_path", keyPath).
			WithContext("hint", "If your key is lost, use 'kairo recover restore <phrase>' if you have a recovery phrase, or 'kairo backup restore <backup-file>' if you have a backup")
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to open secrets file", err).
			WithContext("path", secretsPath)
	}
	defer file.Close()

	r, err := age.Decrypt(file, identity)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption. Try 'kairo recover restore' if you have a recovery phrase, or 'kairo backup restore' if you have a backup.")
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to read decrypted content", err)
	}

	return buf.String(), nil
}

// SecretBytes wraps decrypted secret data with automatic memory zeroization.
// It implements io.Closer so it can be used with defer for automatic cleanup.
//
// Usage:
//
//	defer secrets.Close()
//	secrets, err := DecryptSecretsBytes(path, keyPath)
//
// _CONTENT := secrets.String()
//
// Close() automatically zeroizes the underlying byte slice to prevent secrets
// from lingering in memory. After Close() is called, the secrets are not recoverable.
type SecretBytes struct {
	data []byte
}

// String returns the secrets as a string. The returned string is immutable
// in Go, so it cannot be cleared from memory. Do not use for sensitive data
// that requires explicit memory cleanup.
func (s *SecretBytes) String() string {
	return string(s.data)
}

// Clear explicitly zeroizes the secrets. Called automatically by Close().
func (s *SecretBytes) Clear() {
	if s.data != nil {
		for i := range s.data {
			s.data[i] = 0
		}
	}
}

// Close zeroizes the secrets and clears the reference.
// This method is safe to call multiple times.
func (s *SecretBytes) Close() error {
	s.Clear()
	s.data = nil
	return nil
}

// DecryptSecretsBytes decrypts the secrets file and returns the plaintext wrapped in SecretBytes.
// The SecretBytes type implements io.Closer and will automatically zeroize the memory
// when Close() is called or when used with defer.
//
// Usage:
//
//	defer secrets.Close()
//	secrets, err := DecryptSecretsBytes(path, keyPath)
//	if err != nil {
//	    return err
//	}
//	content := secrets.String()
//
// This function provides better memory safety than DecryptSecrets for applications
// that need to explicitly clear sensitive data after use.
func DecryptSecretsBytes(secretsPath, keyPath string) (*SecretBytes, error) {
	identity, err := loadIdentity(keyPath)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load decryption key", err).
			WithContext("key_path", keyPath).
			WithContext("hint", "If your key is lost, use 'kairo recover restore <phrase>' if you have a recovery phrase, or 'kairo backup restore <backup-file>' if you have a backup")
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to open secrets file", err).
			WithContext("path", secretsPath)
	}
	defer file.Close()

	r, err := age.Decrypt(file, identity)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption. Try 'kairo recover restore' if you have a recovery phrase, or 'kairo backup restore' if you have a backup.")
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to read decrypted content", err)
	}

	// Return data wrapped in SecretBytes for automatic zeroization
	return &SecretBytes{data: buf.Bytes()}, nil
}

// This function opens the key file, skips the identity line (first line),
// and parses the recipient line (second line) which contains the public
// key used for encryption. The recipient is required for encrypting
// secrets that only this identity can decrypt.
//
// Parameters:
//   - keyPath: Path to the age.key file containing encryption keys
//
// Returns:
//   - age.Recipient: Parsed X25519 recipient for encryption operations
//   - error: Returns error if file cannot be read or parsed
//
// Error conditions:
//   - Returns error when key file cannot be opened (e.g., permissions, not found)
//   - Returns error when key file is empty
//   - Returns error when key file is missing recipient line (second line)
//   - Returns error when recipient line cannot be parsed (e.g., malformed, corrupted)
//
// Thread Safety: Not thread-safe (file I/O operations)
// Security Notes: Key file should have 0600 permissions (owner only)
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

// loadIdentity reads and parses the X25519 identity from an age key file.
//
// This function opens the key file and parses the identity line (first line)
// which contains the private key used for decryption. The identity is
// required for decrypting secrets that were encrypted with the corresponding
// recipient public key.
//
// Parameters:
//   - keyPath: Path to age.key file containing encryption keys
//
// Returns:
//   - age.Identity: Parsed X25519 identity for decryption operations
//   - error: Returns error if file cannot be read or parsed
//
// Error conditions:
//   - Returns error when key file cannot be opened (e.g., permissions, not found)
//   - Returns error when key file is empty
//   - Returns error when identity line cannot be parsed (e.g., malformed, corrupted)
//
// Thread Safety: Not thread-safe (file I/O operations)
// Security Notes: Key file should have 0600 permissions (owner only). Identity contains private key material.
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

// EnsureKeyExists generates a new encryption key if one doesn't exist at the specified directory.
func EnsureKeyExists(configDir string) error {
	keyPath := filepath.Join(configDir, config.KeyFileName)
	_, err := os.Stat(keyPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check key file status", err).
			WithContext("path", keyPath)
	}
	return GenerateKey(keyPath)
}

// RotateKey generates a new encryption key and re-encrypts all secrets with it.
// The old key is replaced with the new key. This should be done periodically
// as a security best practice.
func RotateKey(configDir string) error {
	keyPath := filepath.Join(configDir, config.KeyFileName)
	secretsPath := filepath.Join(configDir, config.SecretsFileName)
	backupKeyPath := keyPath + ".backup"

	_, err := os.Stat(secretsPath)
	if os.IsNotExist(err) {
		return generateNewKeyAndReplace(keyPath)
	}
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to check secrets file status", err).
			WithContext("path", secretsPath)
	}

	decrypted, err := DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets with old key during rotation", err).
			WithContext("hint", "old key may be corrupted or invalid").
			WithContext("secrets_path", secretsPath)
	}

	backupKeyData, err := os.ReadFile(keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to backup old key before rotation", err).
			WithContext("path", keyPath)
	}

	if err := os.WriteFile(backupKeyPath, backupKeyData, 0600); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write old key backup", err).
			WithContext("path", backupKeyPath)
	}

	if err := generateNewKeyAndReplace(keyPath); err != nil {
		os.Remove(backupKeyPath)
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate and replace new encryption key", err).
			WithContext("path", keyPath)
	}

	if err := EncryptSecrets(secretsPath, keyPath, decrypted); err != nil {
		restoreErr := os.Rename(backupKeyPath, keyPath)
		if restoreErr != nil {
			return kairoerrors.WrapError(kairoerrors.CryptoError,
				"CRITICAL: failed to re-encrypt secrets and failed to restore old key", err).
				WithContext("restore_error", restoreErr.Error()).
				WithContext("hint", "manual recovery from backup file required: "+backupKeyPath).
				WithContext("backup_path", backupKeyPath)
		}
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to re-encrypt secrets with new key, old key restored", err).
			WithContext("backup_path", backupKeyPath)
	}

	os.Remove(backupKeyPath)
	return nil
}

// generateNewKeyAndReplace generates a new X25519 key and atomically replaces the old key.
//
// This function generates a temporary new key file, then uses os.Rename
// to atomically replace the old key with the new one. If the rename
// fails, the temporary file is cleaned up. This ensures that key
// replacement is atomic - either completely succeeds or fails without leaving
// partial state.
//
// Parameters:
//   - keyPath: Path to existing age.key file to be replaced
//
// Returns:
//   - error: Returns error if key generation or replacement fails
//
// Error conditions:
//   - Returns error when new key cannot be generated (e.g., disk full, permissions)
//   - Returns error when temporary file cannot be renamed to target (e.g., permissions)
//   - Note: If rename fails, temporary file is cleaned up before returning error
//
// Thread Safety: Not thread-safe (file I/O operations)
// Security Notes: Uses atomic rename operation to prevent partial state. Both old and new key files should have 0600 permissions (owner only).
func generateNewKeyAndReplace(keyPath string) error {
	newKeyPath := keyPath + ".new"
	if err := GenerateKey(newKeyPath); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate temporary new key", err).
			WithContext("path", newKeyPath).
			WithContext("target_path", keyPath)
	}

	if err := os.Rename(newKeyPath, keyPath); err != nil {
		os.Remove(newKeyPath)
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to replace old key with new key", err).
			WithContext("from", newKeyPath).
			WithContext("to", keyPath).
			WithContext("hint", "check file permissions")
	}

	return nil
}
