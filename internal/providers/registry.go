// Package providers defines the built-in provider registry with names, base URLs,
// default models, environment variables, API key requirements, and key format rules.
package providers

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/fsutil"
)

//go:embed catalog.json
var embeddedCatalog []byte

const (
	MinAPIKeyLength     = 32
	DefaultMinKeyLength = 20
)

// KeyFormat holds minimum length, prefix, and pattern rules for API key validation.
type KeyFormat struct {
	MinLength int    `json:"min_length"`
	Prefix    string `json:"prefix"`
	Pattern   string `json:"pattern"`
}

// compiledCache caches compiled regexps by pattern to avoid races on
// the lazy-init path. Package-level so it's shared across all KeyFormat values.
var compiledCache sync.Map

func (kf *KeyFormat) validateForKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	if kf.MinLength > 0 && len(key) < kf.MinLength {
		return fmt.Errorf("API key too short (minimum %d characters, got %d)", kf.MinLength, len(key))
	}
	if kf.Prefix != "" && !strings.HasPrefix(key, kf.Prefix) {
		return fmt.Errorf("API key must start with '%s'", kf.Prefix)
	}
	if kf.Pattern != "" {
		v, ok := compiledCache.Load(kf.Pattern)
		if !ok {
			compiled, err := regexp.Compile(kf.Pattern)
			if err != nil {
				return fmt.Errorf("invalid key pattern for provider: %w", err)
			}
			compiledCache.Store(kf.Pattern, compiled)
			v = compiled
		}
		compiled, _ := v.(*regexp.Regexp)
		if !compiled.MatchString(key) {
			return fmt.Errorf("API key format is invalid")
		}
	}

	return nil
}

var (
	KeyFormatMin32      = KeyFormat{MinLength: MinAPIKeyLength}
	KeyFormatAnthropic  = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-ant-"}
	KeyFormatOpenAI     = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-"}
	KeyFormatGroq       = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "gsk_"}
	KeyFormatOpenRouter = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-or-"}
	DefaultKeyFormat    = KeyFormat{MinLength: DefaultMinKeyLength}
)

// catalogProvider is the JSON-deserializable form of a provider definition.
type catalogProvider struct {
	Name           string    `json:"name"`
	BaseURL        string    `json:"base_url"`
	Model          string    `json:"model"`
	EnvVars        []string  `json:"env_vars"`
	RequiresAPIKey bool      `json:"requires_api_key"`
	APIKeyEnvVar   string    `json:"api_key_env_var"`
	KeyFormat      KeyFormat `json:"key_format"`
}

// loadEmbeddedCatalog parses the embedded catalog.json into a map of providers.
func loadEmbeddedCatalog() map[string]ProviderDefinition {
	var raw map[string]catalogProvider
	if err := json.Unmarshal(embeddedCatalog, &raw); err != nil {
		panic("providers: failed to parse embedded catalog.json: " + err.Error())
	}

	result := make(map[string]ProviderDefinition, len(raw))
	for k := range raw {
		result[k] = ProviderDefinition(raw[k])
	}

	return result
}

// builtInProviders maps provider short names to their definitions.
// Populated at init from the embedded catalog.json.
var builtInProviders = loadEmbeddedCatalog()

// ProviderDefinition describes a built-in provider's display name, default
// base URL, model, environment variables, API key requirements, and key format.
type ProviderDefinition struct {
	Name           string
	BaseURL        string
	Model          string
	EnvVars        []string
	RequiresAPIKey bool
	APIKeyEnvVar   string
	KeyFormat      KeyFormat
}

// ValidateAPIKey checks the given key against this provider's key format rules.
func (d ProviderDefinition) ValidateAPIKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: API key cannot be empty or whitespace", d.Name))
	}

	if err := d.KeyFormat.validateForKey(key); err != nil {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: %s", d.Name, err))
	}

	return nil
}

// providerPriority defines the preferred display order for providers.
// Providers not listed here appear after these, in alphabetical order.
var providerPriority = []string{
	"zai", "minimax", "deepseek", "kimi",
	"anthropic", "openai", "google", "mistral",
	"groq", "cerebras", "cloudflare-workers-ai", "xai",
	"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
	"fireworks", "azure-openai-responses", "minimax-cn",
	"custom",
}

// providerOrder is the computed display order for all built-in providers.
// Priority providers appear first in the order defined above; remaining
// providers are sorted alphabetically. Computed once at init time.
var providerOrder []string

func init() {
	providerOrder = computeProviderOrder()
}

func computeProviderOrder() []string {
	seen := make(map[string]bool, len(builtInProviders))
	result := make([]string, 0, len(builtInProviders))

	for _, name := range providerPriority {
		if _, ok := builtInProviders[name]; ok {
			seen[name] = true
			result = append(result, name)
		}
	}

	remaining := make([]string, 0, len(builtInProviders)-len(seen))
	for name := range builtInProviders {
		if !seen[name] {
			remaining = append(remaining, name)
		}
	}
	slices.Sort(remaining)

	return append(result, remaining...)
}

// ProviderRegistry holds built-in and custom provider definitions.
// Package-level functions delegate to DefaultRegistry.
type ProviderRegistry struct {
	mu      sync.RWMutex
	builtIn map[string]ProviderDefinition
	cached  map[string]ProviderDefinition
	custom  map[string]ProviderDefinition
}

