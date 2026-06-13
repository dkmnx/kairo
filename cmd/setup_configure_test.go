package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
)

func TestConfigureProvider_NewProvider(t *testing.T) {
	in, _ := setupTapTest(t)

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{
		Crypto: &mockCrypto{},
	})

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	secrets := map[string]string{}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "zai",
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- result
	}()

	// configureProvider calls: promptForAPIKey, promptForBaseURL, promptForModel
	// Provider "zai" has a default base URL and model, but we must fill them in.

	time.Sleep(50 * time.Millisecond)
	// promptForAPIKey: enters password
	emitText(in, "sk-zai-test-key-abcdefghijklmnopqrst")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// promptForBaseURL: accept default by returning blank, which uses DefaultValue
	// But actually promptForBaseURL calls promptForField which for new provider
	// returns blank → default value = provider.BaseURL
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// promptForModel: accept default by returning blank
	emitReturn(in)

	result := <-resultCh

	if result == "zai" {
		// Provider was added
		t.Logf("configureProvider succeeded: %s is now configured", result)
	} else {
		t.Errorf("configureProvider() = %q, want %q, cfg providers: %+v, err: %s",
			result, "zai", cfg.Providers, result)
	}

	// Verify provider was added to config.
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
	in, _ := setupTapTest(t)

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{
		Crypto: &mockCrypto{},
	})

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	secrets := map[string]string{}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "zai",
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- result
	}()

	time.Sleep(50 * time.Millisecond)
	// API key
	emitText(in, "sk-zai-custom-key-abcdefghijklmnopqr")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Base URL: accept default
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Model: enter custom model
	emitText(in, "my-custom-model-v2")
	emitReturn(in)

	result := <-resultCh

	if result == "zai" {
		t.Log("configureProvider succeeded")
	} else {
		t.Errorf("configureProvider() = %q, want 'zai', err: %s", result, result)
	}

	if prov, ok := cfg.Providers["zai"]; ok {
		if prov.Model != "my-custom-model-v2" {
			t.Errorf("provider model = %q, want %q", prov.Model, "my-custom-model-v2")
		}
	}
}

func TestConfigureProvider_FirstProviderBecomesDefault(t *testing.T) {
	in, _ := setupTapTest(t)

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{
		Crypto: &mockCrypto{},
	})

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	secrets := map[string]string{}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "zai",
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- result
	}()

	time.Sleep(50 * time.Millisecond)
	// API key
	emitText(in, "sk-zai-default-key-abcdefghijklmnop")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Base URL: default
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Model: default
	emitReturn(in)

	result := <-resultCh

	if result == "zai" {
		t.Log("configureProvider succeeded")
	} else {
		t.Errorf("configureProvider() = %q, want 'zai', err: %s", result, result)
	}

	// First provider should become default.
	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}
}

func TestConfigureProvider_EditExisting(t *testing.T) {
	in, _ := setupTapTest(t)

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)
	cliCtx.SetDeps(&Deps{
		Crypto: &mockCrypto{},
	})

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

	secrets := map[string]string{"ZAI_API_KEY": "sk-existing-key-abcdefghijklmnopqr"}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		result, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "zai",
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- result
	}()

	// Editing existing provider: prompts for each field.
	time.Sleep(50 * time.Millisecond)
	// API key edit: "Modify API key?" -> n (keep existing)
	emitText(in, "n")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Base URL edit: "Modify Base URL?" -> n (keep)
	emitText(in, "n")
	emitReturn(in)

	time.Sleep(50 * time.Millisecond)
	// Model edit: "Modify Model? (current: glm-5)" -> n (keep)
	emitText(in, "n")
	emitReturn(in)

	result := <-resultCh

	if result != "zai" {
		t.Errorf("configureProvider() = %q, want 'zai', err: %s", result, result)
	}

	// Existing provider should still be the default.
	if cfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "zai")
	}
}

func TestConfigureProvider_InvalidAPIKey(t *testing.T) {
	in, _ := setupTapTest(t)

	configDir := t.TempDir()
	cliCtx := NewCLIContext()
	cliCtx.SetConfigDir(configDir)

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	secrets := map[string]string{}
	secretsPath := filepath.Join(configDir, "secrets.age")
	keyPath := filepath.Join(configDir, "key.age")

	resultCh := make(chan string)
	go func() {
		_, err := configureProvider(ProviderSetup{
			CLIContext:   cliCtx,
			ConfigDir:    configDir,
			Cfg:          cfg,
			ProviderName: "anthropic",
			Secrets:      secrets,
			SecretsPath:  secretsPath,
			KeyPath:      keyPath,
		})
		if err != nil {
			resultCh <- "error:" + err.Error()
			return
		}
		resultCh <- "success"
	}()

	time.Sleep(50 * time.Millisecond)
	// Antrhopic requires API key starting with "sk-ant-"
	// Enter an invalid key.
	emitText(in, "invalid-key")
	emitReturn(in)

	result := <-resultCh

	if result == "success" {
		t.Error("expected configureProvider to fail with invalid API key")
	} else {
		t.Logf("configureProvider correctly rejected invalid key: %s", result)
	}

	// Provider should not be added.
	if _, exists := cfg.Providers["anthropic"]; exists {
		t.Error("provider should not be added when API key validation fails")
	}
}
