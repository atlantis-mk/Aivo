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
	selection, err := m.clientForPath(ctx, workspaceRoot, path)
	if err != nil || selection.status.Status != domain.CodeIntelligenceStatusReady {
		return nil, selection.status, err
	}
	client, _, err := m.clientWithDefinition(ctx, selection.serverRoot, selection.language, selection.resolved)
	if err != nil {
		return nil, selection.status, nil
	}
	if err := client.openDocument(ctx, selection.target, selection.language, m.requestTimeout); err != nil {
		return nil, lspUnavailableStatus(selection.workspaceRoot, selection.language, "language server document open failed", err.Error()), nil
	}
	deadline := time.NewTimer(m.diagWait)
	defer deadline.Stop()
	select {
	case <-ctx.Done():
		return nil, selection.status, ctx.Err()
	case <-deadline.C:
	}
	diagnostics := client.diagnosticsFor(selection.target)
	for index := range diagnostics {
		absolute := filepath.Join(selection.serverRoot, filepath.FromSlash(diagnostics[index].Path))
		if rel, relErr := filepath.Rel(selection.workspaceRoot, absolute); relErr == nil && !strings.HasPrefix(rel, "..") {
			diagnostics[index].Path = filepath.ToSlash(rel)
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		return diagnostics[i].Range.Start.Line < diagnostics[j].Range.Start.Line
	})
	return diagnostics, selection.status, nil
}

func (m *boundedLSPManager) Symbols(ctx context.Context, workspaceRoot string, query string, path string, kind string, limit int) ([]domain.CodeSymbol, domain.CodeIntelligenceStatus, error) {
	root, err := cleanWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, lspUnavailableStatus(workspaceRoot, "", "invalid workspace root", "workspace root is invalid"), err
	}
	language := ""
	serverRoot := root
	var resolved resolvedLSPDefinition
	if strings.TrimSpace(path) != "" {
		target, joinErr := safeJoin(root, path)
		if joinErr != nil {
			return nil, lspUnavailableStatus(root, "", "invalid workspace path", joinErr.Error()), joinErr
		}
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
			if selected, ok := resolveLSPDefinitionForPath(root, target); ok {
				resolved = selected
				language = languageIDForLSPDefinition(selected.Definition, target)
				serverRoot = nearestLSPRoot(root, target, selected.Definition.RootMarkers)
			}
		}
	}
	if language == "" {
		detected, _, ok := detectWorkspaceLSPLanguage(ctx, root)
		if !ok {
			return nil, lspUnavailableStatus(root, "", "no supported Go or TypeScript/JavaScript source found", ""), nil
		}
		language = detected
	}
	if resolved.Name == "" {
		resolved, _ = resolveLSPDefinitionForLanguage(root, language)
	}
	client, status, err := m.clientWithDefinition(ctx, serverRoot, language, resolved)
	status.WorkspaceRoot = root
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
	selection, err := m.clientForPath(ctx, workspaceRoot, path)
	if err != nil || selection.status.Status != domain.CodeIntelligenceStatusReady {
		return nil, selection.status, err
	}
	client, _, err := m.clientWithDefinition(ctx, selection.serverRoot, selection.language, selection.resolved)
	if err != nil {
		return nil, selection.status, nil
	}
	if err := client.openDocument(ctx, selection.target, selection.language, m.requestTimeout); err != nil {
		return nil, lspUnavailableStatus(selection.workspaceRoot, selection.language, "language server document open failed", err.Error()), nil
	}
	params := map[string]any{
		"textDocument": map[string]any{"uri": fileURI(selection.target)},
		"position":     map[string]any{"line": position.Line - 1, "character": position.Character},
	}
	if contextValue != nil {
		params["context"] = contextValue
	}
	ctx, cancel := context.WithTimeout(ctx, m.requestTimeout)
	defer cancel()
	var raw json.RawMessage
	if err := client.request(ctx, method, params, &raw); err != nil {
		return nil, lspUnavailableStatus(selection.workspaceRoot, selection.language, "language server location request failed", err.Error()), nil
	}
	locations := parseLSPLocations(selection.workspaceRoot, raw, limit)
	return locations, selection.status, nil
}

type lspClientSelection struct {
	workspaceRoot string
	serverRoot    string
	target        string
	language      string
	resolved      resolvedLSPDefinition
	status        domain.CodeIntelligenceStatus
}

