package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
)

// ResolveOptions carries CLI inputs needed to pick a provider and harness args.
type ResolveOptions struct {
	Args                    []string
	HarnessFlag             string
	DefaultProviderExplicit bool
}

// ResolveResult is the resolved launch target for a harness session.
type ResolveResult struct {
	ProviderName string
	Provider     config.Provider
	Harness      string
	HarnessArgs  []string
}

// Sentinel errors returned by resolution. Callers map them to user-facing text.
var (
	ErrNoDefaultProvider     = errors.New("no default provider set")
	ErrProviderNotConfigured = errors.New("provider not configured")
	ErrFlagWithoutProvider   = errors.New("no default provider and first argument looks like a flag")
)

// ProviderNotConfiguredError names the provider missing from the config.
type ProviderNotConfiguredError struct {
	Name string
}

func (e *ProviderNotConfiguredError) Error() string {
	return fmt.Sprintf("provider not configured: %s", e.Name)
}

func (e *ProviderNotConfiguredError) Is(target error) bool {
	return target == ErrProviderNotConfigured
}

// SplitArgs splits args at the first "--" separator.
func SplitArgs(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}

	return args, nil
}

// HasLeadingArgsSeparator reports whether args contain the "--" separator before
// any non-flag argument. It walks past flags and their values (e.g. --harness pi,
// -v value) looking for "--". Once a non-flag argument is seen, "--" is no longer
// valid as a separator. Note: SplitArgs always splits on the first "--" regardless
// of position, so this function has different semantics — it only detects separators
// that appear before the first positional argument.
func HasLeadingArgsSeparator(args []string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			return true
		}
		if args[i] == "-" || !strings.HasPrefix(args[i], "-") {
			return false
		}
		if strings.Contains(args[i], "=") {
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}

	return false
}

// IsKnownProvider reports whether name matches a configured or built-in provider.
func IsKnownProvider(name string, cfg *config.Config) bool {
	if _, ok := cfg.Providers[name]; ok {
		return true
	}

	return providers.IsBuiltInProvider(name)
}

// ProviderFromArgs extracts the provider name from the first non-flag argument.
func ProviderFromArgs(cfg *config.Config, args []string) (string, []string, error) {
	kairoArgs, harnessArgs := SplitArgs(args)

	if len(kairoArgs) > 0 && !strings.HasPrefix(kairoArgs[0], "-") {
		harnessArgs = append(kairoArgs[1:], harnessArgs...)

		return kairoArgs[0], harnessArgs, nil
	}

	if cfg.DefaultProvider != "" {
		return cfg.DefaultProvider, kairoArgs, nil
	}

	return "", nil, fmt.Errorf("%w", ErrFlagWithoutProvider)
}

// ResolveProviderAndArgs resolves the provider name and harness arguments from
// command-line args and configuration.
func ResolveProviderAndArgs(cfg *config.Config, opts ResolveOptions) ([]string, string, error) {
	if len(opts.Args) == 0 || opts.DefaultProviderExplicit {
		if cfg.DefaultProvider == "" {
			return nil, "", ErrNoDefaultProvider
		}

		return opts.Args, cfg.DefaultProvider, nil
	}

	providerName, harnessArgs, err := ProviderFromArgs(cfg, opts.Args)
	if err != nil {
		return nil, "", err
	}

	// When --harness is set and the first arg is not a known provider,
	// treat all args as harness args and use the default provider.
	if opts.HarnessFlag != "" && !IsKnownProvider(providerName, cfg) && cfg.DefaultProvider != "" {
		return opts.Args, cfg.DefaultProvider, nil
	}

	return harnessArgs, providerName, nil
}

// ResolveExecution picks the provider entry and harness for a launch.
func ResolveExecution(cfg *config.Config, opts ResolveOptions) (ResolveResult, error) {
	harnessArgs, providerName, err := ResolveProviderAndArgs(cfg, opts)
	if err != nil {
		return ResolveResult{}, err
	}

	provider, ok := cfg.Providers[providerName]
	if !ok {
		return ResolveResult{}, &ProviderNotConfiguredError{Name: providerName}
	}

	return ResolveResult{
		ProviderName: providerName,
		Provider:     provider,
		Harness:      harness.Resolve(opts.HarnessFlag, cfg.DefaultHarness),
		HarnessArgs:  harnessArgs,
	}, nil
}
