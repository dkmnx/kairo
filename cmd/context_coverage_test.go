package cmd

import (
	"errors"
	"testing"
)

// TestConfigDir_ResolverError exercises the path where the user-supplied
// resolver returns an error. ConfigDir() should return "" without panicking.
func TestConfigDir_ResolverError(t *testing.T) {
	c := NewCLIContext()
	c.SetConfigDirResolver(func() (string, error) {
		return "", errors.New("simulated resolver failure")
	})

	if got := c.ConfigDir(); got != "" {
		t.Errorf("ConfigDir() = %q, want empty string on resolver error", got)
	}
}

// TestConfigDir_ResolverSuccess exercises the happy path of the injected
// resolver. The explicit set covers both branches of the resolver
// invocation.
func TestConfigDir_ResolverSuccess(t *testing.T) {
	c := NewCLIContext()
	c.SetConfigDirResolver(func() (string, error) {
		return "/tmp/from-resolver", nil
	})

	if got := c.ConfigDir(); got != "/tmp/from-resolver" {
		t.Errorf("ConfigDir() = %q, want %q", got, "/tmp/from-resolver")
	}
}
