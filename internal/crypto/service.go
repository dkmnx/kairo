package crypto

import "context"

// Service defines the encryption operations contract. Production code uses
// DefaultService; tests can inject mocks.
type Service interface {
	GenerateKey(ctx context.Context, keyPath string) error
	EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error
	DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error)
	DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error)
	EnsureKeyExists(ctx context.Context, configDir string) error
}

// DefaultService is the production implementation backed by the age library.
type DefaultService struct{}

// GenerateKey delegates to the package-level GenerateKey.
func (DefaultService) GenerateKey(ctx context.Context, keyPath string) error {
	return GenerateKey(ctx, keyPath)
}

// EncryptSecrets delegates to the package-level EncryptSecrets.
func (DefaultService) EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error {
	return EncryptSecrets(ctx, secretsPath, keyPath, secrets)
}

// DecryptSecrets delegates to the package-level DecryptSecrets.
func (DefaultService) DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error) {
	return DecryptSecrets(ctx, secretsPath, keyPath)
}

// DecryptSecretsBytes delegates to the package-level DecryptSecretsBytes.
func (DefaultService) DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error) {
	return DecryptSecretsBytes(ctx, secretsPath, keyPath)
}

// EnsureKeyExists delegates to the package-level EnsureKeyExists.
func (DefaultService) EnsureKeyExists(ctx context.Context, configDir string) error {
	return EnsureKeyExists(ctx, configDir)
}
