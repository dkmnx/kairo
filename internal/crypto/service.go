package crypto

import "context"

type Service interface {
	GenerateKey(ctx context.Context, keyPath string) error
	EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error
	DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error)
	DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error)
	EnsureKeyExists(ctx context.Context, configDir string) error
}

type DefaultService struct{}

func (DefaultService) GenerateKey(ctx context.Context, keyPath string) error {
	return GenerateKey(ctx, keyPath)
}

func (DefaultService) EncryptSecrets(ctx context.Context, secretsPath, keyPath, secrets string) error {
	return EncryptSecrets(ctx, secretsPath, keyPath, secrets)
}

func (DefaultService) DecryptSecrets(ctx context.Context, secretsPath, keyPath string) (string, error) {
	return DecryptSecrets(ctx, secretsPath, keyPath)
}

func (DefaultService) DecryptSecretsBytes(ctx context.Context, secretsPath, keyPath string) ([]byte, error) {
	return DecryptSecretsBytes(ctx, secretsPath, keyPath)
}

func (DefaultService) EnsureKeyExists(ctx context.Context, configDir string) error {
	return EnsureKeyExists(ctx, configDir)
}