// NewRegistry creates a ProviderRegistry initialized with built-in providers.
func NewRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		builtIn: make(map[string]ProviderDefinition, len(builtInProviders)),
		cached:  make(map[string]ProviderDefinition),
		custom:  make(map[string]ProviderDefinition),
	}
	for k := range builtInProviders {
		r.builtIn[k] = builtInProviders[k]
	}

	return r
}

// RegisterCustom merges custom definitions into the registry.
// Custom entries override built-in entries with the same name.
func (r *ProviderRegistry) RegisterCustom(defs map[string]CustomProviderDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k := range defs {
		r.custom[k] = defs[k].ToProviderDefinition()
	}
}

// ClearCustom removes all custom definitions from the registry.
func (r *ProviderRegistry) ClearCustom() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.custom = make(map[string]ProviderDefinition)
}

// IsBuiltInProvider reports whether name is a recognized provider.
func (r *ProviderRegistry) IsBuiltInProvider(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.builtIn[name]; ok {
		return true
	}
	if _, ok := r.cached[name]; ok {
		return true
	}
	_, ok := r.custom[name]

	return ok
}

// BuiltInProvider returns the definition for the named provider.
func (r *ProviderRegistry) BuiltInProvider(name string) (ProviderDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.custom[name]; ok {
		return def, true
	}
	if def, ok := r.cached[name]; ok {
		return def, true
	}
	def, ok := r.builtIn[name]

	return def, ok
}

// ProviderList returns all provider names, built-in + cached + custom.
func (r *ProviderRegistry) ProviderList() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]string, 0, len(r.builtIn)+len(r.cached)+len(r.custom))

	for _, name := range providerOrder {
		if _, ok := r.builtIn[name]; ok {
			seen[name] = true
			result = append(result, name)
		}
	}

	// Append cached providers not already seen (not in embedded).
	for name := range r.cached {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	for name := range r.custom {
		if !seen[name] {
			result = append(result, name)
		}
	}

	return result
}

// RequiresAPIKey reports whether the named provider requires an API key.
func (r *ProviderRegistry) RequiresAPIKey(name string) bool {
	def, ok := r.BuiltInProvider(name)
	if !ok {
		return true
	}

	return def.RequiresAPIKey
}

// APIKeyEnvVarFor returns the environment variable name for the named
// provider's API key, if one is defined.
func (r *ProviderRegistry) APIKeyEnvVarFor(name string) (string, bool) {
	def, ok := r.BuiltInProvider(name)
	if !ok {
		return "", false
	}
	if def.APIKeyEnvVar == "" {
		return "", false
	}

	return def.APIKeyEnvVar, true
}

// ProviderSource returns the source layer for the named provider:
// "custom", "cached", "embedded", or "" if not found.
func (r *ProviderRegistry) ProviderSource(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.custom[name]; ok {
		return "custom"
	}
	if _, ok := r.cached[name]; ok {
		return "cached"
	}
	if _, ok := r.builtIn[name]; ok {
		return "embedded"
	}

	return ""
}

// LoadCache loads a provider catalog from the given JSON file path into
// the cached layer. Providers in the file override embedded definitions
// with the same name. If the file does not exist, LoadCache is a no-op.
func (r *ProviderRegistry) LoadCache(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	var raw map[string]catalogProvider
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cached = make(map[string]ProviderDefinition, len(raw))
	for k := range raw {
		r.cached[k] = ProviderDefinition(raw[k])
	}

	return nil
}

// RefreshCacheFromBytes replaces the cached layer with providers parsed from
// data (JSON) and atomically writes the cache to path. Returns the number of
// providers loaded.
func (r *ProviderRegistry) RefreshCacheFromBytes(data []byte, path string) (int, error) {
	var raw map[string]catalogProvider
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, err
	}

	cached := make(map[string]ProviderDefinition, len(raw))
	for k := range raw {
		cached[k] = ProviderDefinition(raw[k])
	}

	// Write to disk atomically.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	if err := fsutil.WriteAtomic(path, func(f *os.File) error {
		_, err := f.Write(data)

		return err
	}); err != nil {
		return 0, err
	}

	r.mu.Lock()
	r.cached = cached
	r.mu.Unlock()

	return len(cached), nil
}

// DefaultRegistry is the package-level singleton initialized with built-in providers.
var DefaultRegistry = NewRegistry()

// IsBuiltInProvider reports whether name is a recognized built-in provider.
func IsBuiltInProvider(name string) bool {
	return DefaultRegistry.IsBuiltInProvider(name)
}

// BuiltInProvider returns the definition for the named built-in provider.
func BuiltInProvider(name string) (ProviderDefinition, bool) {
	return DefaultRegistry.BuiltInProvider(name)
}

// ProviderList returns the ordered list of provider names.
func ProviderList() []string {
	return DefaultRegistry.ProviderList()
}

// RequiresAPIKey reports whether the named provider requires an API key.
func RequiresAPIKey(name string) bool {
	return DefaultRegistry.RequiresAPIKey(name)
}

// APIKeyEnvVarFor returns the environment variable name for the named
// provider's API key, if one is defined.
func APIKeyEnvVarFor(name string) (string, bool) {
	return DefaultRegistry.APIKeyEnvVarFor(name)
}
