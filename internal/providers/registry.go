// Package providers defines the built-in provider registry with names, base URLs,
// default models, environment variables, API key requirements, and key format rules.
package providers

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/dkmnx/kairo/internal/errors"
)

const (
	MinAPIKeyLength     = 32
	DefaultMinKeyLength = 20
)

// KeyFormat holds minimum length, prefix, and pattern rules for API key validation.
type KeyFormat struct {
	MinLength int
	Prefix    string
	Pattern   string
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

// builtInProviders maps provider short names to their definitions.
var builtInProviders = map[string]ProviderDefinition{
	"zai": {
		Name:           "Z.AI",
		BaseURL:        "https://api.z.ai/api/anthropic",
		Model:          "glm-5.1",
		RequiresAPIKey: true,
		EnvVars:        []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
		APIKeyEnvVar:   "ZAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"minimax": {
		Name:           "MiniMax",
		BaseURL:        "https://api.minimax.io/anthropic",
		Model:          "MiniMax-M2.7",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=120",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=24576",
		},
		APIKeyEnvVar: "MINIMAX_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"kimi": {
		Name:           "Moonshot AI",
		BaseURL:        "https://api.kimi.com/coding/",
		Model:          "kimi-for-coding",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=240",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=200000",
		},
		APIKeyEnvVar: "KIMI_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"deepseek": {
		Name:           "DeepSeek AI",
		BaseURL:        "https://api.deepseek.com/anthropic",
		Model:          "deepseek-v4-pro[1m]",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_DEFAULT_HAIKU_MODEL=deepseek-v4-flash",
			"CLAUDE_CODE_SUBAGENT_MODEL=deepseek-v4-flash",
			"CLAUDE_CODE_EFFORT_LEVEL=max",
			"API_TIMEOUT_MS=600000",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		},
		APIKeyEnvVar: "DEEPSEEK_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"anthropic": {
		Name:           "Anthropic",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "ANTHROPIC_API_KEY",
		KeyFormat:      KeyFormatAnthropic,
	},
	"openai": {
		Name:           "OpenAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENAI_API_KEY",
		KeyFormat:      KeyFormatOpenAI,
	},
	"google": {
		Name:           "Google",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "GEMINI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"mistral": {
		Name:           "Mistral",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "MISTRAL_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"groq": {
		Name:           "Groq",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "GROQ_API_KEY",
		KeyFormat:      KeyFormatGroq,
	},
	"cerebras": {
		Name:           "Cerebras",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "CEREBRAS_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"cloudflare-workers-ai": {
		Name:           "Cloudflare Workers AI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "CLOUDFLARE_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"xai": {
		Name:           "xAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "XAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"openrouter": {
		Name:           "OpenRouter",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENROUTER_API_KEY",
		KeyFormat:      KeyFormatOpenRouter,
	},
	"vercel-ai-gateway": {
		Name:           "Vercel AI Gateway",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "AI_GATEWAY_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"opencode": {
		Name:           "OpenCode",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENCODE_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"huggingface": {
		Name:           "Hugging Face",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "HF_TOKEN",
		KeyFormat:      KeyFormatMin32,
	},
	"fireworks": {
		Name:           "Fireworks",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "FIREWORKS_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"azure-openai-responses": {
		Name:           "Azure OpenAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "AZURE_OPENAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"minimax-cn": {
		Name:           "MiniMax (CN)",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "MINIMAX_CN_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"custom": {
		Name:           "Custom Provider",
		BaseURL:        "",
		Model:          "",
		RequiresAPIKey: true,
		KeyFormat:      DefaultKeyFormat,
	},
}

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
	custom  map[string]ProviderDefinition
}

// NewRegistry creates a ProviderRegistry initialized with built-in providers.
func NewRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		builtIn: make(map[string]ProviderDefinition, len(builtInProviders)),
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

	_, ok := r.builtIn[name]
	if ok {
		return true
	}
	_, ok = r.custom[name]

	return ok
}

// BuiltInProvider returns the definition for the named provider.
func (r *ProviderRegistry) BuiltInProvider(name string) (ProviderDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if def, ok := r.custom[name]; ok {
		return def, true
	}
	def, ok := r.builtIn[name]

	return def, ok
}

// ProviderList returns all provider names, built-in first then custom.
func (r *ProviderRegistry) ProviderList() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]string, 0, len(r.builtIn)+len(r.custom))

	for _, name := range providerOrder {
		if _, ok := r.builtIn[name]; ok {
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
