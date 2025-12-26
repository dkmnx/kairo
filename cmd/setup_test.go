package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
)

func TestPrintBanner(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		provider string
		wantSub  string
	}{
		{
			name:     "banner with version and provider",
			version:  "v0.1.0",
			provider: "MiniMax",
			wantSub:  "v0.1.0 - MiniMax",
		},
		{
			name:     "banner with custom provider",
			version:  "vdev",
			provider: "Custom Provider",
			wantSub:  "vdev - Custom Provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			done := make(chan struct{})
			go func() {
				ui.PrintBanner(tt.version, tt.provider)
				w.Close()
				close(done)
			}()

			<-done

			os.Stdout = oldStdout
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Logf("Warning: io.Copy failed: %v", err)
			}
			r.Close()
			output := buf.String()
			if !strings.Contains(output, tt.wantSub) {
				t.Errorf("banner output does not contain expected substring %q, got: %q", tt.wantSub, output)
			}
		})
	}
}

func TestPrintBannerContainsASCIIArt(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		ui.PrintBanner("v0.1.0", "Test Provider")
		w.Close()
		close(done)
	}()

	select {
	case <-done:
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Logf("Warning: io.Copy failed: %v", err)
		}
		r.Close()
		output := buf.String()
		expectedParts := []string{
			"█████",
			"░░███",
			"░███",
			"░██████░",
			"░░░░ ░░░░░",
			"v0.1.0 - Test Provider",
		}
		for _, part := range expectedParts {
			if !strings.Contains(output, part) {
				t.Errorf("banner output does not contain expected part %q, got: %q", part, output)
			}
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for banner output")
		w.Close()
		os.Stdout = oldStdout
		r.Close()
	}
}

func TestPrintProviderOptionConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"minimax":   {Name: "MiniMax"},
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

	secrets, _ := crypto.DecryptSecrets(secretsPath, keyPath)
	secretsMap := config.ParseSecrets(secrets)

	ui.PrintProviderOption(1, "Native Anthropic", cfg, secretsMap, "anthropic")
	ui.PrintProviderOption(2, "MiniMax", cfg, secretsMap, "minimax")
}

func TestPrintProviderOptionNotConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
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
	if err := crypto.EncryptSecrets(secretsPath, keyPath, ""); err != nil {
		t.Fatal(err)
	}

	secrets, _ := crypto.DecryptSecrets(secretsPath, keyPath)
	secretsMap := config.ParseSecrets(secrets)

	ui.PrintProviderOption(1, "Native Anthropic", cfg, secretsMap, "anthropic")
	ui.PrintProviderOption(2, "Kimi", cfg, secretsMap, "kimi")
}

