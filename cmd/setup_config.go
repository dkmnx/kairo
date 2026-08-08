package cmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/secrets"
)

// EnsureConfigDir creates the config directory and encryption key if they don't exist.
func EnsureConfigDir(cliCtx *CLIContext, configDir string) error {
	if err := os.MkdirAll(configDir, constants.DirPermSecure); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"creating config directory", err)
	}
	if err := cliCtx.Crypto().EnsureKeyExists(cliCtx.RootCtx(), configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"creating encryption key", err)
	}

	return nil
}

// LoadConfig loads the configuration, returning a default if not found.
func LoadConfig(cliCtx *CLIContext, configDir string) (*config.Config, error) {
	cfg, err := cliCtx.ConfigCache().Get(cliCtx.RootCtx(), configDir)
	if err != nil && !errors.Is(err, kairoerrors.ErrConfigNotFound) {
		return nil, err
	}
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.Provider),
		}
	}

	return cfg, nil
}

// AddProviderParams holds parameters for adding a provider to the configuration.
type AddProviderParams struct {
	CLIContext   *CLIContext
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Provider     config.Provider
	SetAsDefault bool
}

// AddAndSaveProvider adds a provider to the config and persists it.
func AddAndSaveProvider(params AddProviderParams) error {
	params.Cfg.Providers[params.ProviderName] = params.Provider
	if params.SetAsDefault && params.Cfg.DefaultProvider == "" {
		params.Cfg.DefaultProvider = params.ProviderName
	}
	if err := config.SaveConfig(params.CLIContext.RootCtx(), params.ConfigDir, params.Cfg); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}

	params.CLIContext.InvalidateCache(params.ConfigDir)

	return nil
}

// SecretsResult holds the result of loading or initializing secrets.
type SecretsResult struct {
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
	SkippedCount int
	Warnings     []string
}

// LoadSecrets loads and decrypts secrets from the config directory.
func LoadSecrets(cliCtx *CLIContext, configDir string) (SecretsResult, error) {
	ctx := cliCtx.RootCtx()
	result := SecretsResult{
		Secrets: make(map[string]string),
	}

	result.SecretsPath = filepath.Join(configDir, constants.SecretsFileName)
	result.KeyPath = filepath.Join(configDir, constants.KeyFileName)

	if _, err := os.Stat(result.SecretsPath); errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}

	existingSecrets, err := cliCtx.Crypto().DecryptSecretsBytes(ctx, result.SecretsPath, result.KeyPath)
	if err != nil {
		return SecretsResult{}, err
	}
	defer crypto.ClearMemory(existingSecrets)

	secretsResult := secrets.ParseWithStats(string(existingSecrets))
	result.Secrets = secretsResult.Secrets
	result.SkippedCount = secretsResult.SkippedCount
	result.Warnings = secretsResult.Warnings

	return result, nil
}

// ResetSecretsFiles regenerates the encryption key and deletes the old
// secrets. The new key is generated to a temporary path before the old key is
// touched, so a failure mid-reset (e.g. disk full) leaves the previous key
// and secrets fully intact instead of destroying the user's stored API keys.
// The old key is only removed after the new one is installed.
func ResetSecretsFiles(ctx context.Context, cliCtx *CLIContext, configDir, secretsPath, keyPath string) error {
	tmpKeyPath := filepath.Join(configDir, constants.KeyFileName+".new")

	// Clear any stale temp key left by an interrupted previous reset.
	if err := os.Remove(tmpKeyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to prepare new key path", err).
			WithContext("path", tmpKeyPath)
	}

	// Generate the new key first. On failure the old key and secrets are
	// untouched and remain usable.
	if err := cliCtx.Crypto().GenerateKey(ctx, tmpKeyPath); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate new encryption key", err).
			WithContext("path", tmpKeyPath)
	}

	// Move the old key aside, then install the new one. os.Rename replaces an
	// existing destination, so no pre-deletion is required.
	backupKeyPath := keyPath + ".backup"
	if err := os.Rename(keyPath, backupKeyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.Remove(tmpKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to move old key aside", err).
			WithContext("path", keyPath)
	}

	if err := os.Rename(tmpKeyPath, keyPath); err != nil {
		// Best effort: restore the old key so the config directory is not
		// left without one. If the restore fails, the old key remains at
		// backupKeyPath.
		_ = os.Rename(backupKeyPath, keyPath)
		_ = os.Remove(tmpKeyPath)

		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to install new encryption key", err).
			WithContext("path", keyPath)
	}

	// The old key and secrets are no longer usable with the new key.
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

// SaveSecrets encrypts and writes the secrets map to the secrets file.
func SaveSecrets(cliCtx *CLIContext, secretsPath, keyPath string, secretsMap map[string]string) error {
	secretsContent := secrets.Format(secretsMap)
	if err := cliCtx.Crypto().EncryptSecrets(cliCtx.RootCtx(), secretsPath, keyPath, secretsContent); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving secrets", err)
	}

	return nil
}

// ProviderSetup holds parameters for the interactive provider setup wizard.
type ProviderSetup struct {
	CLIContext   *CLIContext
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
}
