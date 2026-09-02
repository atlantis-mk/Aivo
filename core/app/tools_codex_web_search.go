package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aivo/core/domain"
)

const (
	chatGPTCodexSearchURL       = "https://chatgpt.com/backend-api/codex/alpha/search"
	codexSearchMaxResponseBytes = 2 * 1024 * 1024
)

var codexSearchHTTPClient = &http.Client{Timeout: 30 * time.Second}

type CodexWebSearchTool struct {
	service *Service
}

func NewCodexWebSearchTool(service *Service) *CodexWebSearchTool {
	return &CodexWebSearchTool{service: service}
}

func (t *CodexWebSearchTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:             "web_search",
		Description:      "Search the public web through the authenticated Codex search service and return bounded source-backed results.",
		Capability:       "web.search",
		RiskLevel:        "medium",
		Category:         "web",
		Toolsets:         []string{"web", "coding"},
		RequiresNetwork:  true,
		ActivationPolicy: providerDeclarationActivationPolicy,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":   map[string]any{"type": "string", "description": "Search query."},
				"limit":   map[string]any{"type": "integer", "minimum": 1, "maximum": webSearchMaxLimit},
				"recency": map[string]any{"type": "integer", "minimum": 1, "maximum": 3650, "description": "Optional maximum age in days."},
				"domains": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 20},
			},
			"required": []string{"query"},
		},
	}
}

