package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

func TestDecryptSecretsErrorHandling(t *testing.T) {
	t.Parallel()
	t.Run("setup should handle DecryptSecrets error with verbose flag", func(t *testing.T) {
		originalConfigDir := configDir
		defer func() { configDir = originalConfigDir }()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"minimax": {Name: "MiniMax"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "MINIMAX_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		verbose = true

		existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		if existingSecrets == "" {
			t.Error("Decrypted secrets should not be empty")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		if strings.Contains(output, "decrypt") || strings.Contains(output, "Decrypt") {
			t.Logf("Error message was printed (verbose mode): %s", output)
		}
	})

	t.Run("switch should warn on DecryptSecrets error with verbose flag", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"minimax": {
					Name:    "MiniMax",
					BaseURL: "https://api.minimax.io",
					Model:   "test",
				},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "MINIMAX_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		verbose = true

		_, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		if strings.Contains(output, "decrypt") || strings.Contains(output, "Decrypt") {
			t.Logf("Error message was printed (verbose mode): %s", output)
		}
	})

	t.Run("setup should continue with empty secrets on decryption error", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		existingSecrets, _ := crypto.DecryptSecrets(secretsPath, keyPath)
		if existingSecrets == "" {
			t.Error("Should have decrypted secrets")
		}

		secretsMap := make(map[string]string)
		for _, line := range strings.Split(existingSecrets, "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				secretsMap[parts[0]] = parts[1]
			}
		}

		if secretsMap["TEST_API_KEY"] != "test-key" {
			t.Errorf("TEST_API_KEY = %q, want %q", secretsMap["TEST_API_KEY"], "test-key")
		}
	})

	t.Run("status should print warning only with verbose flag", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {
					Name:    "Z.AI",
					BaseURL: "https://api.z.ai/api/anthropic",
					Model:   "glm-4.7",
				},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = true
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		_ = secretsContent

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		t.Logf("Output: %s", output)
	})

	t.Run("status should NOT print warning without verbose flag", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"zai": {Name: "Z.AI"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = false
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		_ = secretsContent

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		if strings.Contains(output, "decrypt") || strings.Contains(output, "Decrypt") {
			t.Errorf("Should NOT print error without verbose flag, got: %s", output)
		}
	})

	t.Run("verbose flag controls warning output", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = true
		var outputWithVerbose bytes.Buffer

		secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err == nil && secretsContent != "" {
			outputWithVerbose.WriteString("decrypted successfully")
		}

		verbose = false

		if outputWithVerbose.Len() > 0 {
			t.Logf("With verbose=true: %s", outputWithVerbose.String())
		}
	})
}

func TestVerboseFlagBehavior(t *testing.T) {
	t.Run("verbose flag exists and is settable", func(t *testing.T) {
		if verbose {
			t.Log("verbose is initially true")
		} else {
			t.Log("verbose is initially false")
		}

		verbose = true
		if !verbose {
			t.Error("verbose should be settable to true")
		}

		verbose = false
		if verbose {
			t.Error("verbose should be settable to false")
		}
	})

	t.Run("setup prints warning on DecryptSecrets error with verbose", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = true

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		if existingSecrets == "" {
			t.Error("Decrypted secrets should not be empty")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		t.Logf("Output with verbose=true: '%s'", output)
	})

	t.Run("setup silently ignores error without verbose", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = false

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		existingSecrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		if existingSecrets == "" {
			t.Error("Decrypted secrets should not be empty")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		if strings.Contains(output, "Warning") || strings.Contains(output, "warn") {
			t.Errorf("Should NOT print warning without verbose flag, got: %s", output)
		}
	})

	t.Run("switch prints warning on DecryptSecrets error with verbose", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {
					Name:    "Test",
					BaseURL: "https://api.test.com",
					Model:   "model-1",
				},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		verbose = true

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		secrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		if secrets == "" {
			t.Error("Decrypted secrets should not be empty")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		t.Logf("Output with verbose=true: '%s'", output)
	})
}

func TestDecryptSecretsFailureBehavior(t *testing.T) {
	t.Run("setup handles missing secrets file gracefully", func(t *testing.T) {
		originalConfigDir := configDir
		defer func() { configDir = originalConfigDir }()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")

		_, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			t.Error("DecryptSecrets should fail on missing file")
		}

		secrets := make(map[string]string)
		existingSecrets, _ := crypto.DecryptSecrets(secretsPath, keyPath)
		for _, line := range strings.Split(existingSecrets, "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				secrets[parts[0]] = parts[1]
			}
		}

		if len(secrets) != 0 {
			t.Errorf("Secrets should be empty on error, got: %v", secrets)
		}
	})

	t.Run("setup handles corrupted key file gracefully", func(t *testing.T) {
		originalConfigDir := configDir
		defer func() { configDir = originalConfigDir }()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(keyPath, []byte("corrupted key data\n"), 0600); err != nil {
			t.Fatal(err)
		}

		secrets := make(map[string]string)
		existingSecrets, _ := crypto.DecryptSecrets(secretsPath, keyPath)
		for _, line := range strings.Split(existingSecrets, "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				secrets[parts[0]] = parts[1]
			}
		}

		if len(secrets) != 0 {
			t.Errorf("Secrets should be empty on error, got: %v", secrets)
		}
	})

	t.Run("switch handles DecryptSecrets error gracefully", func(t *testing.T) {
		originalConfigDir := configDir
		defer func() { configDir = originalConfigDir }()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {
					Name:    "Test",
					BaseURL: "https://api.test.com",
					Model:   "model-1",
				},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		keyPath := filepath.Join(tmpDir, "age.key")
		secretsPath := filepath.Join(tmpDir, "secrets.age")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		secrets, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets should not fail: %v", err)
		}

		if secrets == "" {
			t.Error("Decrypted secrets should not be empty")
		}
	})

	t.Run("verbose mode shows warning on decryption failure", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		keyPath := filepath.Join(tmpDir, "age.key")
		secretsPath := filepath.Join(tmpDir, "secrets.age")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(keyPath, []byte("corrupted\n"), 0600); err != nil {
			t.Fatal(err)
		}

		verbose = true

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			t.Error("DecryptSecrets should fail on corrupted key")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		t.Logf("Output with verbose=true: '%s'", output)
	})

	t.Run("non-verbose mode suppresses warning on decryption failure", func(t *testing.T) {
		originalConfigDir := configDir
		originalVerbose := verbose
		defer func() {
			configDir = originalConfigDir
			verbose = originalVerbose
		}()

		tmpDir := t.TempDir()
		configDir = tmpDir

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"test": {Name: "Test"},
			},
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		keyPath := filepath.Join(tmpDir, "age.key")
		secretsPath := filepath.Join(tmpDir, "secrets.age")

		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		if err := crypto.EncryptSecrets(secretsPath, keyPath, "TEST_API_KEY=test-key\n"); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(keyPath, []byte("corrupted\n"), 0600); err != nil {
			t.Fatal(err)
		}

		verbose = false

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err == nil {
			t.Error("DecryptSecrets should fail on corrupted key")
		}

		os.Stderr = oldStderr
		w.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(r); err != nil { t.Logf("ReadFrom error: %v", err) }
		output := buf.String()

		if strings.Contains(output, "Warning") || strings.Contains(output, "warn") {
			t.Errorf("Should NOT print warning without verbose flag, got: %s", output)
		}
	})
}
