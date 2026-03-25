package cmd

import (
	"context"

	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/dkmnx/kairo/internal/validate"
)

func providerStatusIcon(cfg *config.Config, secrets map[string]string, provider string) string {
	if !providers.RequiresAPIKey(provider) {
		if _, exists := cfg.Providers[provider]; exists {
			return ui.Green + "[x]" + ui.Reset
		}
		return "  "
	}

	apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
	for k := range secrets {
		if k == apiKeyKey {
			return ui.Green + "[x]" + ui.Reset
		}
	}
	return "  "
}

func TestPrintBanner(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		provider config.Provider
		wantSub  string
	}{
		{
			name:    "banner with version model and provider",
			version: "v0.1.0",
			provider: config.Provider{
				Model: "claude-sonnet-4-20250514",
				Name:  "MiniMax",
			},
			wantSub: "kairo v0.1.0 · claude-sonnet-4-20250514 · MiniMax",
		},
		{
			name:    "banner with custom provider and model",
			version: "vdev",
			provider: config.Provider{
				Model: "custom-model",
				Name:  "Custom Provider",
			},
			wantSub: "kairo vdev · custom-model · Custom Provider",
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

func TestPrintProviderOptionConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
			"minimax":   {Name: "MiniMax"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "MINIMAX_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	secrets, _ := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	secretsMap := config.ParseSecrets(secrets)

	ui.PrintProviderOption(ui.ProviderOption{Number: 1, Name: "Native Anthropic", Config: cfg, Secrets: secretsMap, Provider: "anthropic"})
	ui.PrintProviderOption(ui.ProviderOption{Number: 2, Name: "MiniMax", Config: cfg, Secrets: secretsMap, Provider: "minimax"})
}

func TestPrintProviderOptionNotConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"anthropic": {Name: "Native Anthropic"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, ""); err != nil {
		t.Fatal(err)
	}

	secrets, _ := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	secretsMap := config.ParseSecrets(secrets)

	ui.PrintProviderOption(ui.ProviderOption{Number: 1, Name: "Native Anthropic", Config: cfg, Secrets: secretsMap, Provider: "anthropic"})
	ui.PrintProviderOption(ui.ProviderOption{Number: 2, Name: "Kimi", Config: cfg, Secrets: secretsMap, Provider: "kimi"})
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
			"zai": {Name: "Z.AI"},
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
			name:     "zai configured",
			provider: "zai",
			secrets:  map[string]string{"ZAI_API_KEY": "key"},
			cfg:      cfg,
			wantIcon: "[x]",
		},
		{
			name:     "zai not configured",
			provider: "zai",
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
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	secretsContent := "ZAI_API_KEY=zai-key\nMINIMAX_API_KEY=minimax-key\nDEEPSEEK_API_KEY=deepseek-key\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
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
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	existingSecrets := "ZAI_API_KEY=zai-secret-123\nMINIMAX_API_KEY=minimax-secret-456\n"
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, existingSecrets); err != nil {
		t.Fatal(err)
	}

	secretsContent, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
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

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatalf("EncryptSecrets(context.Background(), ) error = %v", err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
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
	providerList := []string{"anthropic", "zai", "minimax", "deepseek", "kimi", "custom"}

	if len(providerList) != 6 {
		t.Errorf("providerList has %d entries, want 6", len(providerList))
	}

	expected := []string{"anthropic", "zai", "minimax", "deepseek", "kimi", "custom"}
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
	originalUserProfile := os.Getenv("USERPROFILE")
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("USERPROFILE", originalUserProfile)

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	// Reset configDir to empty so getConfigDir() falls back to env.GetConfigDir()
	originalConfigDir := getConfigDir()
	defer setConfigDir(originalConfigDir)
	setConfigDir("")

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(tmpDir, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(tmpDir, ".config", "kairo")
	}
	dir := getConfigDir()
	if dir != expectedDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestGetConfigDirWithFlag(t *testing.T) {
	originalConfigDir := getConfigDir()
	setConfigDir("/custom/path")
	defer setConfigDir(originalConfigDir)

	dir := getConfigDir()
	if dir != "/custom/path" {
		t.Errorf("getConfigDir() = %q, want %q", dir, "/custom/path")
	}
}

func TestGetConfigDirWithFlagAndEnv(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalConfigDir := getConfigDir()
	defer func() {
		os.Setenv("HOME", originalHome)
		setConfigDir(originalConfigDir)
	}()

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	setConfigDir("/custom/path")

	dir := getConfigDir()
	if dir != "/custom/path" {
		t.Errorf("getConfigDir() = %q, want %q (flag should take precedence)", dir, "/custom/path")
	}
}

func TestGetConfigDirEmptyConfigDir(t *testing.T) {
	originalConfigDir := getConfigDir()
	setConfigDir("")
	defer setConfigDir(originalConfigDir)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home directory")
	}

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(home, ".config", "kairo")
	}
	dir := getConfigDir()
	if dir != expectedDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestSwitchCmdProviderNotFound(t *testing.T) {
	originalConfigDir := getConfigDir()
	defer setConfigDir(originalConfigDir)

	tmpDir := t.TempDir()
	setConfigDir(tmpDir)

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"minimax": {Name: "MiniMax", BaseURL: "https://api.minimax.io", Model: "test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "MINIMAX_API_KEY=test-key\n"); err != nil {
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
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
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

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsBuilder.String()); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
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
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	apiKey := "sk-custom-key-abcdef"
	secretsContent := fmt.Sprintf("%s_API_KEY=%s\n", providerName, apiKey)
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, secretsContent); err != nil {
		t.Fatal(err)
	}

	decrypted, err := crypto.DecryptSecrets(context.Background(), secretsPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptSecrets(context.Background(), ) error = %v", err)
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

func TestEnsureConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cliCtx := NewCLIContext()
	err := EnsureConfigDir(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("EnsureConfigDir() error = %v", err)
	}

	keyPath := filepath.Join(tmpDir, "age.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("age.key was not created: %v", err)
	}
}

func TestLoadOrInitializeConfigExisting(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
		DefaultProvider: "zai",
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "zai")
	}
	if _, ok := loadedCfg.Providers["zai"]; !ok {
		t.Errorf("Provider zai not found in loaded config")
	}
}

