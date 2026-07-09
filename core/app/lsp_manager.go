package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

const (
	lspMaxClients     = 4
	lspIdleTTL        = 10 * time.Minute
	lspStartupTimeout = 2500 * time.Millisecond
	lspRequestTimeout = 3 * time.Second
	lspDiagWait       = 600 * time.Millisecond
)

type boundedLSPManager struct {
	mu             sync.Mutex
	clients        map[string]*boundedLSPClient
	statuses       map[string]domain.CodeIntelligenceStatus
	maxClients     int
	idleTTL        time.Duration
	startupTimeout time.Duration
	requestTimeout time.Duration
	diagWait       time.Duration
	now            func() time.Time
}

func newBoundedLSPManager() *boundedLSPManager {
	return &boundedLSPManager{
		clients:        map[string]*boundedLSPClient{},
		statuses:       map[string]domain.CodeIntelligenceStatus{},
		maxClients:     lspMaxClients,
		idleTTL:        lspIdleTTL,
		startupTimeout: lspStartupTimeout,
		requestTimeout: lspRequestTimeout,
		diagWait:       lspDiagWait,
		now:            time.Now,
	}
}

var defaultCodeIntelligenceService domain.CodeIntelligenceService = newBoundedLSPManager()

func codeIntelligenceService() domain.CodeIntelligenceService {
	return defaultCodeIntelligenceService
}

func setCodeIntelligenceServiceForTest(service domain.CodeIntelligenceService) func() {
	previous := defaultCodeIntelligenceService
	defaultCodeIntelligenceService = service
	return func() {
		if closer, ok := defaultCodeIntelligenceService.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		defaultCodeIntelligenceService = previous
	}
}

func (m *boundedLSPManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, client := range m.clients {
		client.close()
		delete(m.clients, key)
	}
	return nil
}

func (m *boundedLSPManager) Status(ctx context.Context, workspaceRoot string) (domain.CodeIntelligenceStatus, error) {
	root, err := cleanWorkspaceRoot(workspaceRoot)
	if err != nil {
		return lspUnavailableStatus(workspaceRoot, "", "invalid workspace root", "workspace root is invalid"), err
	}
	language, _, ok := detectWorkspaceLSPLanguage(ctx, root)
	if !ok {
		return lspUnavailableStatus(root, "", "no supported Go or TypeScript/JavaScript source found", ""), nil
	}
	client, status, err := m.client(ctx, root, language)
	if err != nil {
		return status, nil
	}
	client.touch(m.now())
	return status, nil
}

func (m *boundedLSPManager) Diagnostics(ctx context.Context, workspaceRoot string, path string) ([]domain.CodeDiagnostic, domain.CodeIntelligenceStatus, error) {
	root, target, language, status, err := m.clientForPath(ctx, workspaceRoot, path)
	if err != nil || status.Status != domain.CodeIntelligenceStatusReady {
		return nil, status, err
	}
	client, _, err := m.client(ctx, root, language)
	if err != nil {
		return nil, status, nil
	}
	if err := client.openDocument(ctx, target, language, m.requestTimeout); err != nil {
		return nil, lspUnavailableStatus(root, language, "language server document open failed", err.Error()), nil
	}
	deadline := time.NewTimer(m.diagWait)
	defer deadline.Stop()
	select {
	case <-ctx.Done():
		return nil, status, ctx.Err()
	case <-deadline.C:
	}
	diagnostics := client.diagnosticsFor(target)
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		return diagnostics[i].Range.Start.Line < diagnostics[j].Range.Start.Line
	})
	return diagnostics, status, nil
}

func (m *boundedLSPManager) Symbols(ctx context.Context, workspaceRoot string, query string, path string, kind string, limit int) ([]domain.CodeSymbol, domain.CodeIntelligenceStatus, error) {
	root, err := cleanWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, lspUnavailableStatus(workspaceRoot, "", "invalid workspace root", "workspace root is invalid"), err
	}
	language := ""
	if strings.TrimSpace(path) != "" {
		target, joinErr := safeJoin(root, path)
		if joinErr != nil {
			return nil, lspUnavailableStatus(root, "", "invalid workspace path", joinErr.Error()), joinErr
		}
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
			language = lspLanguageForPath(target)
		}
	}
	if language == "" {
		detected, _, ok := detectWorkspaceLSPLanguage(ctx, root)
		if !ok {
			return nil, lspUnavailableStatus(root, "", "no supported Go or TypeScript/JavaScript source found", ""), nil
		}
		language = detected
	}
	client, status, err := m.client(ctx, root, language)
	if err != nil {
		return nil, status, nil
	}
	ctx, cancel := context.WithTimeout(ctx, m.requestTimeout)
	defer cancel()
	var raw []lspSymbolInformation
	if err := client.request(ctx, "workspace/symbol", map[string]any{"query": query}, &raw); err != nil {
		return nil, lspUnavailableStatus(root, language, "language server symbol request failed", err.Error()), nil
	}
	symbols := make([]domain.CodeSymbol, 0, len(raw))
	for _, item := range raw {
		symbol, ok := lspSymbolToDomain(root, item)
		if !ok {
			continue
		}
		if kind != "" && symbol.Kind != kind {
			continue
		}
		if strings.TrimSpace(path) != "" && !strings.HasPrefix(symbol.Path, filepath.ToSlash(strings.Trim(strings.TrimSpace(path), "/"))) {
			continue
		}
		if preview := codeSymbolPreviewLine(root, symbol); preview != "" {
			symbol.Signature = preview
		}
		symbols = append(symbols, symbol)
		if len(symbols) >= limit {
			break
		}
	}
	return symbols, status, nil
}

