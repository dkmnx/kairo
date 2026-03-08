package errors

import (
	"errors"
	"fmt"
)

type ErrorType string

const (
	ConfigError     ErrorType = "config"
	CryptoError     ErrorType = "crypto"
	ValidationError ErrorType = "validation"
	ProviderError   ErrorType = "provider"
	FileSystemError ErrorType = "filesystem"
	NetworkError    ErrorType = "network"
	RuntimeError    ErrorType = "runtime"
)

var ErrConfigNotFound = errors.New("configuration file not found")

var ErrProviderModelTooLong = errors.New("provider model name is too long")

var ErrProviderModelInvalidChars = errors.New("provider model name contains invalid characters")

var ErrRecoveryPhraseTooLong = errors.New("recovery phrase exceeds maximum length")

var ErrRecoveryPhraseTooShort = errors.New("recovery phrase too short")

var ErrRecoveryPhraseInvalid = errors.New("recovery phrase is invalid or contains typos")

var ErrEmptyToken = errors.New("token cannot be empty")

var ErrEmptyTokenPath = errors.New("token path cannot be empty")

var ErrEmptyCLIPath = errors.New("cli path cannot be empty")

var ErrInvalidPathInBackup = errors.New("invalid path in backup (may be path traversal attempt)")

var ErrEnvVarCollision = errors.New("environment variable collision detected")

var ErrUnsupportedFormat = errors.New("unsupported export format")

var ErrUserCancelled = errors.New("user cancelled input")

type KairoError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]string
}

func (e *KairoError) Error() string {
	msg := e.Message
	if e.Cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Cause)
	}

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

func NewError(errorType ErrorType, message string) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
	}
}

func WrapError(errorType ErrorType, message string, cause error) *KairoError {
	return &KairoError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

func (e *KairoError) WithContext(key, value string) *KairoError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

func FileError(message, path string, cause error) *KairoError {
	return WrapError(FileSystemError, message, cause).
		WithContext("path", path)
}

func ConfigFileError(message, configPath string, hint string, cause error) *KairoError {
	err := WrapError(ConfigError, message, cause).
		WithContext("path", configPath)
	if hint != "" {
		err = err.WithContext("hint", hint)
	}
	return err
}

func CryptoErrorWithHint(message, hint string, cause error) *KairoError {
	return WrapError(CryptoError, message, cause).
		WithContext("hint", hint)
}

func ProviderErr(message, providerName string, cause error) *KairoError {
	return WrapError(ProviderError, message, cause).
		WithContext("provider", providerName)
}

func RuntimeErr(message string, cause error) *KairoError {
	if cause != nil {
		return WrapError(RuntimeError, message, cause)
	}
	return NewError(RuntimeError, message)
}
