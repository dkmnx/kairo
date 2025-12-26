package crypto

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
)

// GenerateKey generates a new X25519 encryption key and saves it to the specified path.
func GenerateKey(keyPath string) error {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	_, err = fmt.Fprintf(keyFile, "%s\n%s\n", key.String(), key.Recipient().String())
	if err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

// EncryptSecrets encrypts the given secrets string using age encryption and saves to the specified path.
func EncryptSecrets(secretsPath, keyPath, secrets string) error {
	recipient, err := loadRecipient(keyPath)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(secretsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create secrets file: %w", err)
	}
	defer file.Close()

	w, err := age.Encrypt(file, recipient)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	_, err = w.Write([]byte(secrets))
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close encryptor: %w", err)
	}

	return nil
}

// DecryptSecrets decrypts the secrets file and returns the plaintext content.
func DecryptSecrets(secretsPath, keyPath string) (string, error) {
	identity, err := loadIdentity(keyPath)
	if err != nil {
		return "", err
	}

	file, err := os.Open(secretsPath)
	if err != nil {
		return "", fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	r, err := age.Decrypt(file, identity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted content: %w", err)
	}

	return buf.String(), nil
}

func loadRecipient(keyPath string) (age.Recipient, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("key file is empty")
	}
	if !scanner.Scan() {
		return nil, fmt.Errorf("key file missing recipient")
	}

	recipient, err := age.ParseX25519Recipient(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipient: %w", err)
	}

	return recipient, nil
}

func loadIdentity(keyPath string) (age.Identity, error) {
	file, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("key file is empty")
	}

	identity, err := age.ParseX25519Identity(scanner.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse identity: %w", err)
	}

	return identity, nil
}

// EnsureKeyExists generates a new encryption key if one doesn't exist at the specified directory.
func EnsureKeyExists(configDir string) error {
	keyPath := filepath.Join(configDir, "age.key")
	_, err := os.Stat(keyPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return GenerateKey(keyPath)
}
