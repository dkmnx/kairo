// Package cmd implements the Kairo CLI application using the Cobra framework.
package cmd

import (
	"context"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

type cliContextKey struct{}

// CLIContext holds all CLI configuration state.
// This replaces global variables for better testability and encapsulation.
type CLIContext struct {
	configDir   string
	configDirMu sync.RWMutex
	verbose     bool
	verboseMu   sync.RWMutex
	configCache *config.ConfigCache
	rootCtx     context.Context
	rootCtxOnce sync.Once
}

// NewCLIContext creates a new CLIContext with default values.
func NewCLIContext() *CLIContext {
	return &CLIContext{
		configCache: config.NewConfigCache(configCacheTTL),
		rootCtx:     context.Background(),
	}
}

// GetConfigDir returns the configuration directory.
func (c *CLIContext) GetConfigDir() string {
	c.configDirMu.RLock()
	defer c.configDirMu.RUnlock()

	if c.configDir != "" {
		return c.configDir
	}

	dir, err := config.GetConfigDir()
	if err != nil {
		return ""
	}

	return dir
}

// SetConfigDir sets the configuration directory.
func (c *CLIContext) SetConfigDir(dir string) {
	c.configDirMu.Lock()
	defer c.configDirMu.Unlock()

	c.configDir = dir
}

// GetVerbose returns whether verbose mode is enabled.
func (c *CLIContext) GetVerbose() bool {
	c.verboseMu.RLock()
	defer c.verboseMu.RUnlock()

	return c.verbose
}

// SetVerbose sets verbose mode.
func (c *CLIContext) SetVerbose(enabled bool) {
	c.verboseMu.Lock()
	defer c.verboseMu.Unlock()

	c.verbose = enabled
}

// GetConfigCache returns the configuration cache.
func (c *CLIContext) GetConfigCache() *config.ConfigCache {
	return c.configCache
}

// GetRootCtx returns the root context for command execution.
func (c *CLIContext) GetRootCtx() context.Context {
	c.rootCtxOnce.Do(func() {
		if c.rootCtx == nil {
			c.rootCtx = context.Background()
		}
	})

	return c.rootCtx
}

// InvalidateCache invalidates the configuration cache for a given directory.
func (c *CLIContext) InvalidateCache(dir string) {
	c.configCache.Invalidate(dir)
}

// defaultCLIContext is the default CLIContext instance used when no context is set.
var defaultCLIContext = NewCLIContext()

// GetCLIContext retrieves the CLIContext from a cobra command.
// Falls back to defaultCLIContext if no context is set.
func GetCLIContext(cmd *cobra.Command) *CLIContext {
	if cmd == nil {
		return defaultCLIContext
	}

	if ctx := cmd.Context(); ctx != nil {
		if cliCtx, ok := ctx.Value(cliContextKey{}).(*CLIContext); ok {
			return cliCtx
		}
	}

	return defaultCLIContext
}

// WithCLIContext returns a context with the CLIContext attached.
func WithCLIContext(ctx context.Context, cliCtx *CLIContext) context.Context {
	return context.WithValue(ctx, cliContextKey{}, cliCtx)
}
