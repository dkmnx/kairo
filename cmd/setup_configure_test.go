package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
)

func TestConfigureProvider_NewProvider(t *testing.T) {
	in, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "sk-zai-test-key-abcdefghijklmnopqrst")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

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
	in, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "sk-zai-custom-key-abcdefghijklmnopqr")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "my-custom-model-v2")
	emitReturn(in)

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
	in, cfg, resultCh := startConfigureProvider(t, "zai", nil)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "sk-zai-default-key-abcdefghijklmnop")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

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
	in, cfg, resultCh := startConfigureProviderWithSecrets(t, "zai", cfg, secrets)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "n")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "n")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "n")
	emitReturn(in)

	if result := <-resultCh; result != "zai" {
		t.Fatalf("configureProvider() = %q, want 'zai'", result)
	}

	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}
}

func TestConfigureProvider_InvalidAPIKey(t *testing.T) {
	in, cfg, resultCh := startConfigureProvider(t, "anthropic", nil)

	time.Sleep(50 * time.Millisecond)
	emitText(in, "invalid-key")
	emitReturn(in)

	result := <-resultCh
	if !strings.HasPrefix(result, "error:") {
		t.Fatalf("expected configureProvider to fail with invalid API key, got %q", result)
	}

	if _, exists := cfg.Providers["anthropic"]; exists {
		t.Error("provider should not be added when API key validation fails")
	}
}
