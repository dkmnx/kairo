package errors_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"gopkg.in/yaml.v3"
)

func Demo_KairoError_with_context() {
	// Before: "failed to write file: permission denied"
	// After: "failed to write file: permission denied (path=/config/kairo/config, permissions=0600)"

	err := kairoerrors.WrapError(kairoerrors.FileSystemError,
		"failed to write file", errors.New("permission denied")).
		WithContext("path", "/config/kairo/config").
		WithContext("permissions", "0600")

	fmt.Println(err.Error())
	// Output: failed to write file: permission denied (path=/config/kairo/config, permissions=0600)
}

func Demo_KairoError_with_hint() {
	// Before: "failed to decrypt secrets: authentication failed"
	// After: "failed to decrypt secrets file: authentication failed (path=~/.config/kairo/secrets.age, hint=ensure key file matches the one used for encryption)"

	err := kairoerrors.WrapError(kairoerrors.CryptoError,
		"failed to decrypt secrets file", errors.New("authentication failed")).
		WithContext("path", "~/.config/kairo/secrets.age").
		WithContext("hint", "ensure key file matches the one used for encryption")

	fmt.Println(err.Error())
	// Output: failed to decrypt secrets file: authentication failed (path=~/.config/kairo/secrets.age, hint=ensure key file matches the one used for encryption)
}

func Demo_KairoError_crypto_key_rotation() {
	// Demonstrates a complex error scenario during key rotation
	_ = func(tmpDir string) error {
		_ = filepath.Join(tmpDir, "age.key")

		// Simulate a decryption failure
		cause := errors.New("no identity matched for decryption")

		// Before: "failed to decrypt existing secrets: no identity matched for decryption"
		// After: "failed to decrypt secrets with old key during rotation: no identity matched for decryption (secrets_path=/tmp/test/secrets.age, hint=old key may be corrupted or invalid)"

		return kairoerrors.WrapError(kairoerrors.CryptoError,
			"failed to decrypt secrets with old key during rotation", cause).
			WithContext("secrets_path", "/tmp/test/secrets.age").
			WithContext("hint", "old key may be corrupted or invalid")
	}
}

func Demo_KairoError_multiple_context() {
	// Error with multiple context values
	err := kairoerrors.WrapError(kairoerrors.ProviderError,
		"provider not available", errors.New("connection timeout")).
		WithContext("provider", "anthropic").
		WithContext("host", "api.anthropic.com").
		WithContext("port", "443").
		WithContext("attempt", "3").
		WithContext("hint", "check network connectivity and firewall settings")

	fmt.Println(err.Error())
	// Output: provider not available: connection timeout (provider=anthropic, host=api.anthropic.com, port=443, attempt=3, hint=check network connectivity and firewall settings)
}

func Demo_KairoError_error_type_checking() {
	err := kairoerrors.WrapError(kairoerrors.ConfigError,
		"invalid configuration", errors.New("missing field"))

	// Can check error type using Is()
	var configErr *kairoerrors.KairoError
	if errors.As(err, &configErr) {
		fmt.Printf("Error type: %s\n", configErr.Type)
		fmt.Printf("Message: %s\n", configErr.Message)
	}
	// Output:
	// Error type: config
	// Message: invalid configuration
}

// Example of structured error handling in CLI
func Demo_structured_error_handling() error {
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Simulate an invalid configuration
	invalidYAML := `
default_provider: "zai"
providers:
  - invalid: [yaml
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		return kairoerrors.WrapError(kairoerrors.FileSystemError,
			"failed to write test config", err).
			WithContext("path", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var cfg interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Before: "yaml: unmarshal errors:\n  line 4: cannot unmarshal !!seq into map[string]interface {}"
		// After: "failed to parse configuration file (invalid YAML): yaml: unmarshal errors:\n  line 4: cannot unmarshal !!seq into map[string]interface {} (path=/tmp/..., hint=check YAML syntax and indentation)"
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to parse configuration file (invalid YAML)", err).
			WithContext("path", configPath).
			WithContext("hint", "check YAML syntax and indentation")
	}

	_ = cfg
	return nil
}
