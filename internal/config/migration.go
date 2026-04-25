package config

import (
	"context"
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

type MigrationResult struct {
	Changes          []MigrationChange
	SkippedProviders []string
}

func MigrateConfigOnUpdate(ctx context.Context, configDir string) (*MigrationResult, error) {
	cfg, err := LoadConfig(ctx, configDir)
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
	var skipped []string

	for providerName, provider := range cfg.Providers {
		builtinDef, ok := providers.GetBuiltInProvider(providerName)
		if !ok {
			skipped = append(skipped, providerName)

			continue
		}

		if builtinDef.Model == "" {
			continue
		}

		userModel := provider.Model
		if userModel == builtinDef.Model {
			cfg.DefaultModels[providerName] = builtinDef.Model

			continue
		}

		provider.Model = builtinDef.Model
		cfg.Providers[providerName] = provider
		cfg.DefaultModels[providerName] = builtinDef.Model
		changes = append(changes, MigrationChange{
			Provider: providerName,
			Field:    "model",
			Old:      userModel,
			New:      builtinDef.Model,
		})
	}

	if len(changes) > 0 {
		if err := SaveConfig(ctx, configDir, cfg); err != nil {
			return nil, kairoerrors.WrapError(kairoerrors.ConfigError,
				"failed to save config after migration", err)
		}
	}

	return &MigrationResult{Changes: changes, SkippedProviders: skipped}, nil
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
