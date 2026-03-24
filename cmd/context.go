package cmd

import (
	"context"
	"sync"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/spf13/cobra"
)

type cliContextKey struct{}

type CLIContext struct {
	configDir   string
	configDirMu sync.RWMutex
	verbose     bool
	verboseMu   sync.RWMutex
	configCache *config.ConfigCache
	rootCtx     context.Context
	rootCtxOnce sync.Once
}

func NewCLIContext() *CLIContext {
	return &CLIContext{
		configCache: config.NewConfigCache(configCacheTTL),
		rootCtx:     context.Background(),
	}
}

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

func (c *CLIContext) SetConfigDir(dir string) {
	c.configDirMu.Lock()
	defer c.configDirMu.Unlock()

	c.configDir = dir
}

func (c *CLIContext) GetVerbose() bool {
	c.verboseMu.RLock()
	defer c.verboseMu.RUnlock()

	return c.verbose
}

func (c *CLIContext) SetVerbose(enabled bool) {
	c.verboseMu.Lock()
	defer c.verboseMu.Unlock()

	c.verbose = enabled
}

func (c *CLIContext) GetConfigCache() *config.ConfigCache {
	return c.configCache
}

func (c *CLIContext) GetRootCtx() context.Context {
	c.rootCtxOnce.Do(func() {
		if c.rootCtx == nil {
			c.rootCtx = context.Background()
		}
	})

	return c.rootCtx
}

func (c *CLIContext) InvalidateCache(dir string) {
	c.configCache.Invalidate(dir)
}

var defaultCLIContext = NewCLIContext()

func GetCLIContext(cmd *cobra.Command) *CLIContext {
	if ctx := cmd.Context(); ctx != nil {
		if cliCtx, ok := ctx.Value(cliContextKey{}).(*CLIContext); ok {
			return cliCtx
		}
	}

	return defaultCLIContext
}

func WithCLIContext(ctx context.Context, cliCtx *CLIContext) context.Context {
	return context.WithValue(ctx, cliContextKey{}, cliCtx)
}
