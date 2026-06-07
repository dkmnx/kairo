package config

import "testing"

func TestReconcileDefaultModels(t *testing.T) {
	t.Run("populates from providers with no DefaultModels entry", func(t *testing.T) {
		c := &Config{
			Providers:     map[string]Provider{"zai": {Model: "glm-5.1"}},
			DefaultModels: map[string]string{},
		}
		c.reconcileDefaultModels()
		if got := c.DefaultModels["zai"]; got != "glm-5.1" {
			t.Errorf("DefaultModels[zai] = %q, want %q", got, "glm-5.1")
		}
	})

	t.Run("preserves existing DefaultModels entry", func(t *testing.T) {
		c := &Config{
			Providers: map[string]Provider{
				"zai": {Model: "glm-5.1"},
			},
			DefaultModels: map[string]string{
				"zai": "user-override",
			},
		}
		c.reconcileDefaultModels()
		if got := c.DefaultModels["zai"]; got != "user-override" {
			t.Errorf("DefaultModels[zai] = %q, want %q (existing entry must be preserved)", got, "user-override")
		}
	})

	t.Run("empty config is a no-op", func(t *testing.T) {
		c := &Config{}
		c.reconcileDefaultModels()
		if len(c.DefaultModels) != 0 {
			t.Errorf("DefaultModels should remain empty, got %v", c.DefaultModels)
		}
	})
}
