package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/yarlson/tap"
)

// --- Test helper restore utilities ---

type tapFuncs struct {
	selectFn   func(ctx context.Context, opts tap.SelectOptions[string]) string
	textFn     func(ctx context.Context, opts tap.TextOptions) string
	passwordFn func(ctx context.Context, opts tap.PasswordOptions) string
	confirmFn  func(ctx context.Context, opts tap.ConfirmOptions) bool
	introFn    func(title string, opts ...tap.MessageOptions)
	outroFn    func(message string, opts ...tap.MessageOptions)
	messageFn  func(message string, opts ...tap.MessageOptions)
}

// withMockedTAP saves all tap function variables, replaces them with mocks,
// and defers restoration. Returns the original state for inspection.
func withMockedTAP(t *testing.T) *tapFuncs {
	t.Helper()
	origins := &tapFuncs{
		selectFn:   tapSelectFn,
		textFn:     tapTextFn,
		passwordFn: tapPasswordFn,
		confirmFn:  tapConfirmFn,
		introFn:    tapIntroFn,
		outroFn:    tapOutroFn,
		messageFn:  tapMessageFn,
	}

	t.Cleanup(func() {
		tapSelectFn = origins.selectFn
		tapTextFn = origins.textFn
		tapPasswordFn = origins.passwordFn
		tapConfirmFn = origins.confirmFn
		tapIntroFn = origins.introFn
		tapOutroFn = origins.outroFn
		tapMessageFn = origins.messageFn
	})

	return origins
}

// --- Tests for buildProviderListOptions ---

func TestBuildProviderListOptions(t *testing.T) {
	tests := []struct {
		name       string
		input      []string
		wantLen    int
		wantValues []string
	}{
		{name: "empty list", input: []string{}, wantLen: 0, wantValues: []string{}},
		{name: "single provider", input: []string{"zai"}, wantLen: 1, wantValues: []string{"zai"}},
		{name: "multiple providers", input: []string{"zai", "minimax", "deepseek"}, wantLen: 3, wantValues: []string{"zai", "minimax", "deepseek"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildProviderListOptions(tt.input)
			if len(options) != tt.wantLen {
				t.Errorf("buildProviderListOptions() returned %d options, want %d", len(options), tt.wantLen)
			}
			for i, opt := range options {
				if i < len(tt.wantValues) {
					if opt.Value != tt.wantValues[i] {
						t.Errorf("options[%d].Value = %q, want %q", i, opt.Value, tt.wantValues[i])
					}
					if opt.Label != tt.wantValues[i] {
						t.Errorf("options[%d].Label = %q, want %q", i, opt.Label, tt.wantValues[i])
					}
				}
			}
		})
	}
}

// --- Tests for displayProviderHeader ---

func TestDisplayProviderHeader_EditExisting(t *testing.T) {
	messages := []string{}
	withMockedTAP(t)
	tapMessageFn = func(message string, opts ...tap.MessageOptions) {
		messages = append(messages, message)
	}

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		IsEdit:       true, Exists: true,
	}
	displayProviderHeader(cfg)

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "Editing Z.AI") {
		t.Errorf("Expected message containing 'Editing Z.AI', got %q", messages[0])
	}
}

func TestDisplayProviderHeader_NewProvider(t *testing.T) {
	called := false
	withMockedTAP(t)
	tapMessageFn = func(message string, opts ...tap.MessageOptions) { called = true }

	cfg := providerPromptConfig{
		ProviderName: "zai", Provider: config.Provider{Name: "Z.AI"},
		IsEdit: false, Exists: false,
	}
	displayProviderHeader(cfg)
	if called {
		t.Error("displayProviderHeader should not call Message for new provider")
	}
}

func TestDisplayProviderHeader_EditNotExisting(t *testing.T) {
	called := false
	withMockedTAP(t)
	tapMessageFn = func(message string, opts ...tap.MessageOptions) { called = true }

	cfg := providerPromptConfig{
		ProviderName: "zai", Provider: config.Provider{Name: "Z.AI"},
		IsEdit: true, Exists: false,
	}
	displayProviderHeader(cfg)
	if called {
		t.Error("displayProviderHeader should not call Message when Exists=false")
	}
}

// --- Tests for promptForAPIKey ---

func TestPromptForAPIKey_NewProvider(t *testing.T) {
	withMockedTAP(t)
	apiKey := "test-api-key-12345678901234567890"
	tapPasswordFn = func(ctx context.Context, opts tap.PasswordOptions) string { return apiKey }

	cfg := providerPromptConfig{ProviderName: "zai", IsEdit: false, Exists: false}
	result := promptForAPIKey(cfg)
	if result != apiKey {
		t.Errorf("promptForAPIKey() = %q, want %q", result, apiKey)
	}
}

