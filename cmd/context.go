package cmd

import (
	"context"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/constants"
	"github.com/dkmnx/kairo/internal/crypto"
	"github.com/spf13/cobra"
)

type cliContextKey struct{}

// ConfigDirResolver resolves the default configuration directory.
type ConfigDirResolver func() (string, error)

// CLIContext holds shared CLI state: config directory, verbosity, config cache,
// root context, and external dependencies. It is safe for concurrent use.
type CLIContext struct {
	configDir         string
	configDirMu       sync.RWMutex
	configDirResolver ConfigDirResolver
	verbose           bool
	verboseMu         sync.RWMutex
	configCache       *config.ConfigCache
	rootCtx           context.Context
	deps              *Deps

	defaultProviderExplicit bool
}

// NewCLIContext creates a CLIContext with default settings.
func NewCLIContext() *CLIContext {
	return &CLIContext{
		configDirResolver: config.DefaultConfigDir,
		configCache:       config.NewConfigCache(constants.ConfigCacheTTL),
		rootCtx:           context.Background(),
		deps:              NewDeps(),
	}
}

// ConfigDir returns the active configuration directory.
func (c *CLIContext) ConfigDir() string {
	c.configDirMu.RLock()
	defer c.configDirMu.RUnlock()

	if c.configDir != "" {
		return c.configDir
	}

	dir, err := c.configDirResolver()
	if err != nil {
		return ""
	}

	return dir
}

// SetConfigDirResolver sets the function used to locate the config directory.
func (c *CLIContext) SetConfigDirResolver(r ConfigDirResolver) {
	c.configDirMu.Lock()
	defer c.configDirMu.Unlock()

	c.configDirResolver = r
}

// SetConfigDir overrides the configuration directory.
func (c *CLIContext) SetConfigDir(dir string) {
	c.configDirMu.Lock()
	defer c.configDirMu.Unlock()

	c.configDir = dir
}

// Verbose returns whether verbose output is enabled.
func (c *CLIContext) Verbose() bool {
	c.verboseMu.RLock()
	defer c.verboseMu.RUnlock()

	return c.verbose
}

// SetVerbose enables or disables verbose output.
func (c *CLIContext) SetVerbose(enabled bool) {
	c.verboseMu.Lock()
	defer c.verboseMu.Unlock()

	c.verbose = enabled
}

// ConfigCache returns the configuration cache instance.
func (c *CLIContext) ConfigCache() *config.ConfigCache {
	return c.configCache
}

// RootCtx returns the root context for the CLI session.
func (c *CLIContext) RootCtx() context.Context {
	return c.rootCtx
}

// Deps returns the external dependencies for this CLI session.
func (c *CLIContext) Deps() *Deps {
	return c.deps
}

// Crypto returns the crypto service for this CLI session.
func (c *CLIContext) Crypto() crypto.Service {
	return c.deps.Crypto
}

// SetDeps replaces the external dependencies. For use in tests.
func (c *CLIContext) SetDeps(d *Deps) {
	c.deps = d
}

// InvalidateCache removes the cached configuration for the given directory.
func (c *CLIContext) InvalidateCache(dir string) {
	c.configCache.Invalidate(dir)
}

// SetDefaultProviderExplicit records whether the user passed "--" to separate
// kairo flags from harness flags.
func (c *CLIContext) SetDefaultProviderExplicit(v bool) {
	c.defaultProviderExplicit = v
}

// DefaultProviderExplicit reports whether the user passed "--".
func (c *CLIContext) DefaultProviderExplicit() bool {
	return c.defaultProviderExplicit
}

// CLIContextFromCmd extracts the CLIContext from a cobra command's context.
// Returns nil if no CLIContext is set (callers should use MustCLIContextFromCmd
// when a cmd is always available, or handle nil gracefully).
func CLIContextFromCmd(cmd *cobra.Command) *CLIContext {
	if ctx := cmd.Context(); ctx != nil {
		if cliCtx, ok := ctx.Value(cliContextKey{}).(*CLIContext); ok {
			return cliCtx
		}
	}

	return nil
}

// MustCLIContextFromCmd is like CLIContextFromCmd but panics if no CLIContext
// is found. Use when a cobra command is guaranteed to have been initialized
// via Execute().
func MustCLIContextFromCmd(cmd *cobra.Command) *CLIContext {
	cliCtx := CLIContextFromCmd(cmd)
	if cliCtx == nil {
		panic("kairo: no CLIContext in command context; Execute() must be called first")
	}

	return cliCtx
}

// WithCLIContext stores a CLIContext in the given context.
func WithCLIContext(ctx context.Context, cliCtx *CLIContext) context.Context {
	return context.WithValue(ctx, cliContextKey{}, cliCtx)
}
