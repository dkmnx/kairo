package validate

import (
	"strings"
	"testing"
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
				if tt.providerName != "" && !strings.Contains(errMsg, "API") {
					t.Errorf("ValidateAPIKey() error message should mention API key, got: %v", errMsg)
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

func TestValidateAPIKey_PatternMismatch(t *testing.T) {
	err := ValidateAPIKey("short", "openrouter")
	if err == nil {
		t.Error("expected error for short key on openrouter (requires sk-or- prefix + min 32 chars)")
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
