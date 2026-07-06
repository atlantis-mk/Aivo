package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	results := response.Results
	if len(results) == 0 {
		return toolError("web_search", errors.New("no search results found"))
	}
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
	structuredResults := make([]map[string]any, 0, len(results))
	for _, result := range results {
		structuredResults = append(structuredResults, map[string]any{"title": result.Title, "url": result.URL, "snippet": result.Snippet})
	}
	text := content.String()
	return domain.ToolResult{
		Name:         "web_search",
		OK:           response.Status >= 200 && response.Status < 400,
		Content:      text,
		ModelContent: text,
		Structured:   map[string]any{"query": query, "results": structuredResults, "status": response.Status, "provider": backend.Name()},
	}
}

type webSearchResult struct {
	Title   string
	URL     string
	Snippet string
}

func defaultWebClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}

func normalizeWebURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil {
		return "", errors.New("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("url must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("url host is required")
	}
	return parsed.String(), nil
}

func normalizeWebMaxChars(value int) int {
	if value <= 0 {
		return webFetchDefaultMaxChars
	}
	if value > webFetchMaxChars {
		return webFetchMaxChars
	}
	if value < 1000 {
		return 1000
	}
	return value
}

func isReadableWebContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if contentType == "" {
		return true
	}
	return strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/xhtml+xml" ||
		strings.HasSuffix(contentType, "+json") ||
		strings.HasSuffix(contentType, "+xml")
}

func readBoundedBody(body io.Reader, maxBytes int64) ([]byte, bool, error) {
	limited := io.LimitReader(body, maxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(raw)) > maxBytes {
		return raw[:maxBytes], true, nil
	}
	return raw, false, nil
}

func webResponseText(raw []byte, contentType string) string {
	text := string(raw)
	if strings.Contains(strings.ToLower(contentType), "html") || looksLikeHTML(text) {
		text = extractHTMLText(text)
	}
	return normalizeWebWhitespace(html.UnescapeString(text))
}

func looksLikeHTML(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") || strings.Contains(lower, "<body")
}

func extractHTMLTitle(text string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return normalizeWebWhitespace(html.UnescapeString(stripHTMLTags(match[1])))
}

func extractHTMLText(text string) string {
	replacements := []struct {
		re   *regexp.Regexp
		with string
	}{
		{regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`), " "},
		{regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`), " "},
		{regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`), " "},
		{regexp.MustCompile(`(?is)<(br|p|div|li|tr|h[1-6])[^>]*>`), "\n"},
	}
	for _, item := range replacements {
		text = item.re.ReplaceAllString(text, item.with)
	}
	return stripHTMLTags(text)
}

func stripHTMLTags(text string) string {
	return regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(text, " ")
}

func normalizeWebWhitespace(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		out = append(out, strings.Join(fields, " "))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func buildSearchURL(base string, query string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	values := parsed.Query()
	if _, ok := values["q"]; !ok {
		values.Set("q", query)
	}
	parsed.RawQuery = values.Encode()
	return normalizeWebURL(parsed.String())
}

func parseSearchResults(raw string, limit int) []webSearchResult {
	if limit <= 0 || limit > webSearchMaxLimit {
		limit = webSearchDefaultLimit
	}
	linkRe := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	matches := linkRe.FindAllStringSubmatch(raw, -1)
	results := make([]webSearchResult, 0, limit)
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		link := normalizeSearchResultURL(html.UnescapeString(match[1]))
		title := normalizeWebWhitespace(html.UnescapeString(stripHTMLTags(match[2])))
		if link == "" || title == "" || seen[link] || isSearchChromeLink(link, title) {
			continue
		}
		seen[link] = true
		results = append(results, webSearchResult{Title: title, URL: link})
		if len(results) >= limit {
			break
		}
	}
	snippets := parseSearchSnippets(raw)
	for i := range results {
		if i < len(snippets) {
			results[i].Snippet = snippets[i]
		}
	}
	return results
}

func parseSearchSnippets(raw string) []string {
	re := regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>|<div[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</div>`)
	matches := re.FindAllStringSubmatch(raw, -1)
	snippets := make([]string, 0, len(matches))
	for _, match := range matches {
		value := ""
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				value = candidate
				break
			}
		}
		value = normalizeWebWhitespace(html.UnescapeString(stripHTMLTags(value)))
		if value != "" {
			snippets = append(snippets, value)
		}
	}
	return snippets
}

func normalizeSearchResultURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.Path == "/l/" {
		if uddg := parsed.Query().Get("uddg"); uddg != "" {
			if decoded, err := url.QueryUnescape(uddg); err == nil {
				value = decoded
			}
		}
	}
	if strings.HasPrefix(value, "//") {
		value = "https:" + value
	}
	if strings.HasPrefix(value, "/") {
		return ""
	}
	if normalized, err := normalizeWebURL(value); err == nil {
		return normalized
	}
	return ""
}

func isSearchChromeLink(link string, title string) bool {
	lowerTitle := strings.ToLower(strings.TrimSpace(title))
	if lowerTitle == "next" || lowerTitle == "previous" || lowerTitle == "images" || lowerTitle == "videos" || lowerTitle == "news" {
		return true
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "duckduckgo.com" || host == "www.duckduckgo.com"
}
