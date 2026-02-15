package errors

import (
	"errors"
	"strings"
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
		if !strings.Contains(errMsg, "provider=zai") {
			t.Errorf("Error() should contain provider context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "action=switch") {
			t.Errorf("Error() should contain action context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "provider not configured") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
	})

	t.Run("chaining WithContext works correctly", func(t *testing.T) {
		err := NewError(ConfigError, "invalid config").
			WithContext("file", "/path/to/config").
			WithContext("line", "42")

		errMsg := err.Error()
		// Map order is not guaranteed, just check both values are present
		if !strings.Contains(errMsg, "file=/path/to/config") {
			t.Errorf("Error() should contain file context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "line=42") {
			t.Errorf("Error() should contain line context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "invalid config") {
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
		if !strings.Contains(errMsg, "provider unavailable") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "name=anthropic") {
			t.Errorf("Error() should contain name context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "attempt=3") {
			t.Errorf("Error() should contain attempt context, got: %v", errMsg)
		}
	})

	t.Run("complex error with context and cause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := WrapError(NetworkError, "failed to connect", cause).
			WithContext("host", "api.example.com").
			WithContext("port", "443")

		errMsg := err.Error()
		if !strings.Contains(errMsg, "failed to connect") {
			t.Error("Error message should contain original message")
		}
		if !strings.Contains(errMsg, "connection timeout") {
			t.Error("Error message should contain cause")
		}
		if !strings.Contains(errMsg, "host=api.example.com") {
			t.Error("Error message should contain host context")
		}
	})
}

func TestFileError(t *testing.T) {
	t.Run("creates file error with path context", func(t *testing.T) {
		cause := errors.New("permission denied")
		err := FileError("failed to write config", "/path/to/config.yaml", cause)

		if err.Type != FileSystemError {
			t.Errorf("Type = %v, want %v", err.Type, FileSystemError)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		if err.Context == nil {
			t.Fatal("Context should not be nil")
		}

		if err.Context["path"] != "/path/to/config.yaml" {
			t.Errorf("Context[path] = %v, want '/path/to/config.yaml'", err.Context["path"])
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "failed to write config") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "permission denied") {
			t.Errorf("Error() should contain cause, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "path=/path/to/config.yaml") {
			t.Errorf("Error() should contain path context, got: %v", errMsg)
		}
	})

	t.Run("creates file error without cause", func(t *testing.T) {
		err := FileError("file not found", "/missing/file.txt", nil)

		if err.Type != FileSystemError {
			t.Errorf("Type = %v, want %v", err.Type, FileSystemError)
		}

		if err.Cause != nil {
			t.Errorf("Cause should be nil, got %v", err.Cause)
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "file not found") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "path=/missing/file.txt") {
			t.Errorf("Error() should contain path context, got: %v", errMsg)
		}
	})
}

func TestConfigFileError(t *testing.T) {
	t.Run("creates config file error with path and hint", func(t *testing.T) {
		cause := errors.New("invalid YAML")
		err := ConfigFileError(
			"failed to parse config",
			"/home/user/.config/kairo/config",
			"Check YAML syntax and indentation",
			cause,
		)

		if err.Type != ConfigError {
			t.Errorf("Type = %v, want %v", err.Type, ConfigError)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		if len(err.Context) != 2 {
			t.Errorf("Context length = %d, want 2", len(err.Context))
		}

		if err.Context["path"] != "/home/user/.config/kairo/config" {
			t.Errorf("Context[path] = %v, want '/home/user/.config/kairo/config'", err.Context["path"])
		}

		if err.Context["hint"] != "Check YAML syntax and indentation" {
			t.Errorf("Context[hint] = %v, want 'Check YAML syntax and indentation'", err.Context["hint"])
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "failed to parse config") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "path=/home/user/.config/kairo/config") {
			t.Errorf("Error() should contain path context, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "hint=Check YAML syntax and indentation") {
			t.Errorf("Error() should contain hint context, got: %v", errMsg)
		}
	})

	t.Run("creates config file error without hint", func(t *testing.T) {
		err := ConfigFileError(
			"config file not found",
			"/missing/config.yaml",
			"",
			nil,
		)

		if len(err.Context) != 1 {
			t.Errorf("Context length should be 1 when hint is empty, got %d", len(err.Context))
		}

		if _, hasHint := err.Context["hint"]; hasHint {
			t.Error("Context should not contain 'hint' key when hint is empty")
		}

		if err.Context["path"] != "/missing/config.yaml" {
			t.Errorf("Context[path] = %v, want '/missing/config.yaml'", err.Context["path"])
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "config file not found") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
	})
}

func TestCryptoErrorWithHint(t *testing.T) {
	t.Run("creates crypto error with operation hint", func(t *testing.T) {
		cause := errors.New("X25519 key not found")
		err := CryptoErrorWithHint(
			"failed to decrypt secrets",
			"Ensure 'age.key' exists in config directory",
			cause,
		)

		if err.Type != CryptoError {
			t.Errorf("Type = %v, want %v", err.Type, CryptoError)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		if err.Context == nil {
			t.Fatal("Context should not be nil")
		}

		if err.Context["hint"] != "Ensure 'age.key' exists in config directory" {
			t.Errorf("Context[hint] = %v, want 'Ensure 'age.key' exists in config directory'", err.Context["hint"])
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "failed to decrypt secrets") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "X25519 key not found") {
			t.Errorf("Error() should contain cause, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "hint=Ensure 'age.key' exists in config directory") {
			t.Errorf("Error() should contain hint context, got: %v", errMsg)
		}
	})

	t.Run("creates crypto error without cause", func(t *testing.T) {
		err := CryptoErrorWithHint(
			"encryption failed",
			"Check file permissions on 'secrets.age'",
			nil,
		)

		if err.Cause != nil {
			t.Errorf("Cause should be nil, got %v", err.Cause)
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "encryption failed") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "hint=Check file permissions on 'secrets.age'") {
			t.Errorf("Error() should contain hint context, got: %v", errMsg)
		}
	})
}

func TestProviderErr(t *testing.T) {
	t.Run("creates provider error with provider name", func(t *testing.T) {
		cause := errors.New("HTTP 401 Unauthorized")
		err := ProviderErr("API request failed", "zai", cause)

		if err.Type != ProviderError {
			t.Errorf("Type = %v, want %v", err.Type, ProviderError)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		if err.Context == nil {
			t.Fatal("Context should not be nil")
		}

		if err.Context["provider"] != "zai" {
			t.Errorf("Context[provider] = %v, want 'zai'", err.Context["provider"])
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "API request failed") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "HTTP 401 Unauthorized") {
			t.Errorf("Error() should contain cause, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "provider=zai") {
			t.Errorf("Error() should contain provider context, got: %v", errMsg)
		}
	})

	t.Run("creates provider error for different providers", func(t *testing.T) {
		providers := []string{"zai", "minimax", "kimi", "deepseek", "anthropic", "custom-provider"}

		for _, provider := range providers {
			err := ProviderErr("connection timeout", provider, nil)

			if err.Type != ProviderError {
				t.Errorf("Type = %v, want %v", err.Type, ProviderError)
			}

			if err.Context["provider"] != provider {
				t.Errorf("Context[provider] = %v, want '%s'", err.Context["provider"], provider)
			}

			if err.Cause != nil {
				t.Errorf("Cause should be nil, got %v", err.Cause)
			}
		}
	})

	t.Run("creates provider error without cause", func(t *testing.T) {
		err := ProviderErr("provider not configured", "minimax", nil)

		if err.Cause != nil {
			t.Errorf("Cause should be nil, got %v", err.Cause)
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "provider not configured") {
			t.Errorf("Error() should contain message, got: %v", errMsg)
		}
		if !strings.Contains(errMsg, "provider=minimax") {
			t.Errorf("Error() should contain provider context, got: %v", errMsg)
		}
	})
}
