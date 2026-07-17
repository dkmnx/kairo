package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/providers"
)

func TestConfigCache(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)

	_, err := cache.Get(context.Background(), "testdir")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}

	if cache.ttl != 5*time.Minute {
		t.Error("TTL not set correctly")
	}
}

func TestConfigCache_GetAndInvalidate(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: test
providers:
  test:
    name: test
    base_url: https://api.test.com
    model: test-model
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg1, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if cfg1.DefaultProvider != "test" {
		t.Errorf("DefaultProvider = %q, want %q", cfg1.DefaultProvider, "test")
	}

	// Second load - should return cached config (deep copy to prevent mutation)
	cfg2, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() second call error = %v", err)
	}
	if cfg1 == cfg2 {
		t.Error("Second Get() should return a deep copy, not the same pointer")
	}
	if cfg1.DefaultProvider != cfg2.DefaultProvider {
		t.Errorf("Second Get() DefaultProvider = %q, want %q", cfg2.DefaultProvider, cfg1.DefaultProvider)
	}

	// Invalidate
	cache.Invalidate(tmpDir)

	cfg3, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() after invalidation error = %v", err)
	}
	if cfg1 == cfg3 {
		t.Error("Get() after invalidation should reload config (different pointer)")
	}
}

func TestConfigCache_TTLExpiry(t *testing.T) {
	cache := NewConfigCache(100 * time.Millisecond)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: test
providers:
  test:
    name: test
    base_url: https://api.test.com
    model: test-model
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg1, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	cfg2, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() after TTL expiry error = %v", err)
	}
	if cfg1 == cfg2 {
		t.Error("Get() after TTL expiry should reload config (different pointer)")
	}
}

func TestConfigCache_ConcurrentAccess(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: test
providers:
  test:
    name: test
    base_url: https://api.test.com
    model: test-model
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run concurrent accesses
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = cache.Get(context.Background(), tmpDir)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConfigCache_InvalidateNonExistent(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)

	// Invalidate should not panic for non-existent entries
	cache.Invalidate("nonexistent")
}

func TestConfigCache_ConcurrentWrites(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()

	configContent := `default_provider: test
providers:
  test:
    name: test
    base_url: https://api.test.com
    model: test-model
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Run concurrent writes (simulate config modification)
	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			// Invalidate and reload - simulates config modification
			cache.Invalidate(tmpDir)
			cfg, err := cache.Get(context.Background(), tmpDir)
			if err != nil {
				errs <- err
				return
			}
			if cfg == nil {
				errs <- fmt.Errorf("nil config returned")
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("Concurrent write error: %v", err)
	}
}

func TestDeepCopyConfig_IsDeep(t *testing.T) {
	original := &Config{
		DefaultProvider: "anthropic",
		DefaultHarness:  "claude",
		Providers: map[string]Provider{
			"anthropic": {
				Name:    "Anthropic",
				BaseURL: "https://api.anthropic.com",
				Model:   "claude-sonnet-4-20250514",
				EnvKey:  "ANTHROPIC_API_KEY",
				EnvVars: []string{"CUSTOM_VAR=val"},
			},
		},
		DefaultModels: map[string]string{
			"anthropic": "claude-sonnet-4-20250514",
		},
		CustomProviders: map[string]providers.CustomProviderDefinition{
			"custom": {Name: "Custom"},
		},
	}

	copy := deepCopyConfig(original)

	// Modify the copy
	copy.DefaultProvider = "openai"
	copy.DefaultHarness = "qwen"
	anthCopy := copy.Providers["anthropic"]
	anthCopy.Name = "Changed"
	anthCopy.EnvVars[0] = "CHANGED"
	copy.Providers["anthropic"] = anthCopy
	copy.Providers["openai"] = Provider{Name: "OpenAI"}
	copy.DefaultModels["anthropic"] = "changed"
	copy.DefaultModels["openai"] = "gpt-4"
	customCopy := copy.CustomProviders["custom"]
	customCopy.Name = "Changed"
	copy.CustomProviders["custom"] = customCopy
	copy.CustomProviders["new"] = providers.CustomProviderDefinition{Name: "New"}

	// Verify original is unchanged
	if original.DefaultProvider != "anthropic" {
		t.Errorf("DefaultProvider was mutated: got %q, want %q", original.DefaultProvider, "anthropic")
	}
	if original.DefaultHarness != "claude" {
		t.Errorf("DefaultHarness was mutated: got %q, want %q", original.DefaultHarness, "claude")
	}
	if original.Providers["anthropic"].Name != "Anthropic" {
		t.Errorf("Provider.Name was mutated: got %q, want %q", original.Providers["anthropic"].Name, "Anthropic")
	}
	if original.Providers["anthropic"].EnvVars[0] != "CUSTOM_VAR=val" {
		t.Errorf("Provider.EnvVars was mutated: got %q, want %q", original.Providers["anthropic"].EnvVars[0], "CUSTOM_VAR=val")
	}
	if _, ok := original.Providers["openai"]; ok {
		t.Error("Provider map was mutated: new entry appeared")
	}
	if original.DefaultModels["anthropic"] != "claude-sonnet-4-20250514" {
		t.Errorf("DefaultModels was mutated: got %q, want %q", original.DefaultModels["anthropic"], "claude-sonnet-4-20250514")
	}
	if _, ok := original.DefaultModels["openai"]; ok {
		t.Error("DefaultModels map was mutated: new entry appeared")
	}
	if original.CustomProviders["custom"].Name != "Custom" {
		t.Errorf("CustomProviders was mutated: got %q, want %q", original.CustomProviders["custom"].Name, "Custom")
	}
	if _, ok := original.CustomProviders["new"]; ok {
		t.Error("CustomProviders map was mutated: new entry appeared")
	}
}
