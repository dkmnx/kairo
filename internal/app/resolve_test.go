package app

import (
	"errors"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		wantRest []string
	}{
		{"no separator", []string{"zai", "hello"}, []string{"zai", "hello"}, nil},
		{"with separator", []string{"zai", "--", "hello"}, []string{"zai"}, []string{"hello"}},
		{"separator first", []string{"--", "hello"}, []string{}, []string{"hello"}},
		{"empty", []string{}, []string{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotRest := SplitArgs(tt.args)
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("SplitArgs() args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i := range tt.wantArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("SplitArgs() args[%d] = %q, want %q", i, gotArgs[i], tt.wantArgs[i])
				}
			}
			if len(gotRest) != len(tt.wantRest) {
				t.Errorf("SplitArgs() rest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestHasLeadingArgsSeparator(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"leading separator", []string{"--", "hello"}, true},
		{"after flags", []string{"-v", "--harness", "pi", "--", "hello"}, true},
		{"after positional", []string{"anthropic", "--", "hello"}, false},
		{"no separator", []string{"anthropic", "hello"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasLeadingArgsSeparator(tt.args); got != tt.want {
				t.Errorf("HasLeadingArgsSeparator(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestIsKnownProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"custom": {Name: "Custom"},
		},
	}

	if !IsKnownProvider("custom", cfg) {
		t.Error("expected configured provider to be known")
	}
	if !IsKnownProvider("anthropic", cfg) {
		t.Error("expected built-in provider to be known")
	}
	if IsKnownProvider("nonexistent", cfg) {
		t.Error("expected unknown provider to be not known")
	}
}

func TestProviderFromArgs(t *testing.T) {
	cfg := &config.Config{DefaultProvider: "anthropic"}

	t.Run("named provider", func(t *testing.T) {
		name, args, err := ProviderFromArgs(cfg, []string{"zai", "hello"})
		if err != nil || name != "zai" || len(args) != 1 || args[0] != "hello" {
			t.Errorf("ProviderFromArgs() = %q, %v, %v", name, args, err)
		}
	})

	t.Run("flag without default", func(t *testing.T) {
		_, _, err := ProviderFromArgs(&config.Config{}, []string{"--flag"})
		if !errors.Is(err, ErrFlagWithoutProvider) {
			t.Errorf("ProviderFromArgs() err = %v, want ErrFlagWithoutProvider", err)
		}
	})

	t.Run("flag with default uses default provider", func(t *testing.T) {
		name, args, err := ProviderFromArgs(cfg, []string{"--flag", "value"})
		if err != nil || name != "anthropic" || len(args) != 2 {
			t.Errorf("ProviderFromArgs() = %q, %v, %v", name, args, err)
		}
	})
}

func TestResolveExecution(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "zai",
		DefaultHarness:  "pi",
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI", Model: "glm-5"},
		},
	}

	t.Run("empty args uses default provider", func(t *testing.T) {
		res, err := ResolveExecution(cfg, ResolveOptions{})
		if err != nil {
			t.Fatalf("ResolveExecution() error = %v", err)
		}
		if res.ProviderName != "zai" || res.Harness != harness.Pi {
			t.Errorf("ResolveExecution() = %+v", res)
		}
	})

	t.Run("explicit default provider", func(t *testing.T) {
		res, err := ResolveExecution(cfg, ResolveOptions{
			Args:                    []string{"hello"},
			DefaultProviderExplicit: true,
		})
		if err != nil {
			t.Fatalf("ResolveExecution() error = %v", err)
		}
		if res.ProviderName != "zai" {
			t.Errorf("ResolveExecution() = %+v", res)
		}
	})

	t.Run("named provider", func(t *testing.T) {
		res, err := ResolveExecution(cfg, ResolveOptions{Args: []string{"zai", "hello"}})
		if err != nil {
			t.Fatalf("ResolveExecution() error = %v", err)
		}
		if res.ProviderName != "zai" || len(res.HarnessArgs) != 1 || res.HarnessArgs[0] != "hello" {
			t.Errorf("ResolveExecution() = %+v", res)
		}
	})

	t.Run("missing provider", func(t *testing.T) {
		_, err := ResolveExecution(cfg, ResolveOptions{Args: []string{"nope"}})
		var notCfg *ProviderNotConfiguredError
		if !errors.As(err, &notCfg) || notCfg.Name != "nope" {
			t.Errorf("ResolveExecution() err = %v, want ProviderNotConfiguredError(nope)", err)
		}
		if !errors.Is(err, ErrProviderNotConfigured) {
			t.Error("expected errors.Is ErrProviderNotConfigured")
		}
	})

	t.Run("no default provider", func(t *testing.T) {
		_, err := ResolveExecution(&config.Config{}, ResolveOptions{})
		if !errors.Is(err, ErrNoDefaultProvider) {
			t.Errorf("ResolveExecution() err = %v, want ErrNoDefaultProvider", err)
		}
	})
}
