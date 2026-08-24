package app

import (
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

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