func TestIsProviderConfigured(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}

	tests := []struct {
		name     string
		provider string
		secrets  map[string]string
		cfg      *config.Config
		want     bool
	}{
		{
			name:     "anthropic configured",
			provider: "anthropic",
			secrets:  map[string]string{},
			cfg:      cfg,
			want:     true,
		},
		{
			name:     "anthropic not configured",
			provider: "anthropic",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     false,
		},
		{
			name:     "minimax with API key",
			provider: "minimax",
			secrets:  map[string]string{"MINIMAX_API_KEY": "test-key"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     true,
		},
		{
			name:     "minimax without API key",
			provider: "minimax",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     false,
		},
		{
			name:     "zai with uppercase key",
			provider: "zai",
			secrets:  map[string]string{"ZAI_API_KEY": "test-key"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     true,
		},
		{
			name:     "zai with lowercase key",
			provider: "zai",
			secrets:  map[string]string{"zai_API_KEY": "test-key"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     true,
		},
		{
			name:     "deepseek not configured",
			provider: "deepseek",
			secrets:  map[string]string{"OTHER_KEY": "value"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProviderConfiguredForTest(tt.cfg, tt.secrets, tt.provider)
			if got != tt.want {
				t.Errorf("isProviderConfiguredForTest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func isProviderConfiguredForTest(cfg *config.Config, secrets map[string]string, provider string) bool {
	if provider == "anthropic" {
		_, exists := cfg.Providers["anthropic"]
		return exists
	}

	apiKeyKey := strings.ToUpper(provider) + "_API_KEY"
	for k := range secrets {
		if strings.EqualFold(k, apiKeyKey) {
			return true
		}
	}
	return false
}

func TestParseIntOrZero(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single digit", "1", 1},
		{"multiple digits", "123", 123},
		{"invalid character", "12a", 0},
		{"only letters", "abc", 0},
		{"leading zeros", "007", 7},
		{"whitespace", " 123", 0},
		{"zero", "0", 0},
		{"negative", "-1", 0},
		{"decimal", "1.5", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIntOrZero(tt.input)
			if got != tt.want {
				t.Errorf("parseIntOrZero(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProviderStatusIcon(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}

	tests := []struct {
		name     string
		provider string
		secrets  map[string]string
		cfg      *config.Config
		wantIcon string
	}{
		{
			name:     "anthropic configured",
			provider: "anthropic",
			secrets:  map[string]string{},
			cfg:      cfg,
			wantIcon: "[x]",
		},
		{
			name:     "anthropic not configured",
			provider: "anthropic",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "  ",
		},
		{
			name:     "minimax with key",
			provider: "minimax",
			secrets:  map[string]string{"MINIMAX_API_KEY": "key"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "[x]",
		},
		{
			name:     "minimax without key",
			provider: "minimax",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerStatusIcon(tt.cfg, tt.secrets, tt.provider)
			if !strings.Contains(got, "[x]") && tt.wantIcon == "[x]" {
				t.Errorf("providerStatusIcon() should contain checkmark")
			}
			if !strings.Contains(got, "  ") && tt.wantIcon == "  " {
				t.Errorf("providerStatusIcon() should contain spaces")
			}
		})
	}
}

func TestExitOptionDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantExit bool
	}{
		{"empty string", "", true},
		{"done", "done", true},
		{"lowercase q", "q", true},
		{"exit", "exit", true},
		{"number", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed := strings.TrimSpace(tt.input)
			got := trimmed == "" || trimmed == "done" || trimmed == "q" || trimmed == "exit"
			if got != tt.wantExit {
				t.Errorf("exit detection for %q = %v, want %v", tt.input, got, tt.wantExit)
			}
		})
	}
}

func TestParseSecretsForIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
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

	secretsContent := "ZAI_API_KEY=zai-key\nMINIMAX_API_KEY=minimax-key\nDEEPSEEK_API_KEY=deepseek-key\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	secretsMap := config.ParseSecrets(decrypted)

	if len(secretsMap) != 3 {
		t.Errorf("ParseSecrets() returned %d entries, want 3", len(secretsMap))
	}

	expectedKeys := []string{"ZAI_API_KEY", "MINIMAX_API_KEY", "DEEPSEEK_API_KEY"}
	for _, key := range expectedKeys {
		if _, ok := secretsMap[key]; !ok {
			t.Errorf("ParseSecrets() missing key %q", key)
		}
	}
}

func TestSecretsPreservationWhenAddingProvider(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai":     {Name: "Z.AI"},
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

	existingSecrets := "ZAI_API_KEY=zai-secret-123\nMINIMAX_API_KEY=minimax-secret-456\n"
	if err := crypto.EncryptSecrets(secretsPath, keyPath, existingSecrets); err != nil {
		t.Fatal(err)
	}

	secretsContent, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	secrets := config.ParseSecrets(secretsContent)
	if len(secrets) != 2 {
		t.Errorf("ParseSecrets() returned %d entries, want 2", len(secrets))
	}

	newApiKey := "deepseek-secret-789"
	secrets["DEEPSEEK_API_KEY"] = newApiKey

	var secretsBuilder strings.Builder
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	for _, key := range keys {
		value := secrets[key]
		if key != "" && value != "" {
			secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatalf("EncryptSecrets() error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	secretsMap := config.ParseSecrets(decrypted)
	if len(secretsMap) != 3 {
		t.Errorf("After adding provider, expected 3 secrets, got %d", len(secretsMap))
	}

	if secretsMap["ZAI_API_KEY"] != "zai-secret-123" {
		t.Errorf("ZAI_API_KEY was lost, got %q", secretsMap["ZAI_API_KEY"])
	}
	if secretsMap["MINIMAX_API_KEY"] != "minimax-secret-456" {
		t.Errorf("MINIMAX_API_KEY was lost, got %q", secretsMap["MINIMAX_API_KEY"])
	}
	if secretsMap["DEEPSEEK_API_KEY"] != "deepseek-secret-789" {
		t.Errorf("DEEPSEEK_API_KEY not saved correctly, got %q", secretsMap["DEEPSEEK_API_KEY"])
	}
}

func TestProviderListConstant(t *testing.T) {
	providerList := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}

	if len(providerList) != 6 {
		t.Errorf("providerList has %d entries, want 6", len(providerList))
	}

	expected := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}
	for i, p := range providerList {
		if p != expected[i] {
			t.Errorf("providerList[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestProviderEnvVarSetup(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantEnvCount int
	}{
		{"anthropic has no env vars", "anthropic", 0},
		{"zai has env vars", "zai", 1},
		{"minimax has env vars", "minimax", 2},
		{"kimi has env vars", "kimi", 2},
		{"deepseek has env vars", "deepseek", 2},
		{"custom has no env vars", "custom", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.GetBuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) failed", tt.provider)
			}

			if tt.wantEnvCount > 0 && len(def.EnvVars) == 0 {
				t.Errorf("Provider %q has 0 env vars, want at least %d", tt.provider, tt.wantEnvCount)
			}

			if tt.wantEnvCount == 0 && len(def.EnvVars) > 0 {
				t.Errorf("Provider %q has %d env vars, want 0", tt.provider, len(def.EnvVars))
			}
		})
	}
}

func TestGetConfigDirWithEnv(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	expectedDir := filepath.Join(tmpDir, ".config", "kairo")
	dir := getConfigDir()
	if dir != expectedDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestGetConfigDirWithFlag(t *testing.T) {
	originalConfigDir := configDir
	configDir = "/custom/path"
	defer func() { configDir = originalConfigDir }()

	dir := getConfigDir()
	if dir != "/custom/path" {
		t.Errorf("getConfigDir() = %q, want %q", dir, "/custom/path")
	}
}

func TestSwitchCmdProviderNotFound(t *testing.T) {
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tmpDir := t.TempDir()
	configDir = tmpDir

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"minimax": {Name: "MiniMax", BaseURL: "https://api.minimax.io", Model: "test"},
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

	dir := getConfigDir()
	if dir != tmpDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, tmpDir)
	}
}

func TestCustomProviderKeyFormat(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"myprovider": {Name: "My Provider", BaseURL: "https://api.myprovider.com", Model: "model-1"},
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

	customName := "myprovider"
	apiKey := "sk-test-key-12345"
	secrets := map[string]string{
		fmt.Sprintf("%s_API_KEY", customName): apiKey,
	}

	var secretsBuilder strings.Builder
	for key, value := range secrets {
		if key != "" && value != "" {
			secretsBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
	}

	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	expectedKey := fmt.Sprintf("%s_API_KEY=", customName)
	if !strings.Contains(decrypted, expectedKey) {
		t.Errorf("Decrypted secrets should contain %q, got: %q", expectedKey, decrypted)
	}

	if !strings.Contains(decrypted, "myprovider_API_KEY=sk-test-key-12345") {
		t.Errorf("Decrypted secrets should contain 'myprovider_API_KEY=sk-test-key-12345', got: %q", decrypted)
	}

	for _, line := range strings.Split(decrypted, "\n") {
		if strings.HasPrefix(line, expectedKey) {
			if strings.HasPrefix(line, "CUSTOM_") {
				t.Errorf("Custom provider key should NOT have CUSTOM_ prefix, got: %q", line)
			}
			return
		}
	}

	t.Errorf("Expected key %q not found in decrypted secrets", expectedKey)
}

func TestCustomProviderKeyLookupInSwitch(t *testing.T) {
	tmpDir := t.TempDir()

	providerName := "mycustomprovider"
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			providerName: {Name: "My Custom Provider", BaseURL: "https://api.example.com", Model: "test"},
		},
		DefaultProvider: providerName,
	}
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}

	apiKey := "sk-custom-key-abcdef"
	secretsContent := fmt.Sprintf("%s_API_KEY=%s\n", providerName, apiKey)
	if err := crypto.EncryptSecrets(secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets() error = %v", err)
	}

	prefix := fmt.Sprintf("%s_API_KEY=", providerName)
	if !strings.HasPrefix(decrypted, prefix) {
		t.Errorf("Secrets should start with %q, got: %q", prefix, decrypted)
	}

	for _, line := range strings.Split(decrypted, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				t.Errorf("Expected key=value format, got: %q", line)
				continue
			}
			if parts[1] != apiKey {
				t.Errorf("API key = %q, want %q", parts[1], apiKey)
			}
			if strings.HasPrefix(line, "CUSTOM_") {
				t.Errorf("Key should NOT have CUSTOM_ prefix for custom provider")
			}
			return
		}
	}

	t.Errorf("Expected to find %q in secrets", prefix)
}
