package cmd

import (
	"context"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
)

type fakePromptUI struct {
	selectFn      func(ctx context.Context, msg string, opts []SelectOption) string
	textFn        func(ctx context.Context, opts TextOptions) string
	passwordFn    func(ctx context.Context, opts PasswordOptions) string
	confirmFn     func(ctx context.Context, opts ConfirmOptions) bool
	introFn       func(title string, opts MessageOptions)
	messageFn     func(msg string, opts MessageOptions)
	selectCalls   []selectCall
	textCalls     []TextOptions
	passwordCalls []PasswordOptions
	confirmCalls  []ConfirmOptions
}

type selectCall struct {
	Message  string
	Options  []SelectOption
	Response string
}

func (f *fakePromptUI) Select(ctx context.Context, msg string, opts []SelectOption) string {
	f.selectCalls = append(f.selectCalls, selectCall{msg, opts, ""})
	if f.selectFn != nil {
		return f.selectFn(ctx, msg, opts)
	}
	return ""
}

func (f *fakePromptUI) Text(ctx context.Context, opts TextOptions) string {
	f.textCalls = append(f.textCalls, opts)
	if f.textFn != nil {
		return f.textFn(ctx, opts)
	}
	return ""
}

func (f *fakePromptUI) Password(ctx context.Context, opts PasswordOptions) string {
	f.passwordCalls = append(f.passwordCalls, opts)
	if f.passwordFn != nil {
		return f.passwordFn(ctx, opts)
	}
	return ""
}

func (f *fakePromptUI) Confirm(ctx context.Context, opts ConfirmOptions) bool {
	f.confirmCalls = append(f.confirmCalls, opts)
	if f.confirmFn != nil {
		return f.confirmFn(ctx, opts)
	}
	return false
}

func (f *fakePromptUI) Intro(title string, opts MessageOptions) {
	if f.introFn != nil {
		f.introFn(title, opts)
	}
}

func (f *fakePromptUI) Message(msg string, opts MessageOptions) {
	if f.messageFn != nil {
		f.messageFn(msg, opts)
	}
}

func TestSelectProvider_NoExisting(t *testing.T) {
	ui := &fakePromptUI{
		selectFn: func(ctx context.Context, msg string, opts []SelectOption) string {
			return "anthropic"
		},
	}

	result := selectProviderNoExisting(ui)
	if result != "anthropic" {
		t.Errorf("selectProviderNoExisting() = %q, want %q", result, "anthropic")
	}
}

func TestSelectProvider_WithExisting(t *testing.T) {
	ui := &fakePromptUI{
		selectFn: func(ctx context.Context, msg string, opts []SelectOption) string {
			return "existing"
		},
	}

	providerNames := []string{"existing", "Setup new provider"}
	result := selectProviderWithExisting(&config.Config{}, providerNames, ui)

	if result != "existing" {
		t.Errorf("selectProviderWithExisting() = %q, want %q", result, "existing")
	}
}

func TestSelectProvider_WithExisting_SelectsNew(t *testing.T) {
	ui := &fakePromptUI{
		selectFn: func(ctx context.Context, msg string, opts []SelectOption) string {
			return "Setup new provider"
		},
	}

	providerNames := []string{"existing", "Setup new provider"}
	result := selectProviderWithExisting(&config.Config{}, providerNames, ui)

	if result != "Setup new provider" {
		t.Errorf("selectProviderWithExisting() = %q, want %q", result, "Setup new provider")
	}
}

func TestPromptAPIKey_NewProvider(t *testing.T) {
	ui := &fakePromptUI{
		passwordFn: func(ctx context.Context, opts PasswordOptions) string {
			return "secret-key-123"
		},
	}

	result := promptAPIKey("anthropic", nil, false, false, ui)
	if result != "secret-key-123" {
		t.Errorf("promptAPIKey() = %q, want %q", result, "secret-key-123")
	}
}

func TestPromptAPIKey_Edit_DeclineModify(t *testing.T) {
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": "existing-key",
	}

	ui := &fakePromptUI{
		confirmFn: func(ctx context.Context, opts ConfirmOptions) bool {
			return false
		},
	}

	result := promptAPIKey("anthropic", secrets, true, true, ui)
	if result != "existing-key" {
		t.Errorf("promptAPIKey() = %q, want %q (should return existing)", result, "existing-key")
	}
}

