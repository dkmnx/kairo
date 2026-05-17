package config

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kairoErrors "github.com/dkmnx/kairo/internal/errors"
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
		if errors.Is(err, kairoErrors.ErrConfigNotFound) {
			return nil, nil
		}

		return nil, kairoErrors.WrapError(kairoErrors.ConfigError,
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
	defaultsUpdated := false

	for providerName, provider := range cfg.Providers {
		builtinDef, ok := providers.BuiltInProvider(providerName)
		if !ok {
			skipped = append(skipped, providerName)

			continue
		}

		if builtinDef.Model == "" {
			continue
		}

		userModel := provider.Model

		if cfg.DefaultModels[providerName] != builtinDef.Model {
			cfg.DefaultModels[providerName] = builtinDef.Model
			defaultsUpdated = true
		}

		if userModel == builtinDef.Model {
			continue
		}

		if userModel == "" {
			provider.Model = builtinDef.Model
			cfg.Providers[providerName] = provider
			changes = append(changes, MigrationChange{
				Provider: providerName,
				Field:    "model",
				Old:      userModel,
				New:      builtinDef.Model,
			})
		}
	}

	if len(changes) > 0 || defaultsUpdated {
		if err := SaveConfig(ctx, configDir, cfg); err != nil {
			return nil, kairoErrors.WrapError(kairoErrors.ConfigError,
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
		fmt.Fprintf(&b, "\n  %s: %s -> %s", c.Provider, c.Old, c.New)
	}

	return b.String()
}
