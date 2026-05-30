package errors_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func ExampleKairoError_errorTypeChecking() {
	err := kairoerrors.WrapError(kairoerrors.ConfigError,
		"invalid configuration", errors.New("missing field"))

	var configErr *kairoerrors.KairoError
	if errors.As(err, &configErr) {
		fmt.Printf("Error type: %s\n", configErr.Type)
		fmt.Printf("Message: %s\n", configErr.Message)
	}
	// Output:
	// Error type: config
	// Message: invalid configuration
}

func TestErrorExamples(t *testing.T) {
	t.Run("with context", func(t *testing.T) {
		err := kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write file", errors.New("permission denied")).
			WithContext("path", "/config/kairo/config").
			WithContext("permissions", "0600")

		msg := err.Error()
		if !strings.Contains(msg, "failed to write file: permission denied") {
			t.Errorf("expected message with cause, got: %s", msg)
		}
		if !strings.Contains(msg, "path=/config/kairo/config") {
			t.Errorf("expected path context, got: %s", msg)
		}
		if !strings.Contains(msg, "permissions=0600") {
			t.Errorf("expected permissions context, got: %s", msg)
		}
	})

	t.Run("with hint", func(t *testing.T) {
		err := kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets file", errors.New("authentication failed")).
			WithContext("path", "~/.config/kairo/secrets.age").
			WithContext("hint", "ensure key file matches the one used for encryption")

		msg := err.Error()
		if !strings.Contains(msg, "failed to decrypt secrets file: authentication failed") {
			t.Errorf("expected message with cause, got: %s", msg)
		}
		if !strings.Contains(msg, "hint=ensure key file matches the one used for encryption") {
			t.Errorf("expected hint context, got: %s", msg)
		}
	})

	t.Run("crypto key rotation", func(t *testing.T) {
		err := kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets with old key during rotation", errors.New("no identity matched for decryption")).
			WithContext("secrets_path", "/tmp/test/secrets.age").
			WithContext("hint", "old key may be corrupted or invalid")

		msg := err.Error()
		if !strings.Contains(msg, "no identity matched for decryption") {
			t.Errorf("expected cause in message, got: %s", msg)
		}
		if !strings.Contains(msg, "secrets_path=/tmp/test/secrets.age") {
			t.Errorf("expected secrets_path context, got: %s", msg)
		}
	})

	t.Run("multiple context values", func(t *testing.T) {
		err := kairoerrors.WrapError(kairoerrors.ProviderError,
			"provider not available", errors.New("connection timeout")).
			WithContext("provider", "anthropic").
			WithContext("host", "api.anthropic.com").
			WithContext("port", "443").
			WithContext("attempt", "3").
			WithContext("hint", "check network connectivity and firewall settings")

		msg := err.Error()
		if !strings.Contains(msg, "provider not available: connection timeout") {
			t.Errorf("expected message with cause, got: %s", msg)
		}
		for _, ctx := range []string{
			"provider=anthropic",
			"host=api.anthropic.com",
			"port=443",
			"attempt=3",
			"hint=check network connectivity and firewall settings",
		} {
			if !strings.Contains(msg, ctx) {
				t.Errorf("expected context %q in message, got: %s", ctx, msg)
			}
		}
	})
}
