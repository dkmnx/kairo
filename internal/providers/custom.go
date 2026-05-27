package providers

// CustomProviderDefinition is the YAML-deserializable form of a provider
// definition. Users define these under custom_providers in config.yaml.
type CustomProviderDefinition struct {
	Name           string   `yaml:"name"`
	BaseURL        string   `yaml:"base_url"`
	Model          string   `yaml:"model"`
	EnvVars        []string `yaml:"env_vars"`
	RequiresAPIKey bool     `yaml:"requires_api_key"`
	APIKeyEnvVar   string   `yaml:"api_key_env_var"`
	MinKeyLength   int      `yaml:"min_key_length"`
	KeyPrefix      string   `yaml:"key_prefix"`
	KeyPattern     string   `yaml:"key_pattern"`
}

// ToProviderDefinition converts the YAML form into the internal ProviderDefinition.
func (c CustomProviderDefinition) ToProviderDefinition() ProviderDefinition {
	kf := KeyFormat{
		MinLength: c.MinKeyLength,
		Prefix:    c.KeyPrefix,
		Pattern:   c.KeyPattern,
	}
	if kf.MinLength == 0 {
		kf.MinLength = DefaultMinKeyLength
	}

	return ProviderDefinition{
		Name:           c.Name,
		BaseURL:        c.BaseURL,
		Model:          c.Model,
		EnvVars:        c.EnvVars,
		RequiresAPIKey: c.RequiresAPIKey,
		APIKeyEnvVar:   c.APIKeyEnvVar,
		KeyFormat:      kf,
	}
}
