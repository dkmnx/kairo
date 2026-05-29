package crypto

import (
	"context"
	"path/filepath"
	"testing"
)

func BenchmarkEncryptSecrets(b *testing.B) {
	tmpDir := b.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		b.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "ZAI_API_KEY=sk-test-key-1234567890\nMINIMAX_API_KEY=sk-another-key-0987654321\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := EncryptSecrets(context.Background(), secretsPath, keyPath, secrets); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptSecrets(b *testing.B) {
	tmpDir := b.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		b.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "ZAI_API_KEY=sk-test-key-1234567890\nMINIMAX_API_KEY=sk-another-key-0987654321\n"
	if err := EncryptSecrets(context.Background(), secretsPath, keyPath, secrets); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DecryptSecrets(context.Background(), secretsPath, keyPath); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateKey(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyPath := filepath.Join(tmpDir, "bench.key")
		if err := GenerateKey(context.Background(), keyPath); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncryptDecryptRoundtrip(b *testing.B) {
	tmpDir := b.TempDir()
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := GenerateKey(context.Background(), keyPath); err != nil {
		b.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	secrets := "ZAI_API_KEY=sk-test-key-1234567890\nMINIMAX_API_KEY=sk-another-key-0987654321\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := EncryptSecrets(context.Background(), secretsPath, keyPath, secrets); err != nil {
			b.Fatal(err)
		}
		if _, err := DecryptSecrets(context.Background(), secretsPath, keyPath); err != nil {
			b.Fatal(err)
		}
	}
}
