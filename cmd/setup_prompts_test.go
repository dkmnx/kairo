package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/yarlson/tap"
)

func mustProvider(t *testing.T, name string) providers.ProviderDefinition {
	t.Helper()
	def, ok := providers.BuiltInProvider(name)
	if !ok {
		t.Fatalf("provider %q not found", name)
	}

	return def
}

func setupTapTest(t *testing.T) (*tap.MockReadable, *tap.MockWritable) {
	t.Helper()
	in := tap.NewMockReadable()
	out := tap.NewMockWritable()
	tap.SetTermIO(in, out)
	t.Cleanup(func() { tap.SetTermIO(nil, nil) })
	return in, out
}

func emitReturn(in *tap.MockReadable) {
	in.EmitKeypress("", tap.Key{Name: "return"})
}

func emitText(in *tap.MockReadable, text string) {
	for _, ch := range text {
		in.EmitKeypress(string(ch), tap.Key{Name: string(ch)})
	}
}

func TestPromptForNewProvider(t *testing.T) {
	in, _ := setupTapTest(t)

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForNewProvider(context.Background())
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	firstProvider := providers.ProviderList()[0]
	if result != firstProvider {
		t.Errorf("promptForNewProvider() = %q, want %q", result, firstProvider)
	}
}

func TestPromptForProvider_NoExistingProviders(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForProvider(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	firstProvider := providers.ProviderList()[0]
	if result != firstProvider {
		t.Errorf("promptForProvider() = %q, want %q", result, firstProvider)
	}
}

func TestPromptForProvider_WithExistingProviders(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"},
		},
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForProvider(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	if result != "zai" {
		t.Errorf("promptForProvider() = %q, want %q", result, "zai")
	}
}

func TestPromptForExistingOrNewProvider_SelectExisting(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := &config.Config{
		DefaultProvider: "zai",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5"},
		},
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForExistingOrNewProvider(context.Background(), cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	if result != "zai" {
		t.Errorf("promptForExistingOrNewProvider() = %q, want %q", result, "zai")
	}
}

func TestPromptForAPIKey_NewInput(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForAPIKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "sk-zai")
	emitReturn(in)

	result := <-resultCh

	if result != "sk-zai" {
		t.Errorf("promptForAPIKey() = %q, want %q", result, "sk-zai")
	}
}

func TestPromptForAPIKey_EditKeep(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Secrets:      map[string]string{"ZAI_API_KEY": "existing-key"},
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForAPIKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	// Confirm "Modify API key?" -> n + return
	emitText(in, "n")
	emitReturn(in)

	result := <-resultCh

	if result != "existing-key" {
		t.Errorf("promptForAPIKey(edit keep) = %q, want %q", result, "existing-key")
	}
}

func TestPromptForAPIKey_EditChange(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Secrets:      map[string]string{"ZAI_API_KEY": "old-key"},
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForAPIKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	// Confirm "Modify API key?" -> y + return
	emitText(in, "y")
	emitReturn(in)

	// Wait for password prompt to register handler
	time.Sleep(50 * time.Millisecond)
	// New password
	emitText(in, "new-key")
	emitReturn(in)

	result := <-resultCh

	if result != "new-key" {
		t.Errorf("promptForAPIKey(edit change) = %q, want %q", result, "new-key")
	}
}

func TestPromptForField_NewInput(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Base URL",
		DefaultValue: "https://default.url",
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForField(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "https://custom.url")
	emitReturn(in)

	result := <-resultCh

	if result != "https://custom.url" {
		t.Errorf("promptForField() = %q, want %q", result, "https://custom.url")
	}
}

func TestPromptForField_NewInputBlank(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Base URL",
		DefaultValue: "https://default.url",
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForField(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	if result != "https://default.url" {
		t.Errorf("promptForField(blank) = %q, want %q", result, "https://default.url")
	}
}

func TestPromptForFieldEdit_Keep(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Model",
		CurrentValue: "glm-5",
		DefaultValue: "claude-sonnet",
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForFieldEdit(context.Background(), cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	// Confirm "Modify Model? (current: glm-5)" -> n + return
	emitText(in, "n")
	emitReturn(in)

	result := <-resultCh

	if result != "glm-5" {
		t.Errorf("promptForFieldEdit(keep) = %q, want %q", result, "glm-5")
	}
}

func TestPromptForFieldEdit_Change(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Model",
		CurrentValue: "glm-5",
		DefaultValue: "claude-sonnet",
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForFieldEdit(context.Background(), cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	// Confirm "Modify Model? (current: glm-5)" -> y + return
	emitText(in, "y")
	emitReturn(in)

	// Wait for text prompt to register handler
	time.Sleep(50 * time.Millisecond)
	// New model text
	emitText(in, "new-model")
	emitReturn(in)

	result := <-resultCh

	if result != "new-model" {
		t.Errorf("promptForFieldEdit(change) = %q, want %q", result, "new-model")
	}
}

func TestPromptForFieldEdit_NoCurrent(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Model",
		CurrentValue: "",
		DefaultValue: "",
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForFieldEdit(context.Background(), cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "new")
	emitReturn(in)

	result := <-resultCh

	if result != "new" {
		t.Errorf("promptForFieldEdit(no current) = %q, want %q", result, "new")
	}
}

func TestPromptForAPIKey_EditNoExistingKey(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Secrets:      map[string]string{},
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForAPIKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "sk-new")
	emitReturn(in)

	result := <-resultCh

	if result != "sk-new" {
		t.Errorf("promptForAPIKey(edit no key) = %q, want %q", result, "sk-new")
	}
}

func TestPromptForAPIKey_CustomProviderFallback(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "myprovider",
		Provider:     config.Provider{Name: "My Provider"},
		Secrets:      map[string]string{"CUSTOM_API_KEY": "custom-key"},
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForAPIKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "n")
	emitReturn(in)

	result := <-resultCh

	if result != "custom-key" {
		t.Errorf("promptForAPIKey(custom fallback) = %q, want %q", result, "custom-key")
	}
}

func TestPromptForFieldEdit_KeepByEnter(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: "https://current.url",
		DefaultValue: "https://default.url",
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForFieldEdit(context.Background(), cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitReturn(in)

	result := <-resultCh

	if result != "https://current.url" {
		t.Errorf("promptForFieldEdit(enter keep) = %q, want %q", result, "https://current.url")
	}
}

func TestPromptForField_EditMaintainsExisting(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := promptFieldConfig{
		Label:        "Base URL",
		CurrentValue: "https://existing.url",
		DefaultValue: "https://default.url",
		IsEdit:       true,
		Exists:       true,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForField(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "n")
	emitReturn(in)

	result := <-resultCh

	if result != "https://existing.url" {
		t.Errorf("promptForField(edit maintain) = %q, want %q", result, "https://existing.url")
	}
}

func TestDisplayProviderHeader_EditExisting(t *testing.T) {
	// displayProviderHeader doesn't use SetTermIO - it just calls tap.Message
	// Test that it doesn't panic
	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Definition:   mustProvider(t, "zai"),
		IsEdit:       true,
		Exists:       true,
	}
	displayProviderHeader(cfg)
}

func TestDisplayProviderHeader_NewOnly(t *testing.T) {
	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Definition:   mustProvider(t, "zai"),
		IsEdit:       false,
		Exists:       false,
	}
	displayProviderHeader(cfg)
}

func TestPromptForBaseURL(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Definition:   mustProvider(t, "zai"),
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForBaseURL(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "https://custom.api.com")
	emitReturn(in)

	result := <-resultCh

	if result != "https://custom.api.com" {
		t.Errorf("promptForBaseURL() = %q, want %q", result, "https://custom.api.com")
	}
}

func TestPromptForModel(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Definition:   mustProvider(t, "zai"),
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForModel(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "custom-model")
	emitReturn(in)

	result := <-resultCh

	if result != "custom-model" {
		t.Errorf("promptForModel() = %q, want %q", result, "custom-model")
	}
}

func TestPromptForEnvKey(t *testing.T) {
	in, _ := setupTapTest(t)

	cfg := providerPromptConfig{
		ProviderName: "zai",
		Provider:     config.Provider{Name: "Z.AI"},
		Definition:   mustProvider(t, "zai"),
		IsEdit:       false,
		Exists:       false,
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForEnvKey(cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	emitText(in, "CUSTOM_API_KEY")
	emitReturn(in)

	result := <-resultCh

	if result != "CUSTOM_API_KEY" {
		t.Errorf("promptForEnvKey() = %q, want %q", result, "CUSTOM_API_KEY")
	}
}