func TestPromptForAPIKey_EditKeepExisting(t *testing.T) {
	withMockedTAP(t)
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return false }

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Secrets:      map[string]string{"ZAI_API_KEY": "existing-key-12345678901234567890"},
		IsEdit:       true, Exists: true,
	}
	result := promptForAPIKey(cfg)
	if result != "existing-key-12345678901234567890" {
		t.Errorf("promptForAPIKey() = %q, want existing key", result)
	}
}

func TestPromptForAPIKey_EditModifyKey(t *testing.T) {
	withMockedTAP(t)
	newKey := "new-api-key-123456789012345678901"
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return true }
	tapPasswordFn = func(ctx context.Context, opts tap.PasswordOptions) string { return newKey }

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Secrets:      map[string]string{"ZAI_API_KEY": "existing-key-12345678901234567890"},
		IsEdit:       true, Exists: true,
	}
	result := promptForAPIKey(cfg)
	if result != newKey {
		t.Errorf("promptForAPIKey() = %q, want %q", result, newKey)
	}
}

func TestPromptForAPIKey_EditNoExistingKey(t *testing.T) {
	withMockedTAP(t)
	apiKey := "fresh-api-key-12345678901234567890"
	tapPasswordFn = func(ctx context.Context, opts tap.PasswordOptions) string { return apiKey }

	cfg := providerPromptConfig{
		ProviderName: "zai", Secrets: map[string]string{},
		IsEdit: true, Exists: true,
	}
	result := promptForAPIKey(cfg)
	if result != apiKey {
		t.Errorf("promptForAPIKey() = %q, want %q", result, apiKey)
	}
}

// --- Tests for promptForField ---

func TestPromptForField_NewProvider(t *testing.T) {
	withMockedTAP(t)
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "custom-base-url" }

	cfg := promptFieldConfig{Label: "Base URL", DefaultValue: "https://api.default.com", IsEdit: false}
	result := promptForField(cfg)
	if result != "custom-base-url" {
		t.Errorf("promptForField() = %q, want %q", result, "custom-base-url")
	}
}

func TestPromptForField_DefaultOnEmpty(t *testing.T) {
	withMockedTAP(t)
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "" }

	cfg := promptFieldConfig{Label: "Base URL", DefaultValue: "https://api.default.com", IsEdit: false}
	result := promptForField(cfg)
	if result != "https://api.default.com" {
		t.Errorf("promptForField() = %q, want default %q", result, "https://api.default.com")
	}
}

func TestPromptForField_EditKeep(t *testing.T) {
	withMockedTAP(t)
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return false }

	cfg := promptFieldConfig{
		Label: "Base URL", CurrentValue: "https://current.com",
		DefaultValue: "https://api.default.com", IsEdit: true, Exists: true,
	}
	result := promptForField(cfg)
	if result != "https://current.com" {
		t.Errorf("promptForField() = %q, want current value %q", result, "https://current.com")
	}
}

func TestPromptForField_EditModify(t *testing.T) {
	withMockedTAP(t)
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return true }
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "  https://modified.com  " }

	cfg := promptFieldConfig{
		Label: "Base URL", CurrentValue: "https://current.com",
		DefaultValue: "https://api.default.com", IsEdit: true, Exists: true,
	}
	result := promptForField(cfg)
	if result != "https://modified.com" {
		t.Errorf("promptForField() = %q, want %q", result, "https://modified.com")
	}
}

// --- Tests for promptForBaseURL and promptForModel ---

func TestPromptForBaseURL(t *testing.T) {
	withMockedTAP(t)
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "https://custom.api.com/anthropic" }

	cfg := providerPromptConfig{
		ProviderName: "custom",
		Definition:   providers.ProviderDefinition{Name: "Custom", BaseURL: "https://default.com"},
		IsEdit:       false,
	}
	result := promptForBaseURL(cfg)
	if result != "https://custom.api.com/anthropic" {
		t.Errorf("promptForBaseURL() = %q, want %q", result, "https://custom.api.com/anthropic")
	}
}

func TestPromptForModel(t *testing.T) {
	withMockedTAP(t)
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "custom-model-v2" }

	cfg := providerPromptConfig{
		ProviderName: "custom",
		Definition:   providers.ProviderDefinition{Name: "Custom", Model: "default-model"},
		IsEdit:       false,
	}
	result := promptForModel(cfg)
	if result != "custom-model-v2" {
		t.Errorf("promptForModel() = %q, want %q", result, "custom-model-v2")
	}
}

