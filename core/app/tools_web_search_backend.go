package app

import (
	"context"
	"net/http"
	"strings"
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
