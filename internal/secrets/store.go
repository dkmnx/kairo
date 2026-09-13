// Package secrets parses and formats key-value secret entries and persists
// the encrypted secrets store on disk.
package secrets

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// LoadResult holds decrypted secrets plus the paths used to load them.
type LoadResult struct {
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
	SkippedCount int
	Warnings     []string
}

// Paths returns the secrets and key file paths for a config directory.
func Paths(configDir string) (secretsPath, keyPath string) {
	return filepath.Join(configDir, constants.SecretsFileName),
		filepath.Join(configDir, constants.KeyFileName)
}

// Load decrypts and parses secrets from the config directory. A missing
// secrets file is not an error: the result has empty Secrets and the paths
// filled in so callers can save later.
func Load(ctx context.Context, cryptoSvc crypto.Service, configDir string) (LoadResult, error) {
	result := LoadResult{
		Secrets: make(map[string]string),
	}
	result.SecretsPath, result.KeyPath = Paths(configDir)

	if _, err := os.Stat(result.SecretsPath); errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}

	existingSecrets, err := cryptoSvc.DecryptSecretsBytes(ctx, result.SecretsPath, result.KeyPath)
	if err != nil {
		// Keep paths so callers (e.g. --reset-secrets) can still operate
		// on the store after a decrypt failure.
		return result, err
	}
	defer crypto.ClearMemory(existingSecrets)

	parsed := ParseWithStats(string(existingSecrets))
	result.Secrets = parsed.Secrets
	result.SkippedCount = parsed.SkippedCount
	result.Warnings = parsed.Warnings

	return result, nil
}

// Save encrypts and writes the secrets map to the secrets file.
func Save(
	ctx context.Context,
	cryptoSvc crypto.Service,
	secretsPath, keyPath string,
	secretsMap map[string]string,
) error {
	if err := cryptoSvc.EncryptSecrets(ctx, secretsPath, keyPath, Format(secretsMap)); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving secrets", err)
	}

	return nil
}

// Reset regenerates the encryption key and deletes the old secrets. The new
// key is generated to a temporary path before the old key is touched, so a
// failure mid-reset leaves the previous key and secrets intact.
func Reset(ctx context.Context, cryptoSvc crypto.Service, configDir, secretsPath, keyPath string) error {
	tmpKeyPath := filepath.Join(configDir, constants.KeyFileName+".new")

	if err := os.Remove(tmpKeyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to prepare new key path", err).
			WithContext("path", tmpKeyPath)
	}

	if err := cryptoSvc.GenerateKey(ctx, tmpKeyPath); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate new encryption key", err).
			WithContext("path", tmpKeyPath)
	}

	backupKeyPath := keyPath + ".backup"
	if err := os.Rename(keyPath, backupKeyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.Remove(tmpKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to move old key aside", err).
			WithContext("path", keyPath)
	}

	if err := os.Rename(tmpKeyPath, keyPath); err != nil {
		_ = os.Rename(backupKeyPath, keyPath)
		_ = os.Remove(tmpKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to install new encryption key", err).
			WithContext("path", keyPath)
	}

	if err := os.Remove(backupKeyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old key backup", err).
			WithContext("path", backupKeyPath)
	}

	if err := os.Remove(secretsPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old secrets file", err).
			WithContext("path", secretsPath)
	}

	return nil
}
