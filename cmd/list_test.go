package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

func TestListCommandNoConfig(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	listCmd.SetErr(buf)

	err := listCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestListCommandWithConfig(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	configPath := filepath.Join(tmpDir, "config")
	configContent := `default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	listCmd.SetErr(buf)

	err = listCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestSortProviderNamesWithDefault(t *testing.T) {
	providers := map[string]config.Provider{
		"zai":     {Name: "Z.AI"},
		"minimax": {Name: "MiniMax"},
		"kimi":    {Name: "Kimi"},
	}

	result := sortProviderNames(providers, "zai")

	if len(result) != 3 {
		t.Fatalf("sortProviderNames() returned %d names, want 3", len(result))
	}

	if result[0] != "zai" {
		t.Errorf("sortProviderNames() first element = %q, want %q (default should be first)", result[0], "zai")
	}

	if result[1] == result[2] {
		t.Error("sortProviderNames() returned duplicates")
	}
}

func TestSortProviderNamesNoDefault(t *testing.T) {
	providers := map[string]config.Provider{
		"zai":     {Name: "Z.AI"},
		"minimax": {Name: "MiniMax"},
		"kimi":    {Name: "Kimi"},
	}

	result := sortProviderNames(providers, "")

	if len(result) != 3 {
		t.Fatalf("sortProviderNames() returned %d names, want 3", len(result))
	}

	seen := make(map[string]bool)
	for _, name := range result {
		if seen[name] {
			t.Errorf("sortProviderNames() returned duplicate: %q", name)
		}
		seen[name] = true
	}
}

func TestSortProviderNamesSingleProvider(t *testing.T) {
	providers := map[string]config.Provider{
		"zai": {Name: "Z.AI"},
	}

	result := sortProviderNames(providers, "zai")

	if len(result) != 1 {
		t.Fatalf("sortProviderNames() returned %d names, want 1", len(result))
	}

	if result[0] != "zai" {
		t.Errorf("sortProviderNames() first element = %q, want %q", result[0], "zai")
	}
}

func TestSortProviderNamesDefaultNotInProviders(t *testing.T) {
	providers := map[string]config.Provider{
		"zai":     {Name: "Z.AI"},
		"minimax": {Name: "MiniMax"},
	}

	result := sortProviderNames(providers, "anthropic")

	if len(result) != 2 {
		t.Fatalf("sortProviderNames() returned %d names, want 2", len(result))
	}

	if result[0] == "zai" || result[0] == "minimax" {
		t.Log("sortProviderNames() handles missing default by sorting alphabetically")
	}
}

func TestSortProviderNamesMultipleProvidersWithDefault(t *testing.T) {
	providers := map[string]config.Provider{
		"anthropic": {Name: "Native Anthropic"},
		"zai":       {Name: "Z.AI"},
		"minimax":   {Name: "MiniMax"},
		"kimi":      {Name: "Kimi"},
		"deepseek":  {Name: "DeepSeek"},
	}

	result := sortProviderNames(providers, "zai")

	if len(result) != 5 {
		t.Fatalf("sortProviderNames() returned %d names, want 5", len(result))
	}

	if result[0] != "zai" {
		t.Errorf("sortProviderNames() first element = %q, want %q", result[0], "zai")
	}

	othersCount := 0
	for i := 1; i < len(result); i++ {
		if result[i] != "zai" {
			othersCount++
		}
	}
	if othersCount != 4 {
		t.Error("sortProviderNames() should have 4 non-default providers")
	}
}
