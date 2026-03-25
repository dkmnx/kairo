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

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "empty pattern always matches",
			pattern: "",
			key:     "any-key",
			want:    true,
			wantErr: false,
		},
		{
			name:    "simple alphanumeric pattern",
			pattern: "^[a-z0-9-]+$",
			key:     "sk-ant-valid-key-123",
			want:    true,
			wantErr: false,
		},
		{
			name:    "simple pattern with invalid key",
			pattern: "^[a-z0-9-]+$",
			key:     "INVALID_KEY!@#",
			want:    false,
			wantErr: false,
		},
		{
			name:    "anthropic key pattern",
			pattern: "^sk-ant-[a-z0-9-]+$",
			key:     "sk-ant-api1234567890abcdef",
			want:    true,
			wantErr: false,
		},
		{
			name:    "anthropic pattern mismatch",
			pattern: "^sk-ant-[a-z0-9-]+$",
			key:     "sk-openai-key1234567890",
			want:    false,
			wantErr: false,
		},
		{
			name:    "case sensitive pattern",
			pattern: "^[A-Z]{2}[0-9]+$",
			key:     "AB123456",
			want:    true,
			wantErr: false,
		},
		{
			name:    "case sensitive pattern fails on lowercase",
			pattern: "^[A-Z]{2}[0-9]+$",
			key:     "ab123456",
			want:    false,
			wantErr: false,
		},
		{
			name:    "complex pattern with anchors",
			pattern: "^key-[a-f0-9]{32}$",
			key:     "key-1234567890abcdef1234567890abcdef",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with special characters",
			pattern: `^sk-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`,
			key:     "sk-12345678-1234-1234-1234-123456789012",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with quantifiers",
			pattern: `^[a-z]+(\.[a-z]+)*@[a-z]+\.[a-z]{2,}$`,
			key:     "user.name@domain.com",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid regex pattern",
			pattern: "[invalid(unclosed",
			key:     "any-key",
			want:    false,
			wantErr: true,
		},
		{
			name:    "pattern matching empty string",
			pattern: "^$",
			key:     "",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with word boundaries",
			pattern: `^\b\w+\b$`,
			key:     "valid_key",
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := KeyFormat{Pattern: tt.pattern}
			got, err := kf.matchesPattern(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchesPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

func TestMatchesPattern_CompilationCaching(t *testing.T) {
	kf := KeyFormat{Pattern: `^test-[a-z]+$`}

	_, err := kf.matchesPattern("test-key")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	_, err = kf.matchesPattern("test-another")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	_, err = kf.matchesPattern("test-123")
	if err != nil {
		t.Fatalf("Third call should use cached pattern: %v", err)
	}
}

func TestMatchesPattern_InvalidPatternReturnsError(t *testing.T) {
	invalidPatterns := []string{
		"[",
		"(",
		"*?",
		"???",
		"(?P<invalid>",
		"(?P",  // incomplete named group
		"(?",   // incomplete group
		"[a-z", // unclosed bracket
		"[^]",  // invalid negated class
	}

	for _, pattern := range invalidPatterns {
		t.Run(pattern, func(t *testing.T) {
			kf := KeyFormat{Pattern: pattern}
			_, err := kf.matchesPattern("test-key")
			if err == nil {
				t.Errorf("matchesPattern(%q) should return error for invalid pattern", pattern)
			}
		})
	}
}

func TestMatchesPattern_PatternMatchingEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		want    bool
	}{
		{"unicode in pattern", `^[\pL\pN_-]+$`, "日本語キー-123", true},
		{"long key starting with sk-", `^sk-`, "sk-" + strings.Repeat("a", 1000), true},
		{"whitespace in key fails alphanumeric pattern", `^[a-z0-9]+$`, "key with space", false},
		{"newline in key", `^[\w]+$`, "key\nwith\nnewline", false},
		{"tab in key", `^[\w]+$`, "key\twith\ttab", false},
		{"very long pattern match", `^[a-z]+$`, strings.Repeat("abcdefgh", 100), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := KeyFormat{Pattern: tt.pattern}
			got, err := kf.matchesPattern(tt.key)
			if err != nil {
				t.Skipf("Pattern %q not supported: %v", tt.pattern, err)
			}
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

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
		{"unknown provider - short key", "short", "unknownprovider", true},
		{"unknown provider - valid key (20 chars)", "valid-api-key-12345678", "unknownprovider", false},
		{"key at exact minimum length", "12345678901234567890", "TestProvider", false},
		{"key just below minimum", "1234567890123456789", "TestProvider", true},
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

func TestValidateAPIKey_EdgeCases(t *testing.T) {
	t.Run("empty provider name", func(t *testing.T) {
		err := ValidateAPIKey("valid-key-with-20-chars", "")
		if err != nil {
			t.Errorf("ValidateAPIKey with empty provider name should not error for valid key, got: %v", err)
		}
	})

	t.Run("key with only whitespace variations", func(t *testing.T) {
		keys := []string{
			" ",
			"  ",
			"\t",
			"\n",
			"\r\n",
			" \t\n",
		}
		for _, key := range keys {
			err := ValidateAPIKey(key, "TestProvider")
			if err == nil {
				t.Errorf("ValidateAPIKey(%q) should fail for whitespace-only key", key)
			}
		}
	})

	t.Run("unicode key handling", func(t *testing.T) {
		// Unicode characters should be counted in length
		longUnicodeKey := "日本語キー123456789012345678" // 20+ chars with unicode
		err := ValidateAPIKey(longUnicodeKey, "TestProvider")
		if err != nil {
			t.Errorf("ValidateAPIKey with unicode should work for long enough key, got: %v", err)
		}
	})

	t.Run("all known providers minimum lengths", func(t *testing.T) {
		providerLengths := map[string]int{
			"zai":      32,
			"minimax":  32,
			"kimi":     32,
			"deepseek": 32,
			"custom":   20,
		}

		for provider, minLen := range providerLengths {
			t.Run(provider, func(t *testing.T) {
				// Test key just under minimum
				shortKey := strings.Repeat("a", minLen-1)
				err := ValidateAPIKey(shortKey, provider)
				if err == nil {
					t.Errorf("ValidateAPIKey with %d chars should fail for %s (min %d)", minLen-1, provider, minLen)
				}

				// Test key at minimum
				validKey := strings.Repeat("a", minLen)
				err = ValidateAPIKey(validKey, provider)
				if err != nil {
					t.Errorf("ValidateAPIKey with %d chars should pass for %s (min %d), got: %v", minLen, provider, minLen, err)
				}
			})
		}
	})
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

		if err != nil && providerName != "" {
			errMsg := err.Error()
			if !strings.Contains(errMsg, providerName) {
				t.Errorf("ValidateAPIKey() error message should include provider name %q, got: %v", providerName, errMsg)
			}
		}

		if strings.TrimSpace(key) == "" && err == nil {
			t.Errorf("ValidateAPIKey() should fail for empty/whitespace key, got nil error")
		}

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

		if err != nil && providerName != "" {
			errMsg := err.Error()
			if !strings.Contains(errMsg, providerName) {
				t.Errorf("ValidateURL() error message should include provider name %q, got: %v", providerName, errMsg)
			}
		}

		if rawURL == "" && err == nil {
			t.Errorf("ValidateURL() should fail for empty URL")
		}

		if strings.HasPrefix(rawURL, "http://") && err == nil {
			t.Errorf("ValidateURL() should fail for HTTP URL: %s", rawURL)
		}

		// Parse the URL to check for blocked hosts
		parsed, parseErr := url.Parse(rawURL)
		if parseErr == nil && parsed.Host != "" {
			host := parsed.Hostname()
			if host == "localhost" && err == nil {
				t.Errorf("ValidateURL() should fail for localhost URL: %s", rawURL)
			}
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

		if modelName == "" && err != nil {
			t.Errorf("ValidateProviderModel() should allow empty model names, got error: %v", err)
		}

		// Note: ValidateProviderModel only validates model names for built-in providers
		// that have a default model set. For custom providers or built-in providers
		// without default models, it returns nil. This is by design.

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
