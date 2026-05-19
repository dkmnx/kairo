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
	if err := crypto.EnsureKeyExists(cliCtx.RootCtx(), configDir); err != nil {
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
func LoadSecrets(ctx context.Context, configDir string) (SecretsResult, error) {
	result := SecretsResult{
		Secrets: make(map[string]string),
	}

	result.SecretsPath = filepath.Join(configDir, constants.SecretsFileName)
	result.KeyPath = filepath.Join(configDir, constants.KeyFileName)

	if _, err := os.Stat(result.SecretsPath); errors.Is(err, fs.ErrNotExist) {
		return result, nil
	}

	existingSecrets, err := crypto.DecryptSecretsBytes(ctx, result.SecretsPath, result.KeyPath)
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

// ResetSecretsFiles deletes and regenerates the encryption key and secrets files.
func ResetSecretsFiles(ctx context.Context, configDir, secretsPath, keyPath string) error {
	if err := os.Remove(keyPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old key file", err)
	}

	if err := os.Remove(secretsPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old secrets file", err)
	}

	if err := crypto.EnsureKeyExists(ctx, configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate new encryption key", err)
	}

	return nil
}

// SaveSecrets encrypts and writes the secrets map to the secrets file.
func SaveSecrets(ctx context.Context, secretsPath, keyPath string, secretsMap map[string]string) error {
	secretsContent := secrets.Format(secretsMap)
	if err := crypto.EncryptSecrets(ctx, secretsPath, keyPath, secretsContent); err != nil {
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
