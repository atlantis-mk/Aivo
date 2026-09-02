package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"aivo/core/domain"
)

const (
	webNamespace            = "functions"
	webNamespaceDescription = "Network read tools. Use web_fetch to inspect a known URL and web_search to find public web pages before fetching a result. Network output is bounded and converted to text for model context."
	webFetchDefaultMaxChars = 12000
	webFetchMaxChars        = 50000
	webFetchMaxBytes        = 2 * 1024 * 1024
	webSearchDefaultLimit   = 5
	webSearchMaxLimit       = 10
	defaultWebSearchURL     = "https://duckduckgo.com/html/"
)

type WebFetchTool struct {
	client *http.Client
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{client: defaultWebClient()}
}

func (t *WebFetchTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "web_fetch",
		Description:          "Fetch a known http(s) URL and return bounded readable text. Use this for docs, issues, release notes, API references, or error pages when the exact URL is known. Do not use for searching; use web_search first. Binary responses are rejected.",
		Namespace:            webNamespace,
		NamespaceDescription: webNamespaceDescription,
		Capability:           "web.fetch",
		RiskLevel:            "medium",
		Category:             "web",
		Toolsets:             []string{"web", "coding"},
		RequiresNetwork:      true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":      map[string]any{"type": "string", "description": "Absolute http(s) URL to fetch."},
				"maxChars": map[string]any{"type": "integer", "description": "Maximum characters to return. Defaults to 12000; max 50000.", "minimum": 1000, "maximum": webFetchMaxChars},
			},
			"required": []string{"url"},
		},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		URL      string `json:"url"`
		MaxChars int    `json:"maxChars"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("web_fetch", err)
	}
	target, err := normalizeWebURL(input.URL)
	if err != nil {
		return toolError("web_fetch", err)
	}
	maxChars := normalizeWebMaxChars(input.MaxChars)
	client := t.client
	if client == nil {
		client = defaultWebClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return toolError("web_fetch", err)
	}
	req.Header.Set("User-Agent", "Aivo/1.0 (+https://aivo.local)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json,application/xml;q=0.9,*/*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return toolError("web_fetch", err)
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	if !isReadableWebContentType(contentType) {
		return toolError("web_fetch", fmt.Errorf("unsupported content type: %s", firstNonEmpty(contentType, "unknown")))
	}
	raw, truncatedBytes, err := readBoundedBody(resp.Body, webFetchMaxBytes)
	if err != nil {
		return toolError("web_fetch", err)
	}
	text := webResponseText(raw, contentType)
	originalChars := len(text)
	truncated := false
	if len(text) > maxChars {
		text = strings.TrimSpace(text[:maxChars]) + fmt.Sprintf("\n\n[truncated: page text exceeded %d characters]", maxChars)
		truncated = true
	}
	if truncatedBytes {
		text += fmt.Sprintf("\n\n[truncated: response body exceeded %d bytes]", webFetchMaxBytes)
		truncated = true
	}
	title := extractHTMLTitle(string(raw))
	header := fmt.Sprintf("URL: %s\nStatus: %d %s\nContent-Type: %s", resp.Request.URL.String(), resp.StatusCode, http.StatusText(resp.StatusCode), firstNonEmpty(contentType, "unknown"))
	if title != "" {
		header += "\nTitle: " + title
	}
	content := strings.TrimSpace(header + "\n\n" + text)
	return domain.ToolResult{
		Name:         "web_fetch",
		OK:           resp.StatusCode >= 200 && resp.StatusCode < 400,
		Content:      content,
		ModelContent: content,
		Structured: map[string]any{
			"url":           resp.Request.URL.String(),
			"status":        resp.StatusCode,
			"contentType":   contentType,
			"title":         title,
			"truncated":     truncated,
			"originalChars": originalChars,
		},
		Truncated:    truncated,
		OriginalSize: originalChars,
	}
}

type WebSearchTool struct {
	client    *http.Client
	searchURL string
	backend   WebSearchBackend
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{client: defaultWebClient(), searchURL: defaultWebSearchURL}
}

func (t *WebSearchTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "web_search",
		Description:          "Search the public web and return a bounded list of result titles, URLs, and snippets. Use this when you do not know the exact URL. Follow up with web_fetch on selected results when source details matter.",
		Namespace:            webNamespace,
		NamespaceDescription: webNamespaceDescription,
		Capability:           "web.search",
		RiskLevel:            "medium",
		Category:             "web",
		Toolsets:             []string{"web", "coding"},
		RequiresNetwork:      true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search query."},
				"limit": map[string]any{"type": "integer", "description": "Maximum results to return. Defaults to 5; max 10.", "minimum": 1, "maximum": webSearchMaxLimit},
			},
			"required": []string{"query"},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("web_search", err)
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError("web_search", errors.New("query is required"))
	}
	limit := input.Limit
	if limit <= 0 || limit > webSearchMaxLimit {
		limit = webSearchDefaultLimit
	}
	backend := t.backend
	if backend == nil {
		backend = NewDuckDuckGoSearchBackend(t.client, t.searchURL)
	}
	response, err := backend.Search(ctx, WebSearchRequest{Query: query, Limit: limit})
	if err != nil {
		return toolError("web_search", err)
	}
	return webSearchResponseToolResult("web_search", execCtx.ToolCallID, query, response, backend.Name())
}

func webSearchResponseToolResult(name, callID, query string, response WebSearchResponse, provider string) domain.ToolResult {
	results := response.Results
	var content strings.Builder
	content.WriteString("Search results for: " + query)
	for i, result := range results {
		content.WriteString("\n\n")
		content.WriteString(strconv.Itoa(i+1) + ". " + result.Title + "\n")
		content.WriteString(result.URL)
		if result.Snippet != "" {
			content.WriteString("\n" + result.Snippet)
		}
	}
	if len(results) == 0 {
		raw := strings.TrimSpace(response.Content)
		if raw == "" {
			return toolErrorWithCallID(name, callID, errors.New("no search results found"))
		}
		content.WriteString("\n\n")
		content.WriteString(raw)
	}
	structuredResults := make([]map[string]any, 0, len(results))
	for _, result := range results {
		structuredResults = append(structuredResults, map[string]any{"title": result.Title, "url": result.URL, "snippet": result.Snippet})
	}
	text := content.String()
	return domain.ToolResult{
		Name:         "web_search",
		CallID:       callID,
		OK:           response.Status >= 200 && response.Status < 400,
		Content:      text,
		ModelContent: text,
		Structured:   map[string]any{"query": query, "results": structuredResults, "status": response.Status, "provider": provider},
	}
}
