package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func EnsureConfigDir(cliCtx *CLIContext, configDir string) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"creating config directory", err)
	}
	if err := crypto.EnsureKeyExists(cliCtx.GetRootCtx(), configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"creating encryption key", err)
	}

	return nil
}

func LoadConfig(cliCtx *CLIContext, configDir string) (*config.Config, error) {
	cfg, err := cliCtx.GetConfigCache().Get(cliCtx.GetRootCtx(), configDir)
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

type AddProviderParams struct {
	CLIContext   interface{ InvalidateCache(dir string) }
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Provider     config.Provider
	SetAsDefault bool
}

func AddAndSaveProvider(params AddProviderParams) error {
	params.Cfg.Providers[params.ProviderName] = params.Provider
	if params.SetAsDefault && params.Cfg.DefaultProvider == "" {
		params.Cfg.DefaultProvider = params.ProviderName
	}
	if err := config.SaveConfig(context.Background(), params.ConfigDir, params.Cfg); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"saving config", err)
	}

	params.CLIContext.InvalidateCache(params.ConfigDir)

	return nil
}

type SecretsResult struct {
	Secrets     map[string]string
	SecretsPath string
	KeyPath     string
}

func LoadSecrets(ctx context.Context, configDir string) (SecretsResult, error) {
	result := SecretsResult{
		Secrets: make(map[string]string),
	}

	result.SecretsPath = filepath.Join(configDir, config.SecretsFileName)
	result.KeyPath = filepath.Join(configDir, config.KeyFileName)

	if _, err := os.Stat(result.SecretsPath); os.IsNotExist(err) {
		return result, nil
	}

	existingSecrets, err := crypto.DecryptSecretsBytes(ctx, result.SecretsPath, result.KeyPath)
	if err != nil {
		return SecretsResult{}, err
	}
	defer crypto.ClearMemory(existingSecrets)

	result.Secrets = config.ParseSecrets(string(existingSecrets))

	return result, nil
}

func ResetSecretsFiles(ctx context.Context, configDir, secretsPath, keyPath string) error {
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old key file", err)
	}

	if err := os.Remove(secretsPath); err != nil && !os.IsNotExist(err) {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to remove old secrets file", err)
	}

	if err := crypto.EnsureKeyExists(ctx, configDir); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to generate new encryption key", err)
	}

	return nil
}

func SaveSecrets(ctx context.Context, secretsPath, keyPath string, secrets map[string]string) error {
	secretsContent := config.FormatSecrets(secrets)
	if err := crypto.EncryptSecrets(ctx, secretsPath, keyPath, secretsContent); err != nil {
		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"saving secrets", err)
	}

	return nil
}

type ProviderSetup struct {
	CLIContext   *CLIContext
	ConfigDir    string
	Cfg          *config.Config
	ProviderName string
	Secrets      map[string]string
	SecretsPath  string
	KeyPath      string
	IsEdit       bool
}
