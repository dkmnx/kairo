package cmd

import (
	"errors"
	"os"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
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
