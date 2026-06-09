package providers

import (
	"fmt"
	"strings"
)

// ProviderTableMarkdown returns a markdown table of all built-in providers
// in their defined order, suitable for embedding in the project README.
func ProviderTableMarkdown() string {
	var b strings.Builder

	b.WriteString("| Provider | Identifier | Base URL | Default Model | API Key Env Var |\n")
	b.WriteString("|----------|------------|----------|---------------|-----------------|\n")

	for _, name := range ProviderOrder() {
		def, ok := DefaultRegistry.BuiltInProvider(name)
		if !ok {
			continue
		}

		baseURL := def.BaseURL
		if baseURL == "" {
			baseURL = "(default)"
		}

		model := def.Model
		if model == "" {
			model = "(default)"
		}

		fmt.Fprintf(&b, "| %s | `%s` | `%s` | `%s` | `%s` |\n",
			def.Name,
			name,
			baseURL,
			model,
			def.APIKeyEnvVar,
		)
	}

	return b.String()
}

// ProviderOrder returns provider names in display order (priority order first,
// then alphabetical for any remaining).
func ProviderOrder() []string {
	return providerOrder
}
