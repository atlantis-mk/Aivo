package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	defaultProviderEcosystemURL = "https://models.dev/api.json"
	providerEcosystemMaxBytes   = 32 << 20
)

type providerEcosystemCache struct {
	Version     int                                `json:"version"`
	Source      string                             `json:"source"`
	RefreshedAt string                             `json:"refreshedAt"`
	Providers   map[string]modelsDevProviderRecord `json:"providers"`
}

type modelsDevProviderRecord struct {
	ID     string                          `json:"id"`
	Name   string                          `json:"name"`
	API    string                          `json:"api"`
	NPM    string                          `json:"npm"`
	Env    []string                        `json:"env"`
	Models map[string]modelsDevModelRecord `json:"models"`
}

type modelsDevModelRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Attachment bool   `json:"attachment"`
	Reasoning  bool   `json:"reasoning"`
	ToolCall   bool   `json:"tool_call"`
	Modalities struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"modalities"`
	Limit struct {
		Context int `json:"context"`
		Output  int `json:"output"`
	} `json:"limit"`
	Cost modelsDevCost `json:"cost"`
}

// models.dev may include structured pricing metadata such as tier arrays next
// to its flat input/output prices. Aivo's current ModelInfo contract stores
// only flat prices, so retain numeric entries without rejecting the catalog
// when additional structured fields are present.
type modelsDevCost map[string]json.RawMessage

func (cost modelsDevCost) flatPricing() map[string]float64 {
	pricing := make(map[string]float64)
	for key, raw := range cost {
		var value any
		if err := json.Unmarshal(raw, &value); err == nil {
			if numeric, ok := value.(float64); ok {
				pricing[key] = numeric
			}
		}
	}
	if len(pricing) == 0 {
		return nil
	}
	return pricing
}

func (s *Service) RefreshProviderEcosystemCatalog(ctx context.Context, input domain.ProviderEcosystemRefreshInput) (domain.ProviderEcosystemRefreshResult, error) {
	source := strings.TrimSpace(input.URL)
	if source == "" {
		source = strings.TrimSpace(os.Getenv("AIVO_MODELS_URL"))
	}
	if source == "" {
		source = defaultProviderEcosystemURL
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHostname(parsed.Hostname()))) {
		return domain.ProviderEcosystemRefreshResult{}, errors.New("provider ecosystem URL must use HTTPS (HTTP is allowed only for loopback testing)")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return domain.ProviderEcosystemRefreshResult{}, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return domain.ProviderEcosystemRefreshResult{}, fmt.Errorf("refresh provider ecosystem: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return domain.ProviderEcosystemRefreshResult{}, fmt.Errorf("refresh provider ecosystem: HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, providerEcosystemMaxBytes+1))
	if err != nil {
		return domain.ProviderEcosystemRefreshResult{}, err
	}
	if len(raw) > providerEcosystemMaxBytes {
		return domain.ProviderEcosystemRefreshResult{}, fmt.Errorf("provider ecosystem response exceeds %d bytes", providerEcosystemMaxBytes)
	}
	var providers map[string]modelsDevProviderRecord
	if err := json.Unmarshal(raw, &providers); err != nil {
		return domain.ProviderEcosystemRefreshResult{}, fmt.Errorf("decode provider ecosystem: %w", err)
	}
	if len(providers) == 0 {
		return domain.ProviderEcosystemRefreshResult{}, errors.New("provider ecosystem returned no providers")
	}
	refreshedAt := domain.NowString(s.now())
	cache := providerEcosystemCache{Version: 1, Source: source, RefreshedAt: refreshedAt, Providers: providers}
	cachePath, err := providerEcosystemCachePath()
	if err != nil {
		return domain.ProviderEcosystemRefreshResult{}, err
	}
	if err := writeProviderEcosystemCache(cachePath, cache); err != nil {
		return domain.ProviderEcosystemRefreshResult{}, err
	}
	// Rebuild only the global registry. Project catalogs and agent runs build
	// isolated registries from the same immutable cache on demand.
	s.refreshProviderExtensions("")
	definitions, unsupported := providerDefinitionsFromEcosystem(cache, NewDefaultProviderRegistry())
	modelCount := 0
	for _, definition := range definitions {
		modelCount += len(definition.Models)
	}
	return domain.ProviderEcosystemRefreshResult{
		Source: source, CachePath: cachePath, RefreshedAt: refreshedAt,
		ProviderCount: len(definitions), ModelCount: modelCount, UnsupportedCount: unsupported,
	}, nil
}

func loadProviderEcosystemDefinitions(base *ProviderRegistry) []ProviderDefinition {
	cachePath, err := providerEcosystemCachePath()
	if err != nil {
		return nil
	}
	raw, err := os.ReadFile(cachePath)
	if err != nil || len(raw) > providerEcosystemMaxBytes {
		return nil
	}
	var cache providerEcosystemCache
	if json.Unmarshal(raw, &cache) != nil || cache.Version != 1 {
		return nil
	}
	definitions, _ := providerDefinitionsFromEcosystem(cache, base)
	return definitions
}

func providerDefinitionsFromEcosystem(cache providerEcosystemCache, base *ProviderRegistry) ([]ProviderDefinition, int) {
	ids := make([]string, 0, len(cache.Providers))
	for id := range cache.Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	definitions := make([]ProviderDefinition, 0, len(ids))
	unsupported := 0
	for _, key := range ids {
		record := cache.Providers[key]
		id := normalizeProviderKey(firstNonEmpty(record.ID, key))
		if id == "" {
			continue
		}
		definition, exists := base.Definition(id)
		if !exists {
			transport, supported := modelsDevTransport(record.NPM)
			if !supported || strings.TrimSpace(record.API) == "" {
				unsupported++
				continue
			}
			authTypes := []AuthType{AuthAPIKey}
			defaultAuth := AuthAPIKey
			if len(record.Env) == 0 {
				authTypes, defaultAuth = []AuthType{AuthNone}, AuthNone
			}
			definition = ProviderDefinition{
				ID: id, DisplayName: firstNonEmpty(record.Name, id), Transport: transport,
				AuthTypes: authTypes, DefaultAuthType: defaultAuth, DefaultBaseURL: strings.TrimRight(record.API, "/"),
				APIKeyEnvVars: nonEmptyTrimmedStrings(record.Env), ModelFetch: modelsDevModelFetch(transport), BuiltIn: false,
			}
		} else {
			definition.DisplayName = firstNonEmpty(record.Name, definition.DisplayName)
			if definition.DefaultBaseURL == "" {
				definition.DefaultBaseURL = strings.TrimRight(record.API, "/")
			}
			if len(definition.APIKeyEnvVars) == 0 {
				definition.APIKeyEnvVars = nonEmptyTrimmedStrings(record.Env)
			}
		}
		definition.Models = modelsFromEcosystem(id, record.Models, cache.RefreshedAt)
		if len(definition.Models) == 0 {
			unsupported++
			continue
		}
		definition.DefaultModelID = definition.Models[0].ID
		definition.Models[0].Recommended = true
		definitions = append(definitions, definition)
	}
	return definitions, unsupported
}

func modelsDevModelFetch(transport TransportType) ModelFetchStrategy {
	switch transport {
	case TransportAnthropicMessages:
		return ModelFetchAnthropic
	case TransportGoogleGemini:
		return ModelFetchGoogle
	case TransportOpenAICompatible, TransportAzureOpenAI:
		return ModelFetchOpenAICompatible
	default:
		return ModelFetchStatic
	}
}

func modelsFromEcosystem(providerID string, records map[string]modelsDevModelRecord, refreshedAt string) []domain.ModelInfo {
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	models := make([]domain.ModelInfo, 0, len(ids))
	for _, key := range ids {
		record := records[key]
		id := strings.TrimSpace(firstNonEmpty(record.ID, key))
		if id == "" {
			continue
		}
		capabilities := []string{"text"}
		if record.ToolCall {
			capabilities = append(capabilities, "tools")
		}
		if record.Reasoning {
			capabilities = append(capabilities, "reasoning")
		}
		if record.Attachment {
			capabilities = append(capabilities, "attachments")
		}
		modalities := append([]string{}, record.Modalities.Input...)
		for _, modality := range record.Modalities.Output {
			if !stringSliceContains(modalities, modality) {
				modalities = append(modalities, modality)
			}
		}
		models = append(models, domain.ModelInfo{
			ID: id, ProviderID: providerID, Name: firstNonEmpty(record.Name, id), ContextLength: record.Limit.Context,
			OutputLimit: record.Limit.Output, Capabilities: capabilities, Modalities: modalities, Streaming: true,
			ToolSupport: record.ToolCall, Pricing: record.Cost.flatPricing(), LastRefreshed: refreshedAt,
		})
	}
	return models
}

func modelsDevTransport(npm string) (TransportType, bool) {
	npm = strings.ToLower(strings.TrimSpace(npm))
	switch {
	case strings.Contains(npm, "anthropic"):
		return TransportAnthropicMessages, true
	case strings.Contains(npm, "google") && !strings.Contains(npm, "vertex"):
		return TransportGoogleGemini, true
	case strings.Contains(npm, "azure"):
		return TransportAzureOpenAI, true
	case strings.Contains(npm, "bedrock"):
		return TransportBedrockConverse, true
	case strings.Contains(npm, "openai-compatible"), strings.Contains(npm, "openai"),
		strings.Contains(npm, "groq"), strings.Contains(npm, "mistral"), strings.Contains(npm, "xai"),
		strings.Contains(npm, "together"), strings.Contains(npm, "perplexity"), strings.Contains(npm, "deepinfra"),
		strings.Contains(npm, "cerebras"), strings.Contains(npm, "openrouter"), strings.Contains(npm, "gateway"),
		strings.Contains(npm, "baseten"), strings.Contains(npm, "deepseek"), strings.Contains(npm, "fireworks"):
		return TransportOpenAICompatible, true
	default:
		return "", false
	}
}

func providerEcosystemCachePath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("AIVO_MODELS_CACHE")); configured != "" {
		return filepath.Abs(configured)
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "aivo", "models-dev.json"), nil
}

func writeProviderEcosystemCache(path string, cache providerEcosystemCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func cloneFloatMap(input map[string]float64) map[string]float64 {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]float64, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringSliceContains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func isLoopbackHostname(hostname string) bool {
	if strings.EqualFold(strings.TrimSpace(hostname), "localhost") {
		return true
	}
	address := net.ParseIP(strings.TrimSpace(hostname))
	return address != nil && address.IsLoopback()
}