func TestLoadOrInitializeConfigNew(t *testing.T) {
	tmpDir := t.TempDir()

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() returned unexpected error: %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("LoadConfig() returned nil for non-existent config, want empty config")
	}
	if loadedCfg.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty string", loadedCfg.DefaultProvider)
	}
	if loadedCfg.Providers == nil {
		t.Error("Providers map is nil, want empty map")
	}
}

func TestLoadOrInitializeConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"test": {Name: "Test"},
		},
	}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	cliCtx := NewCLIContext()
	loadedCfg, err := LoadConfig(cliCtx, tmpDir)
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if loadedCfg.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty", loadedCfg.DefaultProvider)
	}
}

func TestLoadSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secretsPath := filepath.Join(tmpDir, "secrets.age")
	keyPath := filepath.Join(tmpDir, "age.key")
	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	secretsOut := result.SecretsPath
	keyOut := result.KeyPath
	secrets := result.Secrets
	if err != nil {
		t.Fatalf("LoadSecrets(context.Background(), ) error = %v", err)
	}
	if secretsOut != secretsPath {
		t.Errorf("secretsPath = %q, want %q", secretsOut, secretsPath)
	}
	if keyOut != keyPath {
		t.Errorf("keyPath = %q, want %q", keyOut, keyPath)
	}
	if secrets["ZAI_API_KEY"] != "test-key" {
		t.Errorf("ZAI_API_KEY = %q, want %q", secrets["ZAI_API_KEY"], "test-key")
	}
}

func TestLoadSecretsNoSecretsFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	if err := config.SaveConfig(context.Background(), tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	result, err := LoadSecrets(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("LoadSecrets() error = %v", err)
	}
	secretsPath := result.SecretsPath
	keyPath := result.KeyPath
	secrets := result.Secrets
	if err != nil {
		t.Fatalf("LoadSecrets(context.Background(), ) error = %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("got %d secrets, want 0", len(secrets))
	}
	if !strings.HasSuffix(secretsPath, "secrets.age") {
		t.Errorf("secretsPath = %q, expected to end with secrets.age", secretsPath)
	}
	if !strings.HasSuffix(keyPath, "age.key") {
		t.Errorf("keyPath = %q, expected to end with age.key", keyPath)
	}
}

func TestLoadSecretsWithCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(secretsPath, []byte("corrupted invalid encrypted data"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSecrets(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("Expected error for corrupted secrets file, got nil")
	}
}

func TestLoadSecretsWithCorruptedKey(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "age.key")
	secretsPath := filepath.Join(tmpDir, "secrets.age")

	if err := crypto.GenerateKey(context.Background(), keyPath); err != nil {
		t.Fatal(err)
	}

	if err := crypto.EncryptSecrets(context.Background(), secretsPath, keyPath, "ZAI_API_KEY=test-key\n"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath, []byte("invalid-key-content"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSecrets(context.Background(), tmpDir)
	if err == nil {
		t.Fatal("Expected error for corrupted key file, got nil")
	}
}

func TestParseProviderSelection(t *testing.T) {
	providerList := providers.GetProviderList()
	if len(providerList) < 1 {
		t.Skip("Not enough providers to test selection")
	}

	tests := []struct {
		name      string
		selection string
		wantOk    bool
	}{
		{"empty string", "", false},
		{"done", "done", false},
		{"lowercase q", "q", false},
		{"exit", "exit", false},
		// Numeric selection removed - Tap TUI handles selection internally
		{"out of range", "99", false},
		{"negative", "-1", false},
		{"text", "abc", false},
		{"valid zai", "zai", true},
		{"valid minimax", "minimax", true},
		{"valid custom", "custom", true},
		{"invalid provider", "invalid-provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := parseProviderSelection(tt.selection)
			if ok != tt.wantOk {
				t.Errorf("parseProviderSelection(%q) ok = %v, want %v", tt.selection, ok, tt.wantOk)
			}
			if tt.wantOk && name == "" {
				t.Errorf("parseProviderSelection(%q) returned empty name when ok=true", tt.selection)
			}
			if !tt.wantOk && name != "" {
				t.Errorf("parseProviderSelection(%q) returned non-empty name %q when ok=false", tt.selection, name)
			}
		})
	}
}

func TestGetEnvValue(t *testing.T) {
	result := getEnvValue("TEST_KEY")
	if result != "" {
		t.Errorf("getEnvValue() = %q, want empty string", result)
	}
}

func TestGetEnvFuncDefault(t *testing.T) {
	original := envGetter
	defer func() { envGetter = original }()

	envGetter = getEnvFunc
	value, ok := envGetter("TEST_KEY")
	if value != "" {
		t.Errorf("getEnvFunc() value = %q, want empty", value)
	}
	if ok {
		t.Error("getEnvFunc() ok = true, want false")
	}
}

func TestGetLatestReleaseURLDefault(t *testing.T) {
	original := envGetter
	defer func() { envGetter = original }()

	envGetter = func(key string) (string, bool) {
		return "", false
	}

	url := getLatestReleaseURL()
	if url != defaultUpdateURL {
		t.Errorf("getLatestReleaseURL() = %q, want %q", url, defaultUpdateURL)
	}
}

func TestGetLatestReleaseURLOverride(t *testing.T) {
	original := envGetter
	defer func() { envGetter = original }()

	envGetter = func(key string) (string, bool) {
		if key == "KAIRO_UPDATE_URL" {
			return "https://custom.example.com/releases/latest", true
		}
		return "", false
	}

	url := getLatestReleaseURL()
	expected := "https://custom.example.com/releases/latest"
	if url != expected {
		t.Errorf("getLatestReleaseURL() = %q, want %q", url, expected)
	}
}

func TestSetup_ProviderNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "simple",
			wantErr: false,
		},
		{
			name:    "provider123",
			wantErr: false,
		},
		{
			name:    "my_provider",
			wantErr: false, // Should allow underscores
		},
		{
			name:    "custom-provider",
			wantErr: false, // Should allow hyphens
		},
		{
			name:    "provider_with_underscores",
			wantErr: false, // Should allow underscores
		},
		{
			name:    "provider-with-hyphens",
			wantErr: false, // Should allow hyphens
		},
		{
			name:    "",
			wantErr: true,
		},
		{
			name:    "123invalid",
			wantErr: true, // Must start with letter
		},
		{
			name:    "_invalid",
			wantErr: true, // Must start with letter
		},
		{
			name:    "-invalid",
			wantErr: true, // Must start with letter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ProviderNameLength(t *testing.T) {
	maxValidName := strings.Repeat("a", 50) // Exactly 50 characters
	invalidName := strings.Repeat("b", 51)  // 51 characters - exceeds max

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "a",
			wantErr: false, // Minimum 1 character
		},
		{
			name:    "valid",
			wantErr: false,
		},
		{
			name:    maxValidName,
			wantErr: false, // Exactly 50 characters - max allowed
		},
		{
			name:    invalidName,
			wantErr: true, // 51 characters - exceeds max length
		},
		{
			name:    "this_provider_name_is_way_too_long_and_exceeds_the_maximum_allowed_length_of_fifty_characters",
			wantErr: true, // Much longer than 50 characters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ProviderNameReservedWords(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "zai",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "minimax",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "deepseek",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "kimi",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "custom",
			wantErr: true, // Reserved - built-in provider
		},
		{
			name:    "ZAI",
			wantErr: true, // Reserved - case-insensitive
		},
		{
			name:    "mycustom",
			wantErr: false, // Not reserved
		},
		{
			name:    "provider",
			wantErr: false, // Not reserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
func TestSetup_ValidateBaseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		providerName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid https url",
			url:          "https://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      false,
		},
		{
			name:         "valid https url with path",
			url:          "https://api.example.com/v1/anthropic",
			providerName: "test-provider",
			wantErr:      false,
		},
		{
			name:         "empty url",
			url:          "",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "cannot be empty",
		},
		{
			name:         "whitespace only url",
			url:          "   ",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "non-https url",
			url:          "http://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "ftp url",
			url:          "ftp://api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "localhost url",
			url:          "https://localhost/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "127.0.0.1 url",
			url:          "https://127.0.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 10.x.x.x",
			url:          "https://10.0.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 172.16-31.x.x",
			url:          "https://172.16.0.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 192.168.x.x",
			url:          "https://192.168.1.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "private ip 169.254.x.x",
			url:          "https://169.254.1.1/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "blocked",
		},
		{
			name:         "invalid url format",
			url:          "not-a-url",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "url without scheme",
			url:          "api.example.com/anthropic",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "HTTPS",
		},
		{
			name:         "url with only scheme",
			url:          "https://",
			providerName: "test-provider",
			wantErr:      true,
			errContains:  "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.ValidateURL(tt.url, tt.providerName)

			if (err != nil) != tt.wantErr {
				t.Errorf("validate.ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("validate.ValidateURL(%q) expected error containing %q, got nil", tt.url, tt.errContains)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("validate.ValidateURL(%q) error = %q, want error containing %q", tt.url, err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestSetup_ValidateModel(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		displayName string
		model       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "",
			wantErr:     true,
			errContains: "model name is required",
		},
		{
			name:        "whitespace only model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "   ",
			wantErr:     true,
			errContains: "model name is required",
		},
		{
			name:        "valid model for custom provider",
			provider:    "custom-provider",
			displayName: "custom-provider",
			model:       "gpt-4-turbo",
			wantErr:     false,
		},
		{
			name:        "empty model for built-in provider",
			provider:    "zai",
			displayName: "Z.AI",
			model:       "",
			wantErr:     false,
		},
		{
			name:        "valid model for built-in provider",
			provider:    "zai",
			displayName: "Z.AI",
			model:       "glm-4.7-flash",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguredModel(tt.model, tt.provider, tt.displayName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfiguredModel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errContains != "" && (err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains))) {
				t.Fatalf("validateConfiguredModel() error = %v, want substring %q", err, tt.errContains)
			}
		})
	}
}

