package providers

import (
	"sort"
	"testing"
)

func TestGetProviderList(t *testing.T) {
	providers := GetProviderList()

	expected := []string{"anthropic", "custom", "deepseek", "kimi", "minimax", "zai"}

	if len(providers) != len(expected) {
		t.Errorf("GetProviderList() returned %d providers, want %d", len(providers), len(expected))
	}

	sort.Strings(providers)
	sort.Strings(expected)

	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("GetProviderList()[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestGetProviderListContainsAllBuiltIns(t *testing.T) {
	providers := GetProviderList()

	allBuiltins := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}

	for _, name := range allBuiltins {
		found := false
		for _, p := range providers {
			if p == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetProviderList() should contain %q", name)
		}
	}
}

func TestGetProviderListNoDuplicates(t *testing.T) {
	providers := GetProviderList()

	seen := make(map[string]bool)
	for _, p := range providers {
		if seen[p] {
			t.Errorf("GetProviderList() contains duplicate: %q", p)
		}
		seen[p] = true
	}
}

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"anthropic does not require API key", "anthropic", false},
		{"zai requires API key", "zai", true},
		{"minimax requires API key", "minimax", true},
		{"kimi requires API key", "kimi", true},
		{"deepseek requires API key", "deepseek", true},
		{"custom requires API key", "custom", true},
		{"unknown provider defaults to true", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiresAPIKey(tt.provider)
			if got != tt.want {
				t.Errorf("RequiresAPIKey(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}
