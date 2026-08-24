package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"aivo/core/domain"
)

func TestProviderEcosystemRefreshCachesSupportedCatalogForOfflineStartup(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "models-dev.json")
	t.Setenv("AIVO_MODELS_CACHE", cachePath)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "acme": {"id":"acme","name":"Acme AI","api":"https://api.acme.test/v1","npm":"@ai-sdk/openai-compatible","env":["ACME_API_KEY"],"models":{"acme-pro":{"id":"acme-pro","name":"Acme Pro","tool_call":true,"reasoning":true,"modalities":{"input":["text","image"],"output":["text"]},"limit":{"context":131072,"output":8192},"cost":{"input":0.2,"output":0.8,"cache_read":null,"tiers":[{"input":0.4,"output":1.6,"tier":{"type":"context","size":200000}}],"context_over_200k":{"input":0.4,"output":1.6}}}}},
  "anthropic": {"id":"anthropic","name":"Anthropic","api":"https://api.anthropic.com/v1","npm":"@ai-sdk/anthropic","env":["ANTHROPIC_API_KEY"],"models":{"claude-test":{"id":"claude-test","name":"Claude Test","tool_call":true,"limit":{"context":200000,"output":8192}}}},
  "cohere-only": {"id":"cohere-only","name":"Cohere Only","api":"https://api.cohere.test","npm":"@ai-sdk/cohere","models":{"command":{"id":"command","name":"Command"}}}
}`))
	}))
	defer server.Close()
	service := NewService(&memoryProviderStore{})
	result, err := service.RefreshProviderEcosystemCatalog(context.Background(), domain.ProviderEcosystemRefreshInput{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderCount != 2 || result.ModelCount != 2 || result.UnsupportedCount != 1 || result.CachePath != cachePath {
		t.Fatalf("refresh result = %#v", result)
	}
	definition, ok := service.providerDefinition("acme")
	if !ok || definition.Transport != TransportOpenAICompatible || definition.ModelFetch != ModelFetchOpenAICompatible || !providerModelRefreshable(definition) || definition.DefaultBaseURL != "https://api.acme.test/v1" || len(definition.Models) != 1 {
		t.Fatalf("dynamic definition = %#v ok = %v", definition, ok)
	}
	model := definition.Models[0]
	if model.ContextLength != 131072 || model.OutputLimit != 8192 || !model.ToolSupport || model.Pricing["input"] != 0.2 || model.Pricing["output"] != 0.8 || len(model.Pricing) != 2 {
		t.Fatalf("dynamic model = %#v", model)
	}
	anthropic, ok := service.providerDefinition("anthropic")
	if !ok || anthropic.Transport != TransportAnthropicMessages || len(anthropic.Models) != 1 {
		t.Fatalf("native definition was not preserved: %#v", anthropic)
	}

	offline := NewService(&memoryProviderStore{})
	offlineDefinition, ok := offline.providerDefinition("acme")
	if !ok || len(offlineDefinition.Models) != 1 || offlineDefinition.Models[0].ID != "acme-pro" {
		t.Fatalf("offline cache definition = %#v ok = %v", offlineDefinition, ok)
	}
}

func TestProjectProviderRegistriesRemainIsolatedAcrossConcurrentCatalogsAndRoutes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AIVO_MODELS_CACHE", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("TENANT_KEY", "test")
	rootA, rootB := t.TempDir(), t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(rootA, "aivo.json"), `{"providerExtensions":{"tenant":{"protocol":"openai-compatible","baseUrl":"https://a.example/v1","credentialRef":"TENANT_KEY","models":["a-model"]}}}`)
	writeRuntimeConfigTestFile(t, filepath.Join(rootB, "aivo.json"), `{"providerExtensions":{"tenant":{"protocol":"openai-compatible","baseUrl":"https://b.example/v1","credentialRef":"TENANT_KEY","models":["b-model"]}}}`)
	service := NewService(&memoryProviderStore{})
	ctx := context.Background()
	var wait sync.WaitGroup
	errorsCh := make(chan string, 40)
	for iteration := 0; iteration < 20; iteration++ {
		for _, item := range []struct {
			root  string
			url   string
			model string
		}{{rootA, "https://a.example/v1", "a-model"}, {rootB, "https://b.example/v1", "b-model"}} {
			wait.Add(1)
			go func(item struct{ root, url, model string }) {
				defer wait.Done()
				registry := service.providerRegistryForProject(item.root)
				definition, ok := registry.Definition("tenant")
				if !ok || definition.DefaultBaseURL != item.url || definition.DefaultModelID != item.model {
					errorsCh <- item.root
					return
				}
				cfg := domain.AppConfig{DefaultModel: &domain.ModelRef{ProviderID: "tenant", ModelID: item.model}}
				route, err := service.ResolveModelRoute(withProviderRegistry(ctx, registry), cfg, cfg.DefaultModel)
				if err != nil || route.BaseURL != item.url || route.Model.ModelID != item.model {
					errorsCh <- item.root
				}
			}(item)
		}
	}
	wait.Wait()
	close(errorsCh)
	for failed := range errorsCh {
		t.Fatalf("project provider registry leaked for %s", failed)
	}
	if _, ok := service.providerDefinition("tenant"); ok {
		t.Fatal("project provider leaked into the global registry")
	}
}
