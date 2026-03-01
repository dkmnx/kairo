// Package crypto provides encryption and key management operations using the age library.
//
// This package handles:
//   - X25519 key generation (public/private key pairs)
//   - Secret encryption/decryption for secure API key storage
//   - Atomic key replacement to prevent partial state
//
// Thread Safety:
//   - Key file operations are not thread-safe (file I/O)
//   - Functions should not be called concurrently on same key files
//
// Security:
//   - All key files use 0600 permissions (owner only)
//   - Temporary files are created with secure defaults
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
// Uses atomic writes: writes to a temporary file first, then renames it.
// This ensures that the key file is secure even if interrupted during creation.
func GenerateKey(keyPath string) error {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate encryption key", err).
			WithContext("path", keyPath)
	}

	// Write to temporary file first to ensure atomic operation
	// This prevents incomplete or insecure key files if interrupted
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
		_ = os.Remove(tempKeyPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write key to file", err).
			WithContext("path", tempKeyPath)
	}

	if err := keyFile.Close(); err != nil {
		_ = os.Remove(tempKeyPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temporary key file", err).
			WithContext("path", tempKeyPath)
	}

	// Atomically rename temp file to actual path
	// This is atomic on POSIX systems and ensures the key file is only
	// exposed after it's fully written with correct permissions
	if err := os.Rename(tempKeyPath, keyPath); err != nil {
		_ = os.Remove(tempKeyPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to rename temporary key file", err).
			WithContext("temp_path", tempKeyPath).
			WithContext("key_path", keyPath)
	}

	return nil
}

// EncryptSecrets encrypts the given secrets string using age encryption and saves it to the specified path.
// Uses atomic writes: writes to a temporary file first, then renames it.
// This ensures that the original secrets file is not truncated if encryption fails.
func EncryptSecrets(secretsPath, keyPath, secrets string) error {
	recipient, err := loadRecipient(keyPath)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to load encryption key", err).
			WithContext("key_path", keyPath).
			WithContext("secrets_path", secretsPath)
	}

	// Write to temporary file first to avoid truncating original if encryption fails
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
		_ = os.Remove(tempPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to initialize encryption", err).
			WithContext("secrets_path", secretsPath)
	}

	_, err = encryptor.Write([]byte(secrets))
	if err != nil {
		encryptor.Close()
		file.Close()
		_ = os.Remove(tempPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to encrypt secrets", err)
	}

	if err := encryptor.Close(); err != nil {
		file.Close()
		_ = os.Remove(tempPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to finalize encryption", err).
			WithContext("secrets_path", secretsPath)
	}

	// Close file handle
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to close temporary file", err).
			WithContext("path", tempPath)
	}

	// Atomically rename temp file to actual path
	if err := os.Rename(tempPath, secretsPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file on error
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to replace secrets file", err).
			WithContext("temp_path", tempPath).
			WithContext("secrets_path", secretsPath)
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

	decryptor, err := age.Decrypt(file, identity)
	if err != nil {
		return "", kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption. Try 'kairo recover restore' if you have a recovery phrase, or 'kairo backup restore' if you have a backup.")
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(decryptor)
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

	decryptor, err := age.Decrypt(file, identity)
	if err != nil {
		return nil, kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", err).
			WithContext("path", secretsPath).
			WithContext("hint", "Ensure your encryption key matches the one used for encryption. Try 'kairo recover restore' if you have a recovery phrase, or 'kairo backup restore' if you have a backup.")
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(decryptor)
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
