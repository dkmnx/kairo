package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConfigCache(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)

	// First load - should return error (no config exists)
	_, err := cache.Get(context.Background(), "testdir")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}

	// Just test cache structure
	if cache.ttl != 5*time.Minute {
		t.Error("TTL not set correctly")
	}
}

func TestConfigCache_GetAndInvalidate(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config file
	configContent := `default_provider: test
providers: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// First load - should load from disk
	cfg1, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if cfg1.DefaultProvider != "test" {
		t.Errorf("DefaultProvider = %q, want %q", cfg1.DefaultProvider, "test")
	}

	// Second load - should return cached config
	cfg2, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() second call error = %v", err)
	}
	// Should be the same pointer (cached)
	if cfg1 != cfg2 {
		t.Error("Second Get() should return cached config (same pointer)")
	}

	// Invalidate
	cache.Invalidate(tmpDir)

	// Load again - should reload from disk
	cfg3, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() after invalidation error = %v", err)
	}
	// Should be different pointer (reloaded)
	if cfg1 == cfg3 {
		t.Error("Get() after invalidation should reload config (different pointer)")
	}
}

func TestConfigCache_TTLExpiry(t *testing.T) {
	cache := NewConfigCache(100 * time.Millisecond)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config file
	configContent := `default_provider: test
providers: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Load to cache
	cfg1, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Load again - should reload (TTL expired)
	cfg2, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() after TTL expiry error = %v", err)
	}
	// Should be different pointer (reloaded)
	if cfg1 == cfg2 {
		t.Error("Get() after TTL expiry should reload config (different pointer)")
	}
}

func TestConfigCache_ConcurrentAccess(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config file
	configContent := `default_provider: test
providers: {}
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

	// Create initial config file
	configContent := `default_provider: test
providers: {}
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
			// Verify we got a valid config
			if cfg == nil {
				errs <- fmt.Errorf("nil config returned")
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for any errors
	for err := range errs {
		t.Errorf("Concurrent write error: %v", err)
	}
}

func TestConfigCache_Metrics(t *testing.T) {
	cache := NewConfigCache(5 * time.Minute)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_provider: test
providers: {}
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Initial metrics should be zero
	m := cache.GetMetrics()
	if m.Hits != 0 || m.Misses != 0 || m.Evictions != 0 {
		t.Errorf("Initial metrics should be zero, got: hits=%d, misses=%d, evictions=%d", m.Hits, m.Misses, m.Evictions)
	}

	// First load - miss
	_, err := cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	m = cache.GetMetrics()
	if m.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", m.Misses)
	}

	// Second load - hit
	_, err = cache.Get(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	m = cache.GetMetrics()
	if m.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", m.Hits)
	}

	// Invalidate - eviction
	cache.Invalidate(tmpDir)
	m = cache.GetMetrics()
	if m.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", m.Evictions)
	}
}

func TestCacheMetrics_HitRate(t *testing.T) {
	tests := []struct {
		name     string
		metrics  CacheMetrics
		expected float64
	}{
		{"zero requests", CacheMetrics{}, 0},
		{"all hits", CacheMetrics{Hits: 10, Misses: 0}, 1.0},
		{"all misses", CacheMetrics{Hits: 0, Misses: 10}, 0.0},
		{"half hits", CacheMetrics{Hits: 5, Misses: 5}, 0.5},
		{"75% hit rate", CacheMetrics{Hits: 75, Misses: 25}, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metrics.HitRate()
			if got != tt.expected {
				t.Errorf("HitRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