func (t *CodexWebSearchTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	result := domain.ToolResult{Name: "web_search", CallID: execCtx.ToolCallID}
	if t == nil || t.service == nil {
		result.Error = "web search route is unavailable"
		return result
	}
	var input struct {
		Query   string   `json:"query"`
		Limit   int      `json:"limit"`
		Recency int      `json:"recency"`
		Domains []string `json:"domains"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	input.Query = strings.TrimSpace(input.Query)
	if input.Query == "" {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, errors.New("query is required"))
	}
	if input.Limit <= 0 || input.Limit > webSearchMaxLimit {
		input.Limit = webSearchDefaultLimit
	}
	cfg, err := t.service.AppConfig(ctx)
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	webSearch := normalizeWebSearchRuntimeConfig(cfg.WebSearch)
	if webSearch.Mode == domain.WebSearchModeDisabled || nativeToolDisabled(normalizeNativeToolsRuntimeConfig(cfg.NativeTools), "web_search") {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, errors.New("web search is disabled"))
	}
	if execCtx.ActiveModel != nil {
		route, err := t.service.ResolveModelRoute(ctx, cfg, execCtx.ActiveModel)
		if err == nil && isChatGPTCodexRoute(route) {
			model, ok := t.service.modelInfoForRoute(ctx, route)
			if ok && declaredModelCapabilitySupported(model, codexWebSearchCapability) {
				return t.executeCodexSearch(ctx, execCtx, route, model, webSearch, input.Query, input.Limit, input.Recency, input.Domains)
			}
		}
	}
	return t.executeParallelSearch(ctx, execCtx, input.Query, input.Limit)
}

func (t *CodexWebSearchTool) executeCodexSearch(ctx context.Context, execCtx domain.ToolExecutionContext, route ResolvedModelRoute, model domain.ModelInfo, webSearch domain.WebSearchConfig, query string, limit, recency int, domains []string) domain.ToolResult {
	result := domain.ToolResult{Name: "web_search", CallID: execCtx.ToolCallID}
	access, accountID, err := t.service.validOpenAIAccessToken(ctx, route.Credential)
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	body := codexSearchRequestBody(execCtx.SessionID, route.Model.ModelID, webSearch, query, limit, recency, domains)
	raw, err := json.Marshal(body)
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTCodexSearchURL, bytes.NewReader(raw))
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("User-Agent", openAIUserAgent)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	response, err := codexSearchHTTPClient.Do(req)
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	defer response.Body.Close()
	responseRaw, truncated, err := readBoundedBody(response.Body, codexSearchMaxResponseBytes)
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	if truncated {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, errors.New("Codex search response exceeded the safe limit"))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, providerHTTPError(response.StatusCode, response.Status, string(responseRaw)))
	}
	var payload struct {
		Output  string           `json:"output"`
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(responseRaw, &payload); err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, errors.New("Codex search response could not be parsed"))
	}
	sources := boundedCodexSearchSources(payload.Results, limit)
	content := strings.TrimSpace(payload.Output)
	if content == "" {
		content = formatCodexSearchSources(query, sources)
	}
	if content == "" {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, errors.New("Codex search returned no output"))
	}
	result.OK = true
	result.Content = content
	result.ModelContent = content
	result.Structured = map[string]any{"query": query, "results": sources, "provider": "codex", "status": response.StatusCode}
	return result
}

func (t *CodexWebSearchTool) executeParallelSearch(ctx context.Context, execCtx domain.ToolExecutionContext, query string, limit int) domain.ToolResult {
	backend := NewParallelSearchBackend(nil, "")
	response, err := backend.Search(ctx, WebSearchRequest{Query: query, Limit: limit})
	if err != nil {
		return toolErrorWithCallID("web_search", execCtx.ToolCallID, err)
	}
	return webSearchResponseToolResult("web_search", execCtx.ToolCallID, query, response, backend.Name())
}

func codexSearchRequestBody(sessionID, modelID string, config domain.WebSearchConfig, query string, limit, recency int, domains []string) map[string]any {
	searchQuery := map[string]any{"q": query}
	if recency > 0 {
		searchQuery["recency"] = recency
	}
	domains = normalizeWebSearchDomains(append(append([]string(nil), config.AllowedDomains...), domains...))
	if len(domains) > 0 {
		searchQuery["domains"] = domains
	}
	settings := map[string]any{
		"allowed_callers":     []string{"direct"},
		"external_web_access": codexExternalWebAccess(config.Mode),
	}
	if size := strings.TrimSpace(config.SearchContextSize); size != "" {
		settings["search_context_size"] = size
	}
	if len(domains) > 0 {
		settings["filters"] = map[string]any{"allowed_domains": domains}
	}
	if location := hostedUserLocationMap(config.UserLocation); len(location) > 0 {
		settings["user_location"] = location
	}
	return map[string]any{
		"id":                firstNonEmpty(strings.TrimSpace(sessionID), "aivo-search"),
		"model":             modelID,
		"commands":          map[string]any{"search_query": []any{searchQuery}, "response_length": codexSearchResponseLength(limit)},
		"settings":          settings,
		"max_output_tokens": 2500,
	}
}

func codexExternalWebAccess(mode string) any {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case domain.WebSearchModeCached:
		return false
	case domain.WebSearchModeIndexed:
		return "indexed"
	default:
		return true
	}
}

func codexSearchResponseLength(limit int) string {
	if limit <= 3 {
		return "short"
	}
	if limit >= 8 {
		return "long"
	}
	return "medium"
}

func boundedCodexSearchSources(results []map[string]any, limit int) []map[string]any {
	if limit <= 0 || limit > webSearchMaxLimit {
		limit = webSearchDefaultLimit
	}
	out := make([]map[string]any, 0, min(limit, len(results)))
	seen := map[string]bool{}
	for _, item := range results {
		url := strings.TrimSpace(firstString(item, "url"))
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		source := map[string]any{"url": url}
		if title := strings.TrimSpace(firstString(item, "title", "name")); title != "" {
			source["title"] = bounded(title, 500)
		}
		if refID := strings.TrimSpace(firstString(item, "ref_id", "refId")); refID != "" {
			source["refId"] = bounded(refID, 200)
		}
		out = append(out, source)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func formatCodexSearchSources(query string, sources []map[string]any) string {
	if len(sources) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("Search results for: ")
	builder.WriteString(query)
	for index, source := range sources {
		url := firstString(source, "url")
		title := firstNonEmpty(firstString(source, "title"), url)
		builder.WriteString(fmt.Sprintf("\n\n%d. %s\n%s", index+1, title, url))
	}
	return builder.String()
}

func toolErrorWithCallID(name, callID string, err error) domain.ToolResult {
	result := toolError(name, err)
	result.CallID = callID
	return result
}
