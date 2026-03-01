package validate

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestProviderValidation(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		providerName string
		wantErr      bool
	}{
		{"empty key", "", "TestProvider", true},
		{"whitespace only", "   ", "TestProvider", true},
		{"short key (7 chars)", "sk-abc", "TestProvider", true},
		{"valid key (20 chars)", "sk-ant-" + string(make([]byte, 14)), "TestProvider", false},
		{"long valid key", "sk-ant-" + string(make([]byte, 50)), "TestProvider", false},
		{"zai provider - short key", "short", "zai", true},
		{"zai provider - valid key", "zai-api-key-" + string(make([]byte, 24)), "zai", false},
		{"custom provider - short key", "short", "custom", true},
		{"custom provider - valid key", "custom-key-" + string(make([]byte, 10)), "custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.key, tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				errMsg := err.Error()
				if tt.providerName != "" && !strings.Contains(errMsg, tt.providerName) {
					t.Errorf("ValidateAPIKey() error message should include provider name, got: %v", errMsg)
				}
			}
		})
	}
}

func TestURLValidation(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		providerName string
		wantErr      bool
	}{
		{"empty URL", "", "TestProvider", true},
		{"http instead of https", "http://api.example.com", "TestProvider", true},
		{"localhost", "https://localhost/api", "TestProvider", true},
		{"127.0.0.1", "https://127.0.0.1/api", "TestProvider", true},
		{"private IP 10.x", "https://10.0.0.1/api", "TestProvider", true},
		{"private IP 172.16.x", "https://172.16.0.1/api", "TestProvider", true},
		{"private IP 192.168.x", "https://192.168.1.1/api", "TestProvider", true},
		{"valid HTTPS", "https://api.example.com/anthropic", "TestProvider", false},
		{"valid with path", "https://api.example.com/v1/anthropic", "TestProvider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				errMsg := err.Error()
				if tt.providerName != "" && !strings.Contains(errMsg, tt.providerName) {
					t.Errorf("ValidateURL() error message should include provider name, got: %v", errMsg)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	// Note: ValidationError was replaced with kairoerrors.KairoError
	// Validation errors now use kairoerrors.ValidationError type
	t.Run("validation errors use KairoError type", func(t *testing.T) {
		err := ValidateAPIKey("", "test")
		if err == nil {
			t.Fatal("Expected validation error")
		}
		// Verify error is a KairoError with ValidationError type
		var kErr *kairoerrors.KairoError
		if !errors.As(err, &kErr) {
			t.Errorf("Expected KairoError, got %T", err)
		}
		if kErr.Type != kairoerrors.ValidationError {
			t.Errorf("Expected ValidationError type, got %v", kErr.Type)
		}
	})
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"public IP 8.8.8.8", "8.8.8.8", false},
		{"public IP 1.1.1.1", "1.1.1.1", false},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.255.255.255", "10.255.255.255", true},
		{"private 172.16.0.0", "172.16.0.0", true},
		{"private 172.31.255.255", "172.31.255.255", true},
		{"private 192.168.0.0", "192.168.0.0", true},
		{"private 192.168.255.255", "192.168.255.255", true},
		{"link-local 169.254.0.0", "169.254.0.0", true},
		{"link-local 169.254.255.255", "169.254.255.255", true},
		{"public IP 9.9.9.9", "9.9.9.9", false},
		{"public IP 203.0.113.1", "203.0.113.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsBlockedHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{"localhost", "localhost", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"::1 IPv6 localhost", "::1", true},
		{"public host", "api.example.com", false},
		{"public IP", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlockedHost(tt.host)
			if got != tt.want {
				t.Errorf("isBlockedHost(%s) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

// FuzzValidateAPIKey fuzzes the ValidateAPIKey function with random inputs.
func FuzzValidateAPIKey(f *testing.F) {
	// Seed with some initial values
	f.Add("sk-ant-valid-key-12345678901234567890", "zai")
	f.Add("", "zai")
	f.Add("   ", "TestProvider")
	f.Add("short", "zai")
	f.Add("zai-api-key-123456789012345678901234", "zai")
	f.Add("custom-key-1234567890", "custom")
	f.Add("minimax-api-key-12345678901234567890", "minimax")
	f.Add("kimi-api-key-1234567890123456789012345", "kimi")
	f.Add("deepseek-api-key-12345678901234567890", "deepseek")
	f.Add("test-key-12345678901234567890", "unknown_provider")

	f.Fuzz(func(t *testing.T, key, providerName string) {
		err := ValidateAPIKey(key, providerName)

		// Verify error message always includes provider name when provided
		if err != nil && providerName != "" {
			errMsg := err.Error()
			if !strings.Contains(errMsg, providerName) {
				t.Errorf("ValidateAPIKey() error message should include provider name %q, got: %v", providerName, errMsg)
			}
		}

		// Verify empty/whitespace keys always fail
		if strings.TrimSpace(key) == "" && err == nil {
			t.Errorf("ValidateAPIKey() should fail for empty/whitespace key, got nil error")
		}

		// Verify known providers with short keys always fail
		knownProviders := []string{"zai", "minimax", "kimi", "deepseek", "custom"}
		for _, p := range knownProviders {
			if providerName == p && len(key) < 20 && err == nil {
				t.Errorf("ValidateAPIKey() should fail for %s provider with key length %d", providerName, len(key))
			}
		}
	})
}

// FuzzValidateURL fuzzes the ValidateURL function with random inputs.
func FuzzValidateURL(f *testing.F) {
	// Seed with some initial values
	f.Add("https://api.example.com/anthropic", "TestProvider")
	f.Add("", "TestProvider")
	f.Add("http://api.example.com", "TestProvider")
	f.Add("https://localhost/api", "TestProvider")
	f.Add("https://127.0.0.1/api", "TestProvider")
	f.Add("https://10.0.0.1/api", "TestProvider")
	f.Add("https://172.16.0.1/api", "TestProvider")
	f.Add("https://192.168.1.1/api", "TestProvider")
	f.Add("https://api.z.ai/api/anthropic", "zai")
	f.Add("https://api.minimax.io/v1", "minimax")

	f.Fuzz(func(t *testing.T, rawURL, providerName string) {
		err := ValidateURL(rawURL, providerName)

		// Verify error message always includes provider name when provided
		if err != nil && providerName != "" {
			errMsg := err.Error()
			if !strings.Contains(errMsg, providerName) {
				t.Errorf("ValidateURL() error message should include provider name %q, got: %v", providerName, errMsg)
			}
		}

		// Verify empty URLs always fail
		if rawURL == "" && err == nil {
			t.Errorf("ValidateURL() should fail for empty URL")
		}

		// Verify HTTP (non-HTTPS) URLs always fail
		if strings.HasPrefix(rawURL, "http://") && err == nil {
			t.Errorf("ValidateURL() should fail for HTTP URL: %s", rawURL)
		}

		// Parse the URL to check for blocked hosts
		parsed, parseErr := url.Parse(rawURL)
		if parseErr == nil && parsed.Host != "" {
			host := parsed.Hostname()
			// Verify exact localhost matches always fail
			if host == "localhost" && err == nil {
				t.Errorf("ValidateURL() should fail for localhost URL: %s", rawURL)
			}
			// Verify 127.0.0.1 always fail
			if host == "127.0.0.1" && err == nil {
				t.Errorf("ValidateURL() should fail for 127.0.0.1 URL: %s", rawURL)
			}
		}
	})
}

// FuzzValidateProviderModel fuzzes the ValidateProviderModel function with random inputs.
func FuzzValidateProviderModel(f *testing.F) {
	// Seed with some initial values
	f.Add("claude-3-opus-20240229", "anthropic")
	f.Add("", "anthropic")
	f.Add("gpt-4", "openai")
	f.Add("gemini-pro", "google")
	f.Add("invalid@model#name", "anthropic")

	f.Fuzz(func(t *testing.T, modelName, providerName string) {
		err := ValidateProviderModel(providerName, modelName)

		// Verify empty model names are always valid (allowed to use provider default)
		if modelName == "" && err != nil {
			t.Errorf("ValidateProviderModel() should allow empty model names, got error: %v", err)
		}

		// Note: ValidateProviderModel only validates model names for built-in providers
		// that have a default model set. For custom providers or built-in providers
		// without default models, it returns nil. This is by design.

		// Verify model names exceeding max length always fail (only for built-in providers)
		if len(modelName) > MaxModelNameLength {
			// For built-in providers with default models, this should fail
			if def, ok := providers.GetBuiltInProvider(providerName); ok && def.Model != "" {
				if err == nil {
					t.Errorf("ValidateProviderModel() should fail for model name exceeding max length (%d)", MaxModelNameLength)
				}
			}
		}

		// For built-in providers with default models, verify invalid characters fail
		if modelName != "" && err == nil {
			if def, ok := providers.GetBuiltInProvider(providerName); ok && def.Model != "" {
				// If validation passed for a built-in provider, verify all characters are valid
				for _, r := range modelName {
					if !isValidModelRune(r) {
						t.Errorf("ValidateProviderModel() should fail for model with invalid character %q in %q", r, modelName)
					}
				}
			}
		}
	})
}

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

		err := ValidateCrossProviderConfig(cfg)

		// Verify the validation logic
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

		// Check for collisions
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
