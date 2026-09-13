package validate

import (
	"errors"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
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
	// The catalog "custom" entry is validated like any other provider;
	// "my-model" is a valid model identifier.
	err := ValidateProviderModel("custom", "my-model")
	if err != nil {
		t.Errorf("ValidateProviderModel() should accept a valid model for custom, got: %v", err)
	}
}

func TestValidateProviderModel_NonBuiltInProvider(t *testing.T) {
	// Model validation is uniform for every provider name; "some-model"
	// passes the charset regardless of whether the provider is known.
	err := ValidateProviderModel("nonexistent", "some-model")
	if err != nil {
		t.Errorf("ValidateProviderModel() should accept a valid model for unknown provider, got: %v", err)
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

		// Validation is uniform: the same charset and length limits apply to
		// every provider.
		if len(modelName) > MaxModelNameLength && err == nil {
			t.Errorf("ValidateProviderModel() should fail for model name exceeding max length (%d)", MaxModelNameLength)
		}

		// If validation passed, every character must be in the allowed set.
		if modelName != "" && err == nil {
			for _, r := range modelName {
				if !isValidModelRune(r) {
					t.Errorf("ValidateProviderModel() should fail for model with invalid character %q in %q", r, modelName)
				}
			}
		}
	})
}
