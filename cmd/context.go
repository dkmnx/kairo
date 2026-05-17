package cmd

import (
	"context"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

type cliContextKey struct{}

// CLIContext holds shared CLI state: config directory, verbosity, config cache,
// root context, and external dependencies. It is safe for concurrent use.
type CLIContext struct {
	configDir   string
	configDirMu sync.RWMutex
	verbose     bool
	verboseMu   sync.RWMutex
	configCache *config.ConfigCache
	rootCtx     context.Context
	deps        *Deps
}

// NewCLIContext creates a CLIContext with default settings.
func NewCLIContext() *CLIContext {
	return &CLIContext{
		configCache: config.NewConfigCache(configCacheTTL),
		rootCtx:     context.Background(),
		deps:        NewDeps(),
	}
}

// ConfigDir returns the active configuration directory.
func (c *CLIContext) ConfigDir() string {
	c.configDirMu.RLock()
	defer c.configDirMu.RUnlock()

	if c.configDir != "" {
		return c.configDir
	}

	dir, err := config.ConfigDir()
	if err != nil {
		return ""
	}

	return dir
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

// SetDeps replaces the external dependencies. For use in tests.
func (c *CLIContext) SetDeps(d *Deps) {
	c.deps = d
}

// InvalidateCache removes the cached configuration for the given directory.
func (c *CLIContext) InvalidateCache(dir string) {
	c.configCache.Invalidate(dir)
}

var defaultCLIContext = NewCLIContext()

// CLIContextFromCmd extracts the CLIContext from a cobra command's context.
func CLIContextFromCmd(cmd *cobra.Command) *CLIContext {
	if ctx := cmd.Context(); ctx != nil {
		if cliCtx, ok := ctx.Value(cliContextKey{}).(*CLIContext); ok {
			return cliCtx
		}
	}

	return defaultCLIContext
}

// WithCLIContext stores a CLIContext in the given context.
func WithCLIContext(ctx context.Context, cliCtx *CLIContext) context.Context {
	return context.WithValue(ctx, cliContextKey{}, cliCtx)
}
