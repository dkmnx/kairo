package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/yarlson/tap"
)

// waitForOutput polls the mock writable until a rendered frame contains want.
// tap registers the keypress listener before rendering a prompt, so observing
// the prompt's output guarantees subsequent emitted keypresses are queued and
// handled — unlike fixed sleeps, which race slow prompt setup (e.g. DNS).
func waitForOutput(t *testing.T, out *tap.MockWritable, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, f := range out.GetFrames() {
			if strings.Contains(f, want) {
				return
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for prompt output %q; frames: %v", want, out.GetFrames())
}

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

// advanceWithin reports whether the mock writable gained frames within d.
func advanceWithin(out *tap.MockWritable, before int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if len(out.GetFrames()) != before {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}

	return false
}

// pressReturn emits a return keypress and retries until the prompt output
// advances (the keypress was consumed) or the attempt budget is exhausted.
// A keypress emitted inside tap's async listener-registration window can be
// silently dropped; retrying is safe because an unchanged output means the
// same prompt is still active, and a delivered return always renders.
func pressReturn(in *tap.MockReadable, out *tap.MockWritable) {
	for attempt := 0; attempt < 5; attempt++ {
		before := len(out.GetFrames())
		in.EmitKeypress("", tap.Key{Name: "return"})
		if advanceWithin(out, before, 200*time.Millisecond) {
			return
		}
	}
}

// typeText types text into the prompt labeled want (waiting for it to
// render), confirming the first character is consumed (the drop window), then
// presses return with retry.
func typeText(t *testing.T, in *tap.MockReadable, out *tap.MockWritable, want, text string) {
	t.Helper()
	waitForOutput(t, out, want)

	for i, ch := range text {
		emit := func() {
			in.EmitKeypress(string(ch), tap.Key{Name: string(ch)})
		}
		if i == 0 {
			// First keypress after a prompt starts can be dropped; confirm
			// it renders before continuing.
			for attempt := 0; attempt < 5; attempt++ {
				before := len(out.GetFrames())
				emit()
				if advanceWithin(out, before, 200*time.Millisecond) {
					break
				}
			}
		} else {
			emit()
		}
	}

	pressReturn(in, out)
}

// pressEnter waits for the prompt labeled want and presses return with retry.
func pressEnter(t *testing.T, in *tap.MockReadable, out *tap.MockWritable, want string) {
	t.Helper()
	waitForOutput(t, out, want)
	pressReturn(in, out)
}

// answerConfirm waits for the confirm prompt labeled want and answers it with
// y or n. The character itself submits the confirm (tap submits on y/n), so
// no return keypress follows.
func answerConfirm(t *testing.T, in *tap.MockReadable, out *tap.MockWritable, want, answer string) {
	t.Helper()
	waitForOutput(t, out, want)
	in.EmitKeypress(answer, tap.Key{Name: answer})
	if !advanceWithin(out, len(out.GetFrames()), time.Second) {
		t.Fatalf("confirm %q did not submit on %q; frames: %v", want, answer, out.GetFrames())
	}
}

func TestPromptForNewProvider(t *testing.T) {
	in, out := setupTapTest(t)

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForNewProvider(context.Background())
	}()

	pressEnter(t, in, out, "Select provider to configure")

	result := <-resultCh

	firstProvider := providers.ProviderList()[0]
	if result != firstProvider {
		t.Errorf("promptForNewProvider() = %q, want %q", result, firstProvider)
	}
}

func TestPromptForProvider_NoExistingProviders(t *testing.T) {
	in, out := setupTapTest(t)

	cfg := &config.Config{
		DefaultProvider: "",
		Providers:       map[string]config.Provider{},
	}

	resultCh := make(chan string)
	go func() {
		resultCh <- promptForProvider(cfg)
	}()

	pressEnter(t, in, out, "Select provider to configure")

	result := <-resultCh

	firstProvider := providers.ProviderList()[0]
	if result != firstProvider {
		t.Errorf("promptForProvider() = %q, want %q", result, firstProvider)
	}
}

func TestPromptForProvider_WithExistingProviders(t *testing.T) {
	in, out := setupTapTest(t)

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

	pressEnter(t, in, out, "Select provider to edit or setup new")

	result := <-resultCh

	if result != "zai" {
		t.Errorf("promptForProvider() = %q, want %q", result, "zai")
	}
}

func TestPromptForExistingOrNewProvider_SelectExisting(t *testing.T) {
	in, out := setupTapTest(t)

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

	pressEnter(t, in, out, "Select provider to edit or setup new")

	result := <-resultCh

	if result != "zai" {
		t.Errorf("promptForExistingOrNewProvider() = %q, want %q", result, "zai")
	}
}

func TestPromptForAPIKey_NewInput(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "API Key", "sk-zai")

	result := <-resultCh

	if result != "sk-zai" {
		t.Errorf("promptForAPIKey() = %q, want %q", result, "sk-zai")
	}
}

