// Package validate provides input validation for API keys and provider configuration.
package validate

import (
	"github.com/dkmnx/kairo/internal/providers"
)

// ValidateAPIKey checks that the given key meets the format requirements for the provider.
func ValidateAPIKey(key, providerName string) error {
	def, ok := providers.BuiltInProvider(providerName)
	if !ok {
		def = providers.ProviderDefinition{Name: providerName, KeyFormat: providers.DefaultKeyFormat}
	}

	return def.ValidateAPIKey(key)
}