func (m *boundedLSPManager) clientForPath(ctx context.Context, workspaceRoot string, relPath string) (lspClientSelection, error) {
	root, err := cleanWorkspaceRoot(workspaceRoot)
	if err != nil {
		status := lspUnavailableStatus(workspaceRoot, "", "invalid workspace root", "workspace root is invalid")
		return lspClientSelection{status: status}, err
	}
	target, err := safeJoin(root, relPath)
	if err != nil {
		status := lspUnavailableStatus(root, "", "invalid workspace path", err.Error())
		return lspClientSelection{workspaceRoot: root, status: status}, err
	}
	resolved, ok := resolveLSPDefinitionForPath(root, target)
	if !ok {
		return lspClientSelection{workspaceRoot: root, target: target, status: lspUnavailableStatus(root, "", "unsupported source language", "")}, nil
	}
	language := languageIDForLSPDefinition(resolved.Definition, target)
	if language == "" {
		return lspClientSelection{workspaceRoot: root, target: target, status: lspUnavailableStatus(root, "", "unsupported source language", "")}, nil
	}
	serverRoot := nearestLSPRoot(root, target, resolved.Definition.RootMarkers)
	_, status, err := m.clientWithDefinition(ctx, serverRoot, language, resolved)
	status.WorkspaceRoot = root
	if err != nil {
		return lspClientSelection{workspaceRoot: root, serverRoot: serverRoot, target: target, language: language, resolved: resolved, status: status}, nil
	}
	return lspClientSelection{workspaceRoot: root, serverRoot: serverRoot, target: target, language: language, resolved: resolved, status: status}, nil
}

func (m *boundedLSPManager) client(ctx context.Context, workspaceRoot string, language string) (*boundedLSPClient, domain.CodeIntelligenceStatus, error) {
	resolved, ok := resolveLSPDefinitionForLanguage(workspaceRoot, language)
	if !ok {
		status := lspUnavailableStatus(workspaceRoot, language, "unsupported source language", "")
		return nil, status, errors.New("unsupported source language")
	}
	return m.clientWithDefinition(ctx, workspaceRoot, language, resolved)
}

func (m *boundedLSPManager) clientWithDefinition(ctx context.Context, workspaceRoot string, language string, resolved resolvedLSPDefinition) (*boundedLSPClient, domain.CodeIntelligenceStatus, error) {
	key := workspaceRoot + "\x00" + resolved.Name + "\x00" + language
	now := m.now()
	m.mu.Lock()
	m.evictIdleLocked(now)
	if client := m.clients[key]; client != nil && client.isRunning() && client.revision == resolved.Revision {
		client.touch(now)
		status := lspReadyStatus(workspaceRoot, language, client.source, client.startedAt, now)
		m.statuses[key] = status
		m.mu.Unlock()
		return client, status, nil
	}
	if client := m.clients[key]; client != nil {
		client.close()
		delete(m.clients, key)
	}
	command, source := resolved.Definition.Command, resolved.Name
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

	startupTimeout := m.startupTimeout
	if resolved.Definition.TimeoutSeconds > 0 {
		startupTimeout = time.Duration(resolved.Definition.TimeoutSeconds) * time.Second
	}
	startCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	client, err := startBoundedLSPClientWithDefinition(startCtx, workspaceRoot, language, resolved)
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

func nearestLSPRoot(workspaceRoot string, target string, markers []string) string {
	if root, ok := nearestLSPRootMatch(workspaceRoot, target, markers); ok {
		return root
	}
	return canonicalLSPPath(workspaceRoot)
}

func nearestLSPRootMatch(workspaceRoot string, target string, markers []string) (string, bool) {
	workspaceRoot = canonicalLSPPath(workspaceRoot)
	current := canonicalLSPPath(filepath.Dir(target))
	if !pathWithinRoot(workspaceRoot, current) {
		return workspaceRoot, false
	}
	for {
		for _, marker := range markers {
			marker = strings.TrimSpace(marker)
			if marker == "" {
				continue
			}
			candidate := filepath.Join(current, marker)
			if strings.ContainsAny(marker, "*?[") {
				if matches, _ := filepath.Glob(candidate); len(matches) > 0 {
					return current, true
				}
			} else if _, err := os.Stat(candidate); err == nil {
				return current, true
			}
		}
		if current == workspaceRoot {
			break
		}
		parent := filepath.Dir(current)
		if parent == current || !pathWithinRoot(workspaceRoot, parent) {
			break
		}
		current = parent
	}
	return workspaceRoot, false
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
