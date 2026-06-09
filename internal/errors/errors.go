// Package errors defines Kairo-specific error types with structured context.
package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrorType classifies a KairoError into a high-level category such as
// config, crypto, validation, or network errors.
type ErrorType string

// Error type constants for classifying KairoError instances.
const (
	ConfigError       ErrorType = "config"
	CryptoError       ErrorType = "crypto"
	ValidationError   ErrorType = "validation"
	ProviderError     ErrorType = "provider"
	FileSystemError   ErrorType = "filesystem"
	NetworkError      ErrorType = "network"
	RuntimeError      ErrorType = "runtime"
	VerificationError ErrorType = "verification"
)

// ErrConfigNotFound is returned when the configuration file does not exist.
var ErrConfigNotFound = errors.New("configuration file not found")

// ErrUserCancelled is returned when the user cancels an interactive prompt.
var ErrUserCancelled = errors.New("user canceled input")

// ErrBinaryOutdated is returned when the configuration file contains fields
// not recognized by this binary version, indicating an upgrade is needed.
var ErrBinaryOutdated = errors.New("your installed kairo binary is outdated")

// KairoError is a structured error with a type classification, message,
// optional cause, and key-value context metadata.
type KairoError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]string
}

func (e *KairoError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if e.Cause != nil {
		fmt.Fprintf(&b, ": %v", e.Cause)
	}
	if len(e.Context) > 0 {
		b.WriteString(" (")
		first := true
		for k, v := range e.Context {
			if !first {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s=%s", k, v)
			first = false
		}
		b.WriteString(")")
	}

	return b.String()
}

func (e *KairoError) Unwrap() error {
	return e.Cause
}

func (e *KairoError) Is(target error) bool {
	t, ok := target.(*KairoError)
	if !ok {
		return false
	}

	return e.Type == t.Type
}

// NewError creates a KairoError with the given type and message.
func NewError(errorType ErrorType, message string) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
	}
}

// WrapError creates a KairoError that wraps an existing cause error.
func WrapError(errorType ErrorType, message string, cause error) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// Standard context keys used across the codebase with WithContext():
//   "path"         - file path
//   "config_dir"   - configuration directory path
//   "key_path"     - encryption key file path
//   "secrets_path" - encrypted secrets file path
//   "provider"     - provider name
//   "hint"         - user-facing troubleshooting hint
//   "model"        - model name
//   "env_var"      - environment variable name
//
// Keep keys lowercase_snake_case and consistent across all packages.

// WithContext adds a key-value pair to the error's context metadata.
func (e *KairoError) WithContext(key, value string) *KairoError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value

	return e
}

// FileError creates a FileSystemError with the file path in context.
func FileError(message, path string, cause error) *KairoError {
	return WrapError(FileSystemError, message, cause).
		WithContext("path", path)
}

// CheckContext returns ctx.Err() if the context has been canceled or expired.
func CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// VerificationErr creates a VerificationError wrapping the given cause.
func VerificationErr(message string, cause error) *KairoError {
	return WrapError(VerificationError, message, cause)
}