// --- Tests for promptForProvider ---

func TestPromptForProvider_NoProviders(t *testing.T) {
	withMockedTAP(t)
	tapSelectFn = func(ctx context.Context, opts tap.SelectOptions[string]) string { return "zai" }

	cfg := &config.Config{Providers: make(map[string]config.Provider)}
	result := promptForProvider(cfg)
	if result != "zai" {
		t.Errorf("promptForProvider() = %q, want %q", result, "zai")
	}
}

func TestPromptForProvider_SelectNewProvider(t *testing.T) {
	withMockedTAP(t)
	callCount := 0
	tapSelectFn = func(ctx context.Context, opts tap.SelectOptions[string]) string {
		callCount++
		if callCount == 1 {
			return setupNewProvider
		}
		return "deepseek"
	}
	tapIntroFn = func(title string, opts ...tap.MessageOptions) {}

	cfg := &config.Config{Providers: map[string]config.Provider{"zai": {Name: "Z.AI"}}}
	result := promptForProvider(cfg)
	if result != "deepseek" {
		t.Errorf("promptForProvider() = %q, want %q", result, "deepseek")
	}
}

func TestPromptForProvider_Cancel(t *testing.T) {
	withMockedTAP(t)
	tapSelectFn = func(ctx context.Context, opts tap.SelectOptions[string]) string { return "" }

	cfg := &config.Config{Providers: map[string]config.Provider{"zai": {Name: "Z.AI"}}}
	result := promptForProvider(cfg)
	if result != "" {
		t.Errorf("promptForProvider() should return empty string on cancel, got %q", result)
	}
}

// --- Tests for promptForNewProvider ---

func TestPromptForNewProvider(t *testing.T) {
	withMockedTAP(t)
	tapSelectFn = func(ctx context.Context, opts tap.SelectOptions[string]) string {
		providerNames := make([]string, len(opts.Options))
		for i, opt := range opts.Options {
			providerNames[i] = opt.Value
		}
		for _, name := range providers.GetProviderList() {
			found := false
			for _, pn := range providerNames {
				if pn == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected provider %q in options", name)
			}
		}
		foundCustom := false
		for _, pn := range providerNames {
			if pn == "custom" {
				foundCustom = true
				break
			}
		}
		if !foundCustom {
			t.Error("Expected 'custom' in options")
		}
		return "minimax"
	}

	result := promptForNewProvider(context.Background())
	if result != "minimax" {
		t.Errorf("promptForNewProvider() = %q, want %q", result, "minimax")
	}
}

// --- Tests for promptForFieldEdit ---

func TestPromptForFieldEdit_ConfirmModify(t *testing.T) {
	withMockedTAP(t)
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return true }
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "  https://modified.com  " }

	cfg := promptFieldConfig{
		Label: "Base URL", CurrentValue: "https://current.com",
		DefaultValue: "https://default.com", IsEdit: true, Exists: true,
	}
	result := promptForFieldEdit(context.Background(), cfg)
	if result != "https://modified.com" {
		t.Errorf("promptForFieldEdit() = %q, want %q", result, "https://modified.com")
	}
}

func TestPromptForFieldEdit_DeclineModify(t *testing.T) {
	withMockedTAP(t)
	tapConfirmFn = func(ctx context.Context, opts tap.ConfirmOptions) bool { return false }

	cfg := promptFieldConfig{
		Label: "Base URL", CurrentValue: "https://current.com",
		DefaultValue: "https://default.com", IsEdit: true, Exists: true,
	}
	result := promptForFieldEdit(context.Background(), cfg)
	if result != "https://current.com" {
		t.Errorf("promptForFieldEdit() = %q, want %q", result, "https://current.com")
	}
}

func TestPromptForFieldEdit_NoCurrentNoDefault(t *testing.T) {
	withMockedTAP(t)
	tapTextFn = func(ctx context.Context, opts tap.TextOptions) string { return "user-entered-value" }

	cfg := promptFieldConfig{
		Label: "Base URL", CurrentValue: "",
		DefaultValue: "", IsEdit: true, Exists: true,
	}
	result := promptForFieldEdit(context.Background(), cfg)
	if result != "user-entered-value" {
		t.Errorf("promptForFieldEdit() = %q, want %q", result, "user-entered-value")
	}
}
