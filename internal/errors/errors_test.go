package errors

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	t.Run("creates error with type and message", func(t *testing.T) {
		err := NewError(ConfigError, "configuration not found")
		if err == nil {
			t.Fatal("NewError() returned nil")
		}

		if err.Type != ConfigError {
			t.Errorf("Type = %v, want %v", err.Type, ConfigError)
		}

		if err.Message != "configuration not found" {
			t.Errorf("Message = %v, want 'configuration not found'", err.Message)
		}

		expected := "configuration not found"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})
}

func TestWrapError(t *testing.T) {
	t.Run("wraps error with type and message", func(t *testing.T) {
		cause := errors.New("permission denied")
		err := WrapError(FileSystemError, "failed to write file", cause)

		if err.Type != FileSystemError {
			t.Errorf("Type = %v, want %v", err.Type, FileSystemError)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		errMsg := err.Error()
		expectedPrefix := "failed to write file: permission denied"
		if errMsg != expectedPrefix {
			t.Errorf("Error() = %v, want %v", errMsg, expectedPrefix)
		}
	})
}

func TestWithContext(t *testing.T) {
	t.Run("adds context to error", func(t *testing.T) {
		err := NewError(ProviderError, "provider not configured").
			WithContext("provider", "zai").
			WithContext("action", "switch")

		if len(err.Context) != 2 {
			t.Errorf("Context length = %d, want 2", len(err.Context))
		}

		if err.Context["provider"] != "zai" {
			t.Errorf("Context[provider] = %v, want 'zai'", err.Context["provider"])
		}

		errMsg := err.Error()
		// Map order is not guaranteed, just check values are present
		if !containsSubstring(errMsg, "provider=zai") {
			t.Errorf("Error() should contain provider context, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "action=switch") {
			t.Errorf("Error() should contain action context, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "provider not configured") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
	})

	t.Run("chaining WithContext works correctly", func(t *testing.T) {
		err := NewError(ConfigError, "invalid config").
			WithContext("file", "/path/to/config").
			WithContext("line", "42")

		errMsg := err.Error()
		// Map order is not guaranteed, just check both values are present
		if !containsSubstring(errMsg, "file=/path/to/config") {
			t.Errorf("Error() should contain file context, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "line=42") {
			t.Errorf("Error() should contain line context, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "invalid config") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
	})
}

func TestUnwrap(t *testing.T) {
	t.Run("returns wrapped error", func(t *testing.T) {
		cause := errors.New("original error")
		err := WrapError(CryptoError, "wrapped error", cause)

		unwrapped := errors.Unwrap(err)
		if unwrapped != cause {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
		}

		if !errors.Is(err, cause) {
			t.Error("errors.Is should return true for wrapped error")
		}
	})

	t.Run("returns nil for non-wrapped error", func(t *testing.T) {
		err := NewError(ValidationError, "validation failed")

		unwrapped := errors.Unwrap(err)
		if unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})
}

func TestIs(t *testing.T) {
	t.Run("returns true for same error type", func(t *testing.T) {
		err1 := NewError(ConfigError, "config error")
		err2 := NewError(ConfigError, "another config error")

		if !err1.Is(err2) {
			t.Error("Is() should return true for same error type")
		}
	})

	t.Run("returns false for different error type", func(t *testing.T) {
		err1 := NewError(ConfigError, "config error")
		err2 := NewError(CryptoError, "crypto error")

		if err1.Is(err2) {
			t.Error("Is() should return false for different error type")
		}
	})

	t.Run("returns false for non-KairoError", func(t *testing.T) {
		err1 := NewError(ConfigError, "config error")
		err2 := errors.New("standard error")

		if err1.Is(err2) {
			t.Error("Is() should return false for non-KairoError")
		}
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("all error types are defined", func(t *testing.T) {
		types := []ErrorType{
			ConfigError,
			CryptoError,
			ValidationError,
			ProviderError,
			FileSystemError,
			NetworkError,
		}

		for _, etype := range types {
			if etype == "" {
				t.Errorf("Error type %v is empty", etype)
			}
		}
	})
}

func TestErrorStringFormatting(t *testing.T) {
	t.Run("simple error without context or cause", func(t *testing.T) {
		err := NewError(ValidationError, "invalid input")
		expected := "invalid input"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("error with cause but no context", func(t *testing.T) {
		cause := errors.New("file not found")
		err := WrapError(FileSystemError, "failed to read", cause)
		expected := "failed to read: file not found"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("error with context but no cause", func(t *testing.T) {
		err := NewError(ProviderError, "provider unavailable").
			WithContext("name", "anthropic").
			WithContext("attempt", "3")
		errMsg := err.Error()
		// Map order is not guaranteed, just check values are present
		if !containsSubstring(errMsg, "provider unavailable") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "name=anthropic") {
			t.Errorf("Error() should contain name context, got: %v", errMsg)
		}
		if !containsSubstring(errMsg, "attempt=3") {
			t.Errorf("Error() should contain attempt context, got: %v", errMsg)
		}
	})

	t.Run("complex error with context and cause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := WrapError(NetworkError, "failed to connect", cause).
			WithContext("host", "api.example.com").
			WithContext("port", "443")

		errMsg := err.Error()
		if !containsSubstring(errMsg, "failed to connect") {
			t.Error("Error message should contain original message")
		}
		if !containsSubstring(errMsg, "connection timeout") {
			t.Error("Error message should contain cause")
		}
		if !containsSubstring(errMsg, "host=api.example.com") {
			t.Error("Error message should contain host context")
		}
	})
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