func (m *boundedLSPManager) Definition(ctx context.Context, workspaceRoot string, path string, position domain.SourcePosition, limit int) ([]domain.CodeLocation, domain.CodeIntelligenceStatus, error) {
	return m.locations(ctx, workspaceRoot, path, position, limit, "textDocument/definition", nil)
}

func (m *boundedLSPManager) References(ctx context.Context, workspaceRoot string, path string, position domain.SourcePosition, limit int) ([]domain.CodeLocation, domain.CodeIntelligenceStatus, error) {
	return m.locations(ctx, workspaceRoot, path, position, limit, "textDocument/references", map[string]any{"includeDeclaration": true})
}

func (m *boundedLSPManager) locations(ctx context.Context, workspaceRoot string, path string, position domain.SourcePosition, limit int, method string, contextValue map[string]any) ([]domain.CodeLocation, domain.CodeIntelligenceStatus, error) {
	root, target, language, status, err := m.clientForPath(ctx, workspaceRoot, path)
	if err != nil || status.Status != domain.CodeIntelligenceStatusReady {
		return nil, status, err
	}
	client, _, err := m.client(ctx, root, language)
	if err != nil {
		return nil, status, nil
	}
	if err := client.openDocument(ctx, target, language, m.requestTimeout); err != nil {
		return nil, lspUnavailableStatus(root, language, "language server document open failed", err.Error()), nil
	}
	params := map[string]any{
		"textDocument": map[string]any{"uri": fileURI(target)},
		"position":     map[string]any{"line": position.Line - 1, "character": position.Character},
	}
	if contextValue != nil {
		params["context"] = contextValue
	}
	ctx, cancel := context.WithTimeout(ctx, m.requestTimeout)
	defer cancel()
	var raw json.RawMessage
	if err := client.request(ctx, method, params, &raw); err != nil {
		return nil, lspUnavailableStatus(root, language, "language server location request failed", err.Error()), nil
	}
	locations := parseLSPLocations(root, raw, limit)
	return locations, status, nil
}

func (m *boundedLSPManager) clientForPath(ctx context.Context, workspaceRoot string, relPath string) (string, string, string, domain.CodeIntelligenceStatus, error) {
	root, err := cleanWorkspaceRoot(workspaceRoot)
	if err != nil {
		status := lspUnavailableStatus(workspaceRoot, "", "invalid workspace root", "workspace root is invalid")
		return "", "", "", status, err
	}
	target, err := safeJoin(root, relPath)
	if err != nil {
		status := lspUnavailableStatus(root, "", "invalid workspace path", err.Error())
		return root, "", "", status, err
	}
	language := lspLanguageForPath(target)
	if language == "" {
		return root, target, "", lspUnavailableStatus(root, "", "unsupported source language", ""), nil
	}
	_, status, err := m.client(ctx, root, language)
	if err != nil {
		return root, target, language, status, nil
	}
	return root, target, language, status, nil
}

func (m *boundedLSPManager) client(ctx context.Context, workspaceRoot string, language string) (*boundedLSPClient, domain.CodeIntelligenceStatus, error) {
	key := workspaceRoot + "\x00" + language
	now := m.now()
	m.mu.Lock()
	m.evictIdleLocked(now)
	if client := m.clients[key]; client != nil && client.isRunning() {
		client.touch(now)
		status := lspReadyStatus(workspaceRoot, language, client.source, client.startedAt, now)
		m.statuses[key] = status
		m.mu.Unlock()
		return client, status, nil
	}
	command, args, source, ok := lspCommandForLanguage(language)
	if !ok {
		status := lspUnavailableStatus(workspaceRoot, language, "unsupported source language", "")
		m.statuses[key] = status
		m.mu.Unlock()
		return nil, status, errors.New("unsupported source language")
	}
	if _, err := exec.LookPath(command); err != nil {
		status := lspUnavailableStatus(workspaceRoot, language, fmt.Sprintf("%s not found on PATH", command), "")
		m.statuses[key] = status
		m.mu.Unlock()
		return nil, status, err
	}
	if len(m.clients) >= m.maxClients {
		m.closeOldestLocked()
	}
	m.mu.Unlock()

	startCtx, cancel := context.WithTimeout(ctx, m.startupTimeout)
	defer cancel()
	client, err := startBoundedLSPClient(startCtx, workspaceRoot, language, command, args, source)
	if err != nil {
		status := lspUnavailableStatus(workspaceRoot, language, "language server startup failed", err.Error())
		m.mu.Lock()
		m.statuses[key] = status
		m.mu.Unlock()
		return nil, status, err
	}
	status := lspReadyStatus(workspaceRoot, language, source, client.startedAt, m.now())
	m.mu.Lock()
	if old := m.clients[key]; old != nil {
		old.close()
	}
	m.clients[key] = client
	m.statuses[key] = status
	m.mu.Unlock()
	return client, status, nil
}

func (m *boundedLSPManager) evictIdleLocked(now time.Time) {
	for key, client := range m.clients {
		if now.Sub(client.lastUsed()) > m.idleTTL || !client.isRunning() {
			client.close()
			delete(m.clients, key)
		}
	}
}

func (m *boundedLSPManager) closeOldestLocked() {
	oldestKey := ""
	var oldest time.Time
	for key, client := range m.clients {
		used := client.lastUsed()
		if oldestKey == "" || used.Before(oldest) {
			oldestKey = key
			oldest = used
		}
	}
	if oldestKey != "" {
		m.clients[oldestKey].close()
		delete(m.clients, oldestKey)
	}
}
