package cmd

import (
	"context"
	stderrors "errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
)

// failingEncryptService injects an EncryptSecrets failure to verify that a
// secrets-save failure prevents the provider from being committed to config.
type failingEncryptService struct {
	crypto.DefaultService
}

func (failingEncryptService) EncryptSecrets(context.Context, string, string, string) error {
	return stderrors.New("injected encrypt failure")
}

// TestConfigureProvider_SecretFailurePreventsConfigWrite verifies the
// persistence order: secrets are written before config.yaml, so a failure
// saving secrets leaves the config untouched (no half-configured provider).
func TestConfigureProvider_SecretFailurePreventsConfigWrite(t *testing.T) {
	in, out := setupTapTest(t)
	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{Crypto: &failingEncryptService{}})

	cfg := &config.Config{
		Providers: map[string]config.Provider{},
	}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "zai",
			Secrets:      map[string]string{},
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()

			return
		}
		resultCh <- result
	}()

	typeText(t, in, out, "API Key", "sk-zai-test-key-abcdefghijklmnopqrst")
	pressEnter(t, in, out, "Base URL")
	pressEnter(t, in, out, "Model")

	if result := <-resultCh; !strings.HasPrefix(result, "error:") {
		t.Fatalf("expected error from failing EncryptSecrets, got: %q", result)
	}

	if _, exists := cfg.Providers["zai"]; exists {
		t.Error("provider must not be committed to config when secrets save fails")
	}
}

func TestConfigureProvider_NewProvider(t *testing.T) {
	in, out, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	typeText(t, in, out, "API Key", "sk-zai-test-key-abcdefghijklmnopqrst")
	pressEnter(t, in, out, "Base URL")
	pressEnter(t, in, out, "Model")

	if result := <-resultCh; result != "zai" {
		t.Fatalf("configureProvider() = %q, want 'zai'", result)
	}

	prov, exists := cfg.Providers["zai"]
	if !exists {
		t.Fatal("expected provider 'zai' to exist in config")
	}
	if prov.Name == "" {
		t.Error("expected provider to have a name")
	}
	if prov.Model == "" {
		t.Error("expected provider to have a model")
	}
}

func TestConfigureProvider_NewProviderCustomModel(t *testing.T) {
	in, out, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	typeText(t, in, out, "API Key", "sk-zai-custom-key-abcdefghijklmnopqr")
	pressEnter(t, in, out, "Base URL")
	typeText(t, in, out, "Model", "my-custom-model-v2")

	if result := <-resultCh; result != "zai" {
		t.Fatalf("configureProvider() = %q, want 'zai'", result)
	}

	if prov, ok := cfg.Providers["zai"]; ok {
		if prov.Model != "my-custom-model-v2" {
			t.Errorf("provider model = %q, want %q", prov.Model, "my-custom-model-v2")
		}
	}
}

func TestConfigureProvider_FirstProviderBecomesDefault(t *testing.T) {
	in, out, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	typeText(t, in, out, "API Key", "sk-zai-default-key-abcdefghijklmnop")
	pressEnter(t, in, out, "Base URL")
	pressEnter(t, in, out, "Model")

	if result := <-resultCh; result != "zai" {
		t.Fatalf("configureProvider() = %q, want 'zai'", result)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}
}

func TestConfigureProvider_EditExisting(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {
				Name:    "Z.AI",
				BaseURL: "https://api.z.ai",
				Model:   "glm-5",
			},
		},
	}
	// Seed before launching: configureProvider holds the map reference across
	// the goroutine boundary, so post-launch writes would race.
	secrets := map[string]string{"ZAI_API_KEY": "sk-existing-key-abcdefghijklmnopqr"}
	in, out, cfg, resultCh := startConfigureProviderWithSecrets(t, "zai", cfg, secrets)

	answerConfirm(t, in, out, "Modify API key?", "n")
	answerConfirm(t, in, out, "Modify Base URL?", "n")
	answerConfirm(t, in, out, "Modify Model?", "n")

	if result := <-resultCh; result != "zai" {
		t.Fatalf("configureProvider() = %q, want 'zai'", result)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}
}

func TestConfigureProvider_InvalidAPIKey(t *testing.T) {
	in, out, cfg, resultCh := startConfigureProvider(t, "anthropic", nil)

	typeText(t, in, out, "API Key", "invalid-key")

	result := <-resultCh
	if !strings.HasPrefix(result, "error:") {
		t.Fatalf("expected configureProvider to fail with invalid API key, got %q", result)
	}

	if _, exists := cfg.Providers["anthropic"]; exists {
		t.Error("provider should not be added when API key validation fails")
	}
}
