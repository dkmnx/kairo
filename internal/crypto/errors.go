package crypto

import "errors"

var (
	ErrDecryptionFailed    = errors.New("failed to decrypt secrets")
	ErrEncryptionFailed    = errors.New("failed to encrypt secrets")
	ErrKeyGenerationFailed = errors.New("failed to generate encryption key")
)
