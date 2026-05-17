package validate

import (
	"errors"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/providers"
)

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

func TestValidateProviderModel_DefaultModelEmpty(t *testing.T) {
	// custom provider with no default model should skip validation
	err := ValidateProviderModel("custom", "my-model")
	if err != nil {
		t.Errorf("ValidateProviderModel() should skip for custom provider, got: %v", err)
	}
}

func TestValidateProviderModel_NonBuiltInProvider(t *testing.T) {
	// Non-built-in provider with non-empty model should skip
	err := ValidateProviderModel("nonexistent", "some-model")
	if err != nil {
		t.Errorf("ValidateProviderModel() should skip for unknown provider, got: %v", err)
	}
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
			if def, ok := providers.BuiltInProvider(providerName); ok && def.Model != "" {
				if err == nil {
					t.Errorf("ValidateProviderModel() should fail for model name exceeding max length (%d)", MaxModelNameLength)
				}
			}
		}

		// For built-in providers with default models, verify invalid characters fail
		if modelName != "" && err == nil {
			if def, ok := providers.BuiltInProvider(providerName); ok && def.Model != "" {
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
