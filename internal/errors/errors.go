package errors

import (
	"errors"
	"fmt"
)

// ErrorType represents different categories of errors.
type ErrorType string

const (
	// ConfigError indicates configuration-related errors
	ConfigError ErrorType = "config"
	// CryptoError indicates encryption/decryption errors
	CryptoError ErrorType = "crypto"
	// ValidationError indicates input validation errors
	ValidationError ErrorType = "validation"
	// ProviderError indicates provider-related errors
	ProviderError ErrorType = "provider"
	// FileSystemError indicates file system errors
	FileSystemError ErrorType = "filesystem"
	// NetworkError indicates network-related errors
	NetworkError ErrorType = "network"
)

// ErrConfigNotFound is returned when the configuration file does not exist.
var ErrConfigNotFound = errors.New("configuration file not found")

// KairoError is a structured error type that provides context about what went wrong.
type KairoError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]string
}

// Error implements the error interface.
func (e *KairoError) Error() string {
	msg := e.Message
	if e.Cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Cause)
	}

	// Add context if available
	if len(e.Context) > 0 {
		ctx := " ("
		first := true
		for k, v := range e.Context {
			if !first {
				ctx += ", "
			}
			ctx += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		ctx += ")"
		msg += ctx
	}

	return msg
}

// Unwrap returns the underlying cause for use with errors.Is/As.
func (e *KairoError) Unwrap() error {
	return e.Cause
}

// Is returns true if the target error is of the same type.
func (e *KairoError) Is(target error) bool {
	t, ok := target.(*KairoError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// NewError creates a new KairoError with the given type and message.
func NewError(errorType ErrorType, message string) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
	}
}

// WrapError creates a new KairoError that wraps an existing error.
func WrapError(errorType ErrorType, message string, cause error) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// WithContext adds context to an existing KairoError.
func (e *KairoError) WithContext(key, value string) *KairoError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// FileError creates a formatted filesystem error with file path context.
func FileError(message, path string, cause error) *KairoError {
	return WrapError(FileSystemError, message, cause).
		WithContext("path", path)
}

// ConfigFileError creates a formatted configuration file error with path and hints.
func ConfigFileError(message, configPath string, hint string, cause error) *KairoError {
	err := WrapError(ConfigError, message, cause).
		WithContext("path", configPath)
	if hint != "" {
		err = err.WithContext("hint", hint)
	}
	return err
}

// CryptoErrorWithHint creates a formatted crypto error with operation hints.
func CryptoErrorWithHint(message, hint string, cause error) *KairoError {
	return WrapError(CryptoError, message, cause).
		WithContext("hint", hint)
}

// ProviderErr creates a formatted provider error with provider name.
func ProviderErr(message, providerName string, cause error) *KairoError {
	return WrapError(ProviderError, message, cause).
		WithContext("provider", providerName)
}