func TestResolveProviderName_NonCustom(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		want         string
		wantErr      bool
	}{
		{
			name:         "builtin provider zai",
			providerName: "zai",
			want:         "zai",
			wantErr:      false,
		},
		{
			name:         "builtin provider anthropic",
			providerName: "anthropic",
			want:         "anthropic",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProviderName(tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProviderName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveProviderName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureConfigDirectory_ErrorPaths(t *testing.T) {
	t.Run("invalid path with permission issue", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "notadir")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Try to create directory inside a file path (should fail)
		invalidPath := filepath.Join(tmpFile.Name(), "config")
		cliCtx := NewCLIContext()
		err = EnsureConfigDir(cliCtx, invalidPath)
		if err == nil {
			t.Error("expected error for invalid config directory path")
		}
	})
}

func TestSaveProviderConfiguration_ValidationErrors(t *testing.T) {
	t.Run("missing config directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := crypto.EnsureKeyExists(context.Background(), tmpDir); err != nil {
			t.Fatalf("EnsureKeyExists() error = %v", err)
		}

		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}

		err := AddAndSaveProvider(AddProviderParams{
			CLIContext:   NewCLIContext(),
			ConfigDir:    "/nonexistent/path/that/cannot/be/created",
			Cfg:          cfg,
			ProviderName: "testprovider",
			Provider: config.Provider{
				Name:    "Test Provider",
				BaseURL: "https://test.com",
				Model:   "test-model",
			},
			SetAsDefault: true,
		})
		if err == nil {
			t.Error("expected error for invalid config directory")
		}
	})
}
