package errors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestCheckContext(t *testing.T) {
	t.Run("returns nil when context is not done", func(t *testing.T) {
		ctx := context.Background()
		err := CheckContext(ctx)
		if err != nil {
			t.Errorf("CheckContext() = %v, want nil", err)
		}
	})

	t.Run("returns context error when done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := CheckContext(ctx)
		if err == nil {
			t.Error("CheckContext() should return error when context is done")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("CheckContext() error = %v, want context.Canceled", err)
		}
	})

	t.Run("returns deadline exceeded when expired", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1))
		defer cancel()
		err := CheckContext(ctx)
		if err == nil {
			t.Error("CheckContext() should return error when context deadline exceeded")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("CheckContext() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestVerificationErr(t *testing.T) {
	t.Run("creates verification error with cause", func(t *testing.T) {
		cause := errors.New("checksum mismatch")
		err := VerificationErr("integrity check failed", cause)

		if err.Type != VerificationError {
			t.Errorf("Type = %v, want %v", err.Type, VerificationError)
		}

		if err.Message != "integrity check failed" {
			t.Errorf("Message = %v, want 'integrity check failed'", err.Message)
		}

		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		errMsg := err.Error()
		expected := "integrity check failed: checksum mismatch"
		if errMsg != expected {
			t.Errorf("Error() = %v, want %v", errMsg, expected)
		}
	})

	t.Run("creates verification error without cause", func(t *testing.T) {
		err := VerificationErr("signature invalid", nil)

		if err.Type != VerificationError {
			t.Errorf("Type = %v, want %v", err.Type, VerificationError)
		}

		if err.Message != "signature invalid" {
			t.Errorf("Message = %v, want 'signature invalid'", err.Message)
		}

		expected := "signature invalid"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("verification error is detectable via errors.Is", func(t *testing.T) {
		err := VerificationErr("check failed", nil)
		target := &KairoError{Type: VerificationError}

		if !err.Is(target) {
			t.Error("errors.Is should detect VerificationError type")
		}
	})
}
