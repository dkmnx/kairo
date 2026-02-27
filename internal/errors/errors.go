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
	// RuntimeError indicates runtime/panic errors
	RuntimeError ErrorType = "runtime"
)

// ErrConfigNotFound is returned when the configuration file does not exist.
var ErrConfigNotFound = errors.New("configuration file not found")

// ErrProviderModelTooLong is returned when a provider model name exceeds the maximum length.
var ErrProviderModelTooLong = errors.New("provider model name is too long")

// ErrProviderModelInvalidChars is returned when a provider model name contains invalid characters.
var ErrProviderModelInvalidChars = errors.New("provider model name contains invalid characters")

// ErrRecoveryPhraseTooLong is returned when a recovery phrase exceeds maximum length.
var ErrRecoveryPhraseTooLong = errors.New("recovery phrase exceeds maximum length")

// ErrRecoveryPhraseTooShort is returned when a recovery phrase is too short.
var ErrRecoveryPhraseTooShort = errors.New("recovery phrase too short")

// ErrRecoveryPhraseInvalid is returned when a recovery phrase is invalid or contains typos.
var ErrRecoveryPhraseInvalid = errors.New("recovery phrase is invalid or contains typos")

// ErrEmptyToken is returned when a token is empty.
var ErrEmptyToken = errors.New("token cannot be empty")

// ErrEmptyTokenPath is returned when a token path is empty.
var ErrEmptyTokenPath = errors.New("token path cannot be empty")

// ErrEmptyCLIPath is returned when a CLI path is empty.
var ErrEmptyCLIPath = errors.New("cli path cannot be empty")

// ErrInvalidPathInBackup is returned when a backup contains an invalid path (path traversal attempt).
var ErrInvalidPathInBackup = errors.New("invalid path in backup (may be path traversal attempt)")

// ErrEnvVarCollision is returned when multiple providers set the same environment variable with different values.
var ErrEnvVarCollision = errors.New("environment variable collision detected")

// ErrUnsupportedFormat is returned when an unsupported export format is used.
var ErrUnsupportedFormat = errors.New("unsupported export format")

// ErrUserCancelled is returned when the user cancels input (Ctrl+C or Ctrl+D).
var ErrUserCancelled = errors.New("user cancelled input")

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

// RuntimeErr creates a formatted runtime/panic error.
// Uses "Err" suffix to avoid conflict with RuntimeError type constant (similar to ProviderErr).
func RuntimeErr(message string, cause error) *KairoError {
	if cause != nil {
		return WrapError(RuntimeError, message, cause)
	}
	return NewError(RuntimeError, message)
}
