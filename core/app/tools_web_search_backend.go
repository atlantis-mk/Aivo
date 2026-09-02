package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type WebSearchBackend interface {
	Name() string
	Search(ctx context.Context, request WebSearchRequest) (WebSearchResponse, error)
}

type WebSearchRequest struct {
	Query string
	Limit int
}

type WebSearchResponse struct {
	Results []webSearchResult
	Status  int
	Content string
}

type DuckDuckGoSearchBackend struct {
	client    *http.Client
	searchURL string
}

func NewDuckDuckGoSearchBackend(client *http.Client, searchURL string) *DuckDuckGoSearchBackend {
	if client == nil {
		client = defaultWebClient()
	}
	if strings.TrimSpace(searchURL) == "" {
		searchURL = defaultWebSearchURL
	}
	return &DuckDuckGoSearchBackend{client: client, searchURL: searchURL}
}

func (b *DuckDuckGoSearchBackend) Name() string {
	return "duckduckgo"
}

func (b *DuckDuckGoSearchBackend) Search(ctx context.Context, request WebSearchRequest) (WebSearchResponse, error) {
	endpoint, err := buildSearchURL(b.searchURL, request.Query)
	if err != nil {
		return WebSearchResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return WebSearchResponse{}, err
	}
	req.Header.Set("User-Agent", "Aivo/1.0 (+https://aivo.local)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.1")
	resp, err := b.client.Do(req)
	if err != nil {
		return WebSearchResponse{}, err
	}
	defer resp.Body.Close()
	raw, _, err := readBoundedBody(resp.Body, webFetchMaxBytes)
	if err != nil {
		return WebSearchResponse{}, err
	}
	return WebSearchResponse{Results: parseSearchResults(string(raw), request.Limit), Status: resp.StatusCode}, nil
}

const (
	defaultParallelSearchMCPURL   = "https://search.parallel.ai/mcp"
	parallelSearchMaxResponseSize = 2 * 1024 * 1024
)

var parallelSearchHTTPClient = &http.Client{Timeout: 30 * time.Second}

type ParallelSearchBackend struct {
	client *http.Client
	mcpURL string
}

func NewParallelSearchBackend(client *http.Client, mcpURL string) *ParallelSearchBackend {
	if client == nil {
		client = parallelSearchHTTPClient
	}
	if strings.TrimSpace(mcpURL) == "" {
		mcpURL = defaultParallelSearchMCPURL
	}
	return &ParallelSearchBackend{client: client, mcpURL: mcpURL}
}

func (b *ParallelSearchBackend) Name() string {
	return "parallel"
}

