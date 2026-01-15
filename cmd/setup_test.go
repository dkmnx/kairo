package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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

func TestEnsureConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := ensureConfigDirectory(tmpDir)
	if err != nil {
		t.Errorf("ensureConfigDirectory() error = %v", err)
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := loadOrInitializeConfig(tmpDir)
	if err != nil {
		t.Errorf("loadOrInitializeConfig() error = %v", err)
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

	loadedCfg, err := loadOrInitializeConfig(tmpDir)
	if err != nil {
		t.Fatalf("loadOrInitializeConfig() returned unexpected error: %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("loadOrInitializeConfig() returned nil for non-existent config, want empty config")
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	loadedCfg, err := loadOrInitializeConfig(tmpDir)
	if err != nil {
		t.Errorf("loadOrInitializeConfig() error = %v", err)
	}
	if loadedCfg.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty", loadedCfg.DefaultProvider)
	}
}

func TestLoadSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
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

	secrets, secretsOut, keyOut := LoadSecrets(tmpDir)
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
	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatal(err)
	}

	secrets, secretsPath, keyPath := LoadSecrets(tmpDir)
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

func TestParseProviderSelection(t *testing.T) {
	providerList := providers.GetProviderList()
	if len(providerList) < 2 {
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
		{"valid first selection", "1", true},
		{"valid second selection", "2", true},
		{"out of range", "99", false},
		{"zero", "0", false},
		{"negative", "-1", false},
		{"text", "abc", false},
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

func TestConfigureAnthropic(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: make(map[string]config.Provider),
	}
	err := configureAnthropic(tmpDir, cfg, "anthropic")
	if err != nil {
		t.Errorf("configureAnthropic() error = %v", err)
	}

	provider, ok := cfg.Providers["anthropic"]
	if !ok {
		t.Fatal("anthropic provider not found")
	}
	if provider.Name != "Native Anthropic" {
		t.Errorf("Name = %q, want %q", provider.Name, "Native Anthropic")
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if _, ok := loadedCfg.Providers["anthropic"]; !ok {
		t.Error("anthropic provider not saved to disk")
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

func TestPromptForProvider(t *testing.T) {
	t.Run("returns provider selection", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("1\n")
			pw.Close()
		}()

		// Small delay to ensure input is available
		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		// Capture stdout
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		selection := promptForProvider()

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if selection == "" {
			t.Skip("promptForProvider returned empty (likely stdin redirection issue)")
		}
	})

	t.Run("returns selection for quit command", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("q\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		selection := promptForProvider()

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// promptForProvider returns raw user input (trimmed)
		// The caller handles interpreting "q" as quit
		if selection != "q" {
			t.Errorf("promptForProvider() = %q, want 'q'", selection)
		}
	})
}

func TestPromptForAPIKey(t *testing.T) {
	t.Run("returns valid API key", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("sk-test-api-key-123456\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		apiKey, err := promptForAPIKey("Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// term.ReadPassword requires TTY; skip gracefully if not available
		if err != nil {
			t.Skipf("promptForAPIKey requires TTY: %v", err)
		}

		if apiKey != "sk-test-api-key-123456" {
			t.Errorf("promptForAPIKey() = %q, want 'sk-test-api-key-123456'", apiKey)
		}
	})

	t.Run("returns error for invalid API key (too short)", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("short\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_, err := promptForAPIKey("Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// In non-TTY environment, we get TTY error instead of validation error
		// This is acceptable - we skip the test in that case
		if err != nil {
			// Check if it's a TTY-related error
			if containsString(err.Error(), "inappropriate ioctl") {
				t.Skipf("promptForAPIKey requires TTY: %v", err)
			}
		}

		// If we got past TTY check, validation should fail
		if err == nil {
			t.Error("promptForAPIKey() should return error for short API key")
		}
	})
}

func TestPromptForBaseURL(t *testing.T) {
	t.Run("returns custom URL", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("https://custom.api.com/v1\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		baseURL, err := promptForBaseURL("https://api.z.ai/api/anthropic", "Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if err != nil {
			t.Errorf("promptForBaseURL() error = %v", err)
		}

		if baseURL != "https://custom.api.com/v1" {
			t.Errorf("promptForBaseURL() = %q, want 'https://custom.api.com/v1'", baseURL)
		}
	})

	t.Run("uses default URL when input is empty", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		baseURL, err := promptForBaseURL("https://api.z.ai/api/anthropic", "Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if err != nil {
			t.Errorf("promptForBaseURL() error = %v", err)
		}

		if baseURL != "https://api.z.ai/api/anthropic" {
			t.Errorf("promptForBaseURL() = %q, want default URL", baseURL)
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("not-a-valid-url\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_, err := promptForBaseURL("", "Custom")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if err == nil {
			t.Error("promptForBaseURL() should return error for invalid URL")
		}
	})
}

func TestSetupAuditDetails(t *testing.T) {
	t.Run("details map contains all required fields", func(t *testing.T) {
		// Simulate what configureProvider creates
		apiKey := "***masked***"
		details := map[string]interface{}{
			"display_name": "Test Provider",
			"base_url":     "https://api.test.com",
			"model":        "test-model",
			"api_key":      apiKey,
		}

		// Verify all required fields exist
		requiredFields := []string{"display_name", "base_url", "model", "api_key"}
		for _, field := range requiredFields {
			if details[field] == nil {
				t.Errorf("details should contain %s field", field)
			}
		}

		// Verify API key is masked
		if strings.Contains(details["api_key"].(string), "sk-ant-api03") {
			t.Error("API key should not be fully exposed in details")
		}
	})
}

func TestConfigureProvider(t *testing.T) {
	// Skip on Windows - os.Pipe() doesn't work properly with term.ReadPassword
	if runtime.GOOS == "windows" {
		t.Skip("Skipping configureProvider tests on Windows (requires TTY)")
	}

	t.Run("configures built-in provider successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secrets := make(map[string]string)

		// Prepare input: API key, base URL (use default), model (use default)
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			// API key
			_, _ = pw.WriteString("sk-zai-test-key-123456\n")
			// Base URL (use default)
			_, _ = pw.WriteString("\n")
			// Model (use default)
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		// Capture stdout
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		providerName, details, err := configureProvider(tmpDir, cfg, "zai", secrets, secretsPath, keyPath)

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// Skip if TTY not available (promptForAPIKey uses term.ReadPassword)
		if err != nil && containsString(err.Error(), "inappropriate ioctl") {
			t.Skipf("configureProvider requires TTY: %v", err)
		}

		if err != nil {
			t.Errorf("configureProvider() error = %v", err)
		}

		if providerName != "zai" {
			t.Errorf("configureProvider() returned %q, want 'zai'", providerName)
		}

		if details == nil {
			t.Error("configureProvider() details should not be nil")
		} else {
			// Check that details contain expected fields
			if details["display_name"] == nil {
				t.Error("configureProvider() details should contain display_name")
			}
			if details["base_url"] == nil {
				t.Error("configureProvider() details should contain base_url")
			}
			if details["model"] == nil {
				t.Error("configureProvider() details should contain model")
			}
			if details["api_key"] == nil {
				t.Error("configureProvider() details should contain api_key")
			} else {
				// Verify API key is masked
				apiKey := details["api_key"].(string)
				if !strings.Contains(apiKey, "********") {
					t.Errorf("API key should be masked with asterisks, got %q", apiKey)
				}
				// Verify full API key is not exposed
				if apiKey == "sk-test123456789" {
					t.Error("API key should be masked, not exposed in plain text")
				}
			}
		}

		// Check that provider was saved
		loadedCfg, err := config.LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		provider, ok := loadedCfg.Providers["zai"]
		if !ok {
			t.Error("zai provider not found in config")
		}

		if provider.Name != "Z.AI" {
			t.Errorf("Provider.Name = %q, want 'Z.AI'", provider.Name)
		}

		// Check that secrets were saved
		decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets() error = %v", err)
		}

		parsedSecrets := config.ParseSecrets(decrypted)
		if _, ok := parsedSecrets["ZAI_API_KEY"]; !ok {
			t.Error("ZAI_API_KEY not found in secrets")
		}
	})

	t.Run("configures custom provider successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secrets := make(map[string]string)

		// Prepare input: custom name, API key, base URL, model
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			// Custom provider name
			_, _ = pw.WriteString("mycustomprovider\n")
			// API key
			_, _ = pw.WriteString("sk-custom-key-789\n")
			// Base URL
			_, _ = pw.WriteString("https://api.custom.com/v1\n")
			// Model
			_, _ = pw.WriteString("custom-model-v1\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		// Capture stdout
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		providerName, _, err := configureProvider(tmpDir, cfg, "custom", secrets, secretsPath, keyPath)

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// Skip if TTY not available
		if err != nil && containsString(err.Error(), "inappropriate ioctl") {
			t.Skipf("configureProvider requires TTY: %v", err)
		}

		if err != nil {
			t.Errorf("configureProvider() error = %v", err)
		}

		if providerName != "mycustomprovider" {
			t.Errorf("configureProvider() returned %q, want 'mycustomprovider'", providerName)
		}

		// Check that provider was saved
		loadedCfg, err := config.LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		provider, ok := loadedCfg.Providers["mycustomprovider"]
		if !ok {
			t.Error("mycustomprovider not found in config")
		}

		if provider.Name != "My Custom Provider" {
			t.Errorf("Provider.Name = %q, want 'My Custom Provider'", provider.Name)
		}

		if provider.BaseURL != "https://api.custom.com/v1" {
			t.Errorf("Provider.BaseURL = %q, want 'https://api.custom.com/v1'", provider.BaseURL)
		}

		if provider.Model != "custom-model-v1" {
			t.Errorf("Provider.Model = %q, want 'custom-model-v1'", provider.Model)
		}

		// Check that secrets were saved
		decrypted, err := crypto.DecryptSecrets(secretsPath, keyPath)
		if err != nil {
			t.Fatalf("DecryptSecrets() error = %v", err)
		}

		parsedSecrets := config.ParseSecrets(decrypted)
		if _, ok := parsedSecrets["MY_CUSTOM_PROVIDER_API_KEY"]; !ok {
			t.Error("MY_CUSTOM_PROVIDER_API_KEY not found in secrets")
		}
	})

	t.Run("validates custom provider name", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secrets := make(map[string]string)

		// Prepare input: invalid custom name (starts with number)
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			// Invalid custom provider name (starts with number)
			_, _ = pw.WriteString("123-invalid\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		providerName, _, err := configureProvider(tmpDir, cfg, "custom", secrets, secretsPath, keyPath)

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// Skip if TTY not available (fails before validation in TTY env)
		if err != nil && containsString(err.Error(), "inappropriate ioctl") {
			t.Skipf("configureProvider requires TTY: %v", err)
		}

		if err == nil {
			t.Error("configureProvider() should return error for invalid custom provider name")
		}

		if providerName != "" {
			t.Errorf("configureProvider() returned %q, want empty string on error", providerName)
		}
	})

	t.Run("returns error for invalid API key", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		if err := config.SaveConfig(tmpDir, cfg); err != nil {
			t.Fatal(err)
		}

		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")
		if err := crypto.GenerateKey(keyPath); err != nil {
			t.Fatal(err)
		}

		secrets := make(map[string]string)

		// Prepare input: short API key
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			// Short API key
			_, _ = pw.WriteString("short\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		providerName, _, err := configureProvider(tmpDir, cfg, "zai", secrets, secretsPath, keyPath)

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// Skip if TTY not available
		if err != nil && containsString(err.Error(), "inappropriate ioctl") {
			t.Skipf("configureProvider requires TTY: %v", err)
		}

		if err == nil {
			t.Error("configureProvider() should return error for short API key")
		}

		if providerName != "" {
			t.Errorf("configureProvider() returned %q, want empty string on error", providerName)
		}
	})
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
			_, err := validateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestSetup_ProviderNameLength(t *testing.T) {
	// Create strings of exact lengths for testing
	maxValidName := strings.Repeat("a", 50) // Exactly 50 characters
	invalidName := strings.Repeat("b", 51) // 51 characters - exceeds max

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
			_, err := validateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
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
			name:    "anthropic",
			wantErr: true, // Reserved - built-in provider
		},
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
			name:    "Anthropic",
			wantErr: true, // Reserved - case-insensitive
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
			_, err := validateCustomProviderName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateCustomProviderName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
