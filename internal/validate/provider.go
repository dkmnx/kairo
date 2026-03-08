package validate

import (
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
)

const (
	MaxProviderNameLength = 50
	MaxModelNameLength    = 100
)

func ValidateCrossProviderConfig(cfg *config.Config) error {
	type envVarSource struct {
		provider string
		value    string
	}
	envVarMap := make(map[string][]envVarSource)

	for providerName, provider := range cfg.Providers {
		for _, envVar := range provider.EnvVars {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			envVarMap[key] = append(envVarMap[key], envVarSource{
				provider: providerName,
				value:    value,
			})
		}
	}

	for key, sources := range envVarMap {
		if len(sources) > 1 {
			firstValue := sources[0].value
			allSame := true
			for _, s := range sources {
				if s.value != firstValue {
					allSame = false

					break
				}
			}
			if !allSame {
				return kairoerrors.NewError(kairoerrors.ValidationError,
					fmt.Sprintf("environment variable collision: '%s' is set to different values by providers: %v", key, sources)).
					WithContext("env_var", key)
			}
		}
	}

	return nil
}

func ValidateProviderModel(providerName, modelName string) error {
	if modelName == "" {
		return nil
	}

	def, ok := providers.GetBuiltInProvider(providerName)
	if !ok {
		return nil
	}

	if def.Model == "" {
		return nil
	}

	if err := validateModelName(modelName, providerName); err != nil {
		return err
	}

	return nil
}

func validateModelName(modelName, providerName string) error {
	if len(modelName) > MaxModelNameLength {
		return kairoerrors.NewError(kairoerrors.ValidationError,
			fmt.Sprintf("model name '%s' for provider '%s' is too long (max %d characters)",
				modelName, providerName, MaxModelNameLength)).
			WithContext("model", modelName).
			WithContext("provider", providerName)
	}

	for _, r := range modelName {
		if !isValidModelRune(r) {
			return kairoerrors.NewError(kairoerrors.ValidationError,
				fmt.Sprintf("model name '%s' for provider '%s' contains invalid characters", modelName, providerName)).
				WithContext("model", modelName).
				WithContext("provider", providerName)
		}
	}

	return nil
}

func isValidModelRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.'
}
