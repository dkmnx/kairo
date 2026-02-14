package config

import (
	"fmt"
	"os"
	"strings"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
)

type MigrationChange struct {
	Provider string
	Field    string
	Old      string
	New      string
}

func MigrateConfigOnUpdate(configDir string) ([]MigrationChange, error) {
	cfg, err := LoadConfig(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to load config for migration", err)
	}

	if len(cfg.Providers) == 0 {
		return nil, nil
	}

	if cfg.DefaultModels == nil {
		cfg.DefaultModels = make(map[string]string)
	}

	var changes []MigrationChange

	for providerName, provider := range cfg.Providers {
		builtinDef, ok := providers.GetBuiltInProvider(providerName)
		if !ok {
			continue
		}

		if builtinDef.Model == "" {
			continue
		}

		userModel := provider.Model

		if userModel == "" {
			provider.Model = builtinDef.Model
			cfg.Providers[providerName] = provider
			cfg.DefaultModels[providerName] = builtinDef.Model
			changes = append(changes, MigrationChange{
				Provider: providerName,
				Field:    "model",
				Old:      "",
				New:      builtinDef.Model,
			})
			continue
		}

		// Update model if user has explicitly set a model that differs from the new default
		// This handles cases where the builtin default changed (e.g., model version upgrade)
		if userModel != builtinDef.Model {
			oldModel := userModel
			provider.Model = builtinDef.Model
			cfg.Providers[providerName] = provider
			cfg.DefaultModels[providerName] = builtinDef.Model
			changes = append(changes, MigrationChange{
				Provider: providerName,
				Field:    "model",
				Old:      oldModel,
				New:      builtinDef.Model,
			})
		} else {
			// User's model already matches the new default, just ensure DefaultModels is set
			cfg.DefaultModels[providerName] = builtinDef.Model
		}
	}

	if len(changes) > 0 {
		if err := SaveConfig(configDir, cfg); err != nil {
			return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
				"failed to save config after migration", err)
		}
	}

	return changes, nil
}

func FormatMigrationChanges(changes []MigrationChange) string {
	if len(changes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nConfig updates:")
	for _, c := range changes {
		b.WriteString(fmt.Sprintf("\n  %s: %s -> %s", c.Provider, c.Old, c.New))
	}
	return b.String()
}