func TestPromptForAPIKey_EditKeep(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify API key?", "n")

	result := <-resultCh

	if result != "existing-key" {
		t.Errorf("promptForAPIKey(edit keep) = %q, want %q", result, "existing-key")
	}
}

func TestPromptForAPIKey_EditChange(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify API key?", "y")
	typeText(t, in, out, "New API Key", "new-key")

	result := <-resultCh

	if result != "new-key" {
		t.Errorf("promptForAPIKey(edit change) = %q, want %q", result, "new-key")
	}
}

func TestPromptForField_NewInput(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "Base URL", "https://custom.url")

	result := <-resultCh

	if result != "https://custom.url" {
		t.Errorf("promptForField() = %q, want %q", result, "https://custom.url")
	}
}

func TestPromptForField_NewInputBlank(t *testing.T) {
	in, out := setupTapTest(t)

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

	pressEnter(t, in, out, "Base URL")

	result := <-resultCh

	if result != "https://default.url" {
		t.Errorf("promptForField(blank) = %q, want %q", result, "https://default.url")
	}
}

func TestPromptForFieldEdit_Keep(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify Model?", "n")

	result := <-resultCh

	if result != "glm-5" {
		t.Errorf("promptForFieldEdit(keep) = %q, want %q", result, "glm-5")
	}
}

func TestPromptForFieldEdit_Change(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify Model?", "y")
	typeText(t, in, out, "New Model", "new-model")

	result := <-resultCh

	if result != "new-model" {
		t.Errorf("promptForFieldEdit(change) = %q, want %q", result, "new-model")
	}
}

func TestPromptForFieldEdit_NoCurrent(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "Model", "new")

	result := <-resultCh

	if result != "new" {
		t.Errorf("promptForFieldEdit(no current) = %q, want %q", result, "new")
	}
}

func TestPromptForAPIKey_EditNoExistingKey(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "API Key", "sk-new")

	result := <-resultCh

	if result != "sk-new" {
		t.Errorf("promptForAPIKey(edit no key) = %q, want %q", result, "sk-new")
	}
}

func TestPromptForAPIKey_CustomProviderFallback(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify API key?", "n")

	result := <-resultCh

	if result != "custom-key" {
		t.Errorf("promptForAPIKey(custom fallback) = %q, want %q", result, "custom-key")
	}
}

func TestPromptForFieldEdit_KeepByEnter(t *testing.T) {
	in, out := setupTapTest(t)

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

	pressEnter(t, in, out, "Modify Base URL?")

	result := <-resultCh

	if result != "https://current.url" {
		t.Errorf("promptForFieldEdit(enter keep) = %q, want %q", result, "https://current.url")
	}
}

func TestPromptForField_EditMaintainsExisting(t *testing.T) {
	in, out := setupTapTest(t)

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

	answerConfirm(t, in, out, "Modify Base URL?", "n")

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
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "Base URL", "https://custom.api.com")

	result := <-resultCh

	if result != "https://custom.api.com" {
		t.Errorf("promptForBaseURL() = %q, want %q", result, "https://custom.api.com")
	}
}

func TestPromptForModel(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "Model", "custom-model")

	result := <-resultCh

	if result != "custom-model" {
		t.Errorf("promptForModel() = %q, want %q", result, "custom-model")
	}
}

func TestPromptForEnvKey(t *testing.T) {
	in, out := setupTapTest(t)

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

	typeText(t, in, out, "Env Key", "CUSTOM_API_KEY")

	result := <-resultCh

	if result != "CUSTOM_API_KEY" {
		t.Errorf("promptForEnvKey() = %q, want %q", result, "CUSTOM_API_KEY")
	}
}
