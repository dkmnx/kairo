package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ErrorType string

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

var ErrConfigNotFound = errors.New("configuration file not found")

var ErrUserCancelled = errors.New("user cancelled input")

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

func CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func VerificationErr(message string, cause error) *KairoError {
	return WrapError(VerificationError, message, cause)
}
