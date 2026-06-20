package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/dkmnx/kairo/internal/providers"
)

func TestProvidersListCmd(t *testing.T) {
	d := testDepsWithCatalog(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate, mc *mockCatalog) {
		mc.ProviderListFn = func() []string {
			return []string{"anthropic", "deepseek", "zai"}
		}
		mc.ProviderSourceFn = func(name string) string {
			switch name {
			case "zai":
				return "embedded"
			case "deepseek":
				return "embedded"
			case "anthropic":
				return "embedded"
			default:
				return ""
			}
		}
		mc.BuiltInProviderFn = func(name string) (providers.ProviderDefinition, bool) {
			switch name {
			case "zai":
				return providers.ProviderDefinition{Name: "Z.AI", BaseURL: "https://api.z.ai", Model: "glm-5.1", APIKeyEnvVar: "ZAI_API_KEY", RequiresAPIKey: true}, true
			case "deepseek":
				return providers.ProviderDefinition{Name: "DeepSeek AI", BaseURL: "https://api.deepseek.com", RequiresAPIKey: true, APIKeyEnvVar: "DEEPSEEK_API_KEY"}, true
			case "anthropic":
				return providers.ProviderDefinition{Name: "Anthropic", RequiresAPIKey: true, APIKeyEnvVar: "ANTHROPIC_API_KEY"}, true
			default:
				return providers.ProviderDefinition{}, false
			}
		}
	})

	cliCtx := NewCLIContext()
	cliCtx.SetDeps(d)
	providersListCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	err := providersListCmd.Execute()
	if err != nil {
		t.Fatalf("providers list failed: %v", err)
	}
}

func TestProvidersRefreshCmd_Success(t *testing.T) {
	d := testDepsWithCatalog(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate, mc *mockCatalog) {
		mc.RefreshFromRemoteFn = func(ctx context.Context) (int, error) {
			return 5, nil
		}
	})

	cliCtx := NewCLIContext()
	cliCtx.SetDeps(d)
	providersRefreshCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	if err := providersRefreshCmd.Execute(); err != nil {
		t.Fatalf("providers refresh failed: %v", err)
	}
}

func TestProvidersRefreshCmd_Error(t *testing.T) {
	d := testDepsWithCatalog(func(mp *mockProcess, mw *mockWrapper, mu *mockUpdate, mc *mockCatalog) {
		mc.RefreshFromRemoteFn = func(ctx context.Context) (int, error) {
			return 0, errors.New("network down")
		}
	})

	cliCtx := NewCLIContext()
	cliCtx.SetDeps(d)
	providersRefreshCmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	// Error should be printed, not returned
	_ = providersRefreshCmd.Execute()
}