func (b *ParallelSearchBackend) Search(ctx context.Context, request WebSearchRequest) (WebSearchResponse, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return WebSearchResponse{}, errors.New("query is required")
	}
	limit := request.Limit
	if limit <= 0 || limit > webSearchMaxLimit {
		limit = webSearchDefaultLimit
	}
	target, err := normalizeWebURL(b.mcpURL)
	if err != nil {
		return WebSearchResponse{}, err
	}
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      "aivo-web-search",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "web_search",
			"arguments": map[string]any{
				"objective":      query,
				"search_queries": []string{query},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return WebSearchResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(raw))
	if err != nil {
		return WebSearchResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", "Aivo/1.0 (+https://aivo.local)")
	if key := strings.TrimSpace(os.Getenv("PARALLEL_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return WebSearchResponse{}, err
	}
	defer resp.Body.Close()
	responseRaw, truncated, err := readBoundedBody(resp.Body, parallelSearchMaxResponseSize)
	if err != nil {
		return WebSearchResponse{}, err
	}
	if truncated {
		return WebSearchResponse{}, errors.New("Parallel search response exceeded the safe limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WebSearchResponse{}, providerHTTPError(resp.StatusCode, resp.Status, string(responseRaw))
	}
	content, err := parseParallelSearchMCPContent(responseRaw)
	if err != nil {
		return WebSearchResponse{}, err
	}
	text := strings.TrimSpace(content)
	results := parseParallelSearchResults(text, limit)
	return WebSearchResponse{Results: results, Status: resp.StatusCode, Content: text}, nil
}

func parseParallelSearchMCPContent(raw []byte) (string, error) {
	payloads := parallelSearchJSONPayloads(raw)
	if len(payloads) == 0 {
		payloads = [][]byte{bytes.TrimSpace(raw)}
	}
	var parseErr error
	for _, payload := range payloads {
		content, err := parseParallelSearchJSONRPCPayload(payload)
		if err == nil {
			return content, nil
		}
		parseErr = err
	}
	if parseErr != nil {
		return "", parseErr
	}
	return "", errors.New("Parallel search response could not be parsed")
}

func parallelSearchJSONPayloads(raw []byte) [][]byte {
	text := strings.TrimSpace(string(raw))
	if text == "" || !strings.Contains(text, "data:") {
		return nil
	}
	var payloads [][]byte
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		payloads = append(payloads, []byte(data))
	}
	return payloads
}

func parseParallelSearchJSONRPCPayload(raw []byte) (string, error) {
	var envelope struct {
		Result *struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", err
	}
	if envelope.Error != nil {
		return "", fmt.Errorf("Parallel search failed: %s", firstNonEmpty(envelope.Error.Message, strconv.Itoa(envelope.Error.Code)))
	}
	if envelope.Result == nil {
		return "", errors.New("Parallel search response did not include a result")
	}
	parts := make([]string, 0, len(envelope.Result.Content))
	for _, item := range envelope.Result.Content {
		if strings.ToLower(strings.TrimSpace(item.Type)) == "text" && strings.TrimSpace(item.Text) != "" {
			parts = append(parts, strings.TrimSpace(item.Text))
		}
	}
	text := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if envelope.Result.IsError {
		return "", fmt.Errorf("Parallel search failed: %s", firstNonEmpty(text, "remote tool returned an error"))
	}
	if text == "" {
		return "", errors.New("Parallel search returned no output")
	}
	return text, nil
}

func parseParallelSearchResults(text string, limit int) []webSearchResult {
	if limit <= 0 || limit > webSearchMaxLimit {
		limit = webSearchDefaultLimit
	}
	var decoded any
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &decoded); err != nil {
		return nil
	}
	results := make([]webSearchResult, 0, limit)
	collectParallelSearchResults(decoded, &results, limit, map[string]bool{})
	return results
}

func collectParallelSearchResults(value any, results *[]webSearchResult, limit int, seen map[string]bool) {
	if len(*results) >= limit {
		return
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			collectParallelSearchResults(item, results, limit, seen)
			if len(*results) >= limit {
				return
			}
		}
	case map[string]any:
		if result := parallelSearchResultFromMap(typed); result.URL != "" && !seen[result.URL] {
			seen[result.URL] = true
			*results = append(*results, result)
			return
		}
		for _, key := range []string{"results", "search_results", "organic", "items", "data"} {
			if child, ok := typed[key]; ok {
				collectParallelSearchResults(child, results, limit, seen)
			}
			if len(*results) >= limit {
				return
			}
		}
	}
}

func parallelSearchResultFromMap(item map[string]any) webSearchResult {
	url := strings.TrimSpace(firstStringAny(item, "url", "link", "href"))
	if normalized := normalizeSearchResultURL(url); normalized != "" {
		url = normalized
	}
	if url == "" {
		return webSearchResult{}
	}
	title := strings.TrimSpace(firstStringAny(item, "title", "name", "headline"))
	if title == "" {
		title = url
	}
	snippet := strings.TrimSpace(firstStringAny(item, "snippet", "description", "excerpt", "text", "content"))
	if snippet == "" {
		if excerpts, ok := item["excerpts"].([]any); ok {
			values := make([]string, 0, min(2, len(excerpts)))
			for _, excerpt := range excerpts {
				if text, ok := excerpt.(string); ok && strings.TrimSpace(text) != "" {
					values = append(values, strings.TrimSpace(text))
				}
				if len(values) >= 2 {
					break
				}
			}
			snippet = strings.Join(values, " ")
		}
	}
	return webSearchResult{Title: bounded(title, 500), URL: url, Snippet: bounded(snippet, 1000)}
}

func firstStringAny(values map[string]any, keys ...string) string {
	for _, key := range keys {
		switch typed := values[key].(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		case fmt.Stringer:
			if text := strings.TrimSpace(typed.String()); text != "" {
				return text
			}
		}
	}
	return ""
}
