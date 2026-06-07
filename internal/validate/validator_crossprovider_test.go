package validate

import (
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

// FuzzValidateCrossProviderConfig fuzzes the ValidateCrossProviderConfig function with random inputs.
// Note: Go's native fuzzing doesn't support complex struct types, so we fuzz with string inputs
// that represent serialized provider configurations.
func FuzzValidateCrossProviderConfig(f *testing.F) {
	// Seed with some initial test cases representing different scenarios
	// Format: "provider1:env1=val1;provider2:env2=val2"
	f.Add("provider1:API_KEY=value1")
	f.Add("provider1:API_KEY=value1;provider2:API_KEY=value2")
	f.Add("provider1:API_KEY=same;provider2:API_KEY=same")
	f.Add("provider1:VAR1=val1;provider2:VAR2=val2")
	f.Add("provider1:API_KEY=val1;provider2:API_KEY=val1;provider3:API_KEY=val2")

	f.Fuzz(func(t *testing.T, input string) {
		// Parse the input string into a config
		cfg := parseFuzzConfig(input)
		if cfg == nil {
			t.Skip("Invalid config generated from fuzz input")
		}

		err := ValidateCrossProviderConfig(cfg.Providers)

		envVarValues := make(map[string]map[string]string) // envVar -> provider -> value
		for providerName, provider := range cfg.Providers {
			for _, envVar := range provider.EnvVars {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if envVarValues[key] == nil {
						envVarValues[key] = make(map[string]string)
					}
					envVarValues[key][providerName] = value
				}
			}
		}

		hasCollision := false
		for _, values := range envVarValues {
			if len(values) > 1 {
				vals := make([]string, 0, len(values))
				for _, v := range values {
					vals = append(vals, v)
				}
				for i := 0; i < len(vals); i++ {
					for j := i + 1; j < len(vals); j++ {
						if vals[i] != vals[j] {
							hasCollision = true
							break
						}
					}
				}
			}
		}

		if hasCollision && err == nil {
			t.Errorf("ValidateCrossProviderConfig() should fail when env vars have different values across providers")
		}
	})
}

// parseFuzzConfig parses a fuzz input string into a Config struct.
// Format: "provider1:env1=val1;provider2:env2=val2"
func parseFuzzConfig(input string) *config.Config {
	cfg := &config.Config{
		Providers: make(map[string]config.Provider),
	}

	if input == "" {
		return cfg
	}

	// Split by semicolon to get provider entries
	entries := strings.Split(input, ";")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Split by colon to get provider name and env vars
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}

		providerName := strings.TrimSpace(parts[0])
		if providerName == "" {
			continue
		}

		envVars := strings.Split(strings.TrimSpace(parts[1]), ";")
		cleanEnvVars := make([]string, 0, len(envVars))
		for _, envVar := range envVars {
			envVar = strings.TrimSpace(envVar)
			if envVar != "" {
				cleanEnvVars = append(cleanEnvVars, envVar)
			}
		}

		cfg.Providers[providerName] = config.Provider{
			Name:    providerName,
			BaseURL: "https://api." + providerName + ".example.com",
			EnvVars: cleanEnvVars,
		}
	}

	return cfg
}