func TestPromptAPIKey_Edit_AcceptModify(t *testing.T) {
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": "existing-key",
	}

	passwordCall := ""
	ui := &fakePromptUI{
		confirmFn: func(ctx context.Context, opts ConfirmOptions) bool {
			return true
		},
		passwordFn: func(ctx context.Context, opts PasswordOptions) string {
			passwordCall = opts.Message
			return "new-key-456"
		},
	}

	result := promptAPIKey("anthropic", secrets, true, true, ui)
	if result != "new-key-456" {
		t.Errorf("promptAPIKey() = %q, want %q", result, "new-key-456")
	}
	if passwordCall != "New API Key" {
		t.Errorf("password prompt message = %q, want %q", passwordCall, "New API Key")
	}
}

func TestPromptAPIKey_Edit_NoExistingKey(t *testing.T) {
	secrets := map[string]string{}

	ui := &fakePromptUI{
		passwordFn: func(ctx context.Context, opts PasswordOptions) string {
			return "fresh-key"
		},
	}

	result := promptAPIKey("anthropic", secrets, true, true, ui)
	if result != "fresh-key" {
		t.Errorf("promptAPIKey() = %q, want %q", result, "fresh-key")
	}
}

func TestPromptField_KeepCurrent(t *testing.T) {
	ui := &fakePromptUI{
		confirmFn: func(ctx context.Context, opts ConfirmOptions) bool {
			return false
		},
	}

	cfg := testPromptFieldConfig{
		Label:        "Model",
		CurrentValue: "claude-3",
		DefaultValue: "claude-3",
		IsEdit:       true,
		Exists:       true,
	}

	result := promptFieldKeepCurrent(cfg, ui)
	if result != "claude-3" {
		t.Errorf("promptFieldKeepCurrent() = %q, want %q", result, "claude-3")
	}
}

func TestPromptField_ModifyValue(t *testing.T) {
	ui := &fakePromptUI{
		confirmFn: func(ctx context.Context, opts ConfirmOptions) bool {
			return true
		},
		textFn: func(ctx context.Context, opts TextOptions) string {
			return "claude-4"
		},
	}

	cfg := testPromptFieldConfig{
		Label:        "Model",
		CurrentValue: "claude-3",
		DefaultValue: "claude-3",
		IsEdit:       true,
		Exists:       true,
	}

	result := promptField(cfg, ui)
	if result != "claude-4" {
		t.Errorf("promptField() = %q, want %q", result, "claude-4")
	}
}

func TestPromptField_NewProvider(t *testing.T) {
	ui := &fakePromptUI{
		textFn: func(ctx context.Context, opts TextOptions) string {
			return "claude-3-sonnet"
		},
	}

	cfg := testPromptFieldConfig{
		Label:        "Model",
		CurrentValue: "",
		DefaultValue: "claude-3-sonnet",
		IsEdit:       false,
		Exists:       false,
	}

	result := promptFieldNew(cfg, ui)
	if result != "claude-3-sonnet" {
		t.Errorf("promptFieldNew() = %q, want %q", result, "claude-3-sonnet")
	}
}

func TestPromptField_NewProvider_EmptyReturnsDefault(t *testing.T) {
	ui := &fakePromptUI{
		textFn: func(ctx context.Context, opts TextOptions) string {
			return ""
		},
	}

	cfg := testPromptFieldConfig{
		Label:        "Model",
		CurrentValue: "",
		DefaultValue: "default-model",
		IsEdit:       false,
		Exists:       false,
	}

	result := promptFieldNew(cfg, ui)
	if result != "default-model" {
		t.Errorf("promptFieldNew() = %q, want %q (fallback to default)", result, "default-model")
	}
}

func TestPromptField_NewProvider_TrimsWhitespace(t *testing.T) {
	ui := &fakePromptUI{
		textFn: func(ctx context.Context, opts TextOptions) string {
			return "  claude-4  "
		},
	}

	cfg := testPromptFieldConfig{
		Label:        "Model",
		CurrentValue: "",
		DefaultValue: "default",
		IsEdit:       false,
		Exists:       false,
	}

	result := promptFieldNew(cfg, ui)
	if result != "claude-4" {
		t.Errorf("promptFieldNew() = %q, want %q (trimmed)", result, "claude-4")
	}
}
