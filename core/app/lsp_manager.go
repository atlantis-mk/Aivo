package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

type boundedLSPClient struct {
	root      string
	language  string
	source    string
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	cancel    context.CancelFunc
	startedAt time.Time

	mu          sync.Mutex
	nextID      int
	pending     map[int]chan lspResponse
	opened      map[string]bool
	diagnostics map[string][]domain.CodeDiagnostic
	lastUsedAt  time.Time
	running     bool
	stderr      bytes.Buffer
}

func startBoundedLSPClient(ctx context.Context, workspaceRoot string, language string, command string, args []string, source string) (*boundedLSPClient, error) {
	processCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processCtx, command, args...)
	cmd.Dir = workspaceRoot
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	client := &boundedLSPClient{
		root: workspaceRoot, language: language, source: source, cmd: cmd, stdin: stdin, cancel: cancel,
		startedAt: time.Now(), pending: map[int]chan lspResponse{}, opened: map[string]bool{}, diagnostics: map[string][]domain.CodeDiagnostic{},
		lastUsedAt: time.Now(), running: true,
	}
	go client.readLoop(stdout)
	go client.stderrLoop(stderr)
	go func() {
		_ = cmd.Wait()
		client.mu.Lock()
		client.running = false
		for id, ch := range client.pending {
			delete(client.pending, id)
			close(ch)
		}
		client.mu.Unlock()
	}()
	initCtx, cancelInit := context.WithTimeout(ctx, lspStartupTimeout)
	defer cancelInit()
	var result map[string]any
	if err := client.request(initCtx, "initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   fileURI(workspaceRoot),
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"publishDiagnostics": map[string]any{},
				"definition":         map[string]any{},
				"references":         map[string]any{},
				"documentSymbol":     map[string]any{},
			},
			"workspace": map[string]any{"symbol": map[string]any{}},
		},
	}, &result); err != nil {
		client.close()
		return nil, err
	}
	_ = client.notify("initialized", map[string]any{})
	return client, nil
}

func (c *boundedLSPClient) touch(now time.Time) {
	c.mu.Lock()
	c.lastUsedAt = now
	c.mu.Unlock()
}

func (c *boundedLSPClient) lastUsed() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastUsedAt
}

func (c *boundedLSPClient) isRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *boundedLSPClient) close() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	pending := c.pending
	c.pending = map[int]chan lspResponse{}
	c.mu.Unlock()
	for _, ch := range pending {
		close(ch)
	}
	_ = c.stdin.Close()
	c.cancel()
}

func (c *boundedLSPClient) request(ctx context.Context, method string, params any, result any) error {
	id, responseCh, err := c.sendRequest(method, params)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	case response, ok := <-responseCh:
		if !ok {
			return errors.New("language server stopped")
		}
		if response.Error != nil {
			return errors.New(response.Error.Message)
		}
		if result == nil || len(response.Result) == 0 {
			return nil
		}
		return json.Unmarshal(response.Result, result)
	}
}

func (c *boundedLSPClient) sendRequest(method string, params any) (int, chan lspResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return 0, nil, errors.New("language server is not running")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan lspResponse, 1)
	c.pending[id] = ch
	err := c.writeLocked(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		delete(c.pending, id)
	}
	return id, ch, err
}

func (c *boundedLSPClient) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return errors.New("language server is not running")
	}
	return c.writeLocked(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *boundedLSPClient) writeLocked(message any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(raw))
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err = c.stdin.Write(raw)
	return err
}

func (c *boundedLSPClient) openDocument(ctx context.Context, path string, language string, timeout time.Duration) error {
	uri := fileURI(path)
	c.mu.Lock()
	if c.opened[uri] {
		c.mu.Unlock()
		return nil
	}
	c.opened[uri] = true
	c.mu.Unlock()
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	params := map[string]any{"textDocument": map[string]any{
		"uri":        uri,
		"languageId": languageID(language, path),
		"version":    1,
		"text":       string(raw),
	}}
	done := make(chan error, 1)
	go func() { done <- c.notify("textDocument/didOpen", params) }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	case <-timer.C:
		return context.DeadlineExceeded
	}
}

func (c *boundedLSPClient) diagnosticsFor(path string) []domain.CodeDiagnostic {
	uri := fileURI(path)
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]domain.CodeDiagnostic(nil), c.diagnostics[uri]...)
	return out
}

func (c *boundedLSPClient) readLoop(stdout io.Reader) {
	reader := bufio.NewReader(stdout)
	for {
		contentLength, err := readLSPContentLength(reader)
		if err != nil {
			return
		}
		raw := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, raw); err != nil {
			return
		}
		var message lspMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			continue
		}
		if message.ID != nil {
			c.mu.Lock()
			ch := c.pending[*message.ID]
			delete(c.pending, *message.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- lspResponse{Result: message.Result, Error: message.Error}
				close(ch)
			}
			continue
		}
		if message.Method == "textDocument/publishDiagnostics" {
			c.recordDiagnostics(message.Params)
		}
	}
}

func (c *boundedLSPClient) stderrLoop(stderr io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			c.mu.Lock()
			if c.stderr.Len() < 8192 {
				_, _ = c.stderr.Write(buf[:n])
			}
			c.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (c *boundedLSPClient) recordDiagnostics(raw json.RawMessage) {
	var params struct {
		URI         string          `json:"uri"`
		Diagnostics []lspDiagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return
	}
	path, ok := pathFromFileURI(params.URI)
	if !ok {
		return
	}
	rel := filepath.ToSlash(mustRel(c.root, path))
	out := make([]domain.CodeDiagnostic, 0, len(params.Diagnostics))
	for _, diagnostic := range params.Diagnostics {
		out = append(out, domain.CodeDiagnostic{
			Path:     rel,
			Range:    diagnostic.Range.toDomain(),
			Severity: lspSeverity(diagnostic.Severity),
			Message:  diagnostic.Message,
			Source:   nonEmpty(diagnostic.Source, c.source),
			Code:     diagnostic.CodeString(),
		})
	}
	c.mu.Lock()
	c.diagnostics[params.URI] = out
	c.mu.Unlock()
}

func readLSPContentLength(reader *bufio.Reader) (int, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			value, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return 0, err
			}
			contentLength = value
		}
	}
	if contentLength < 0 {
		return 0, errors.New("missing Content-Length")
	}
	return contentLength, nil
}

type lspMessage struct {
	ID     *int            `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *lspRPCError    `json:"error,omitempty"`
}

type lspResponse struct {
	Result json.RawMessage
	Error  *lspRPCError
}

type lspRPCError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
}

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

func (r lspRange) toDomain() domain.SourceRange {
	return domain.SourceRange{
		Start: domain.SourcePosition{Line: r.Start.Line + 1, Character: r.Start.Character},
		End:   domain.SourcePosition{Line: r.End.Line + 1, Character: r.End.Character},
	}
}

type lspDiagnostic struct {
	Range    lspRange        `json:"range"`
	Severity int             `json:"severity,omitempty"`
	Message  string          `json:"message"`
	Source   string          `json:"source,omitempty"`
	Code     json.RawMessage `json:"code,omitempty"`
}

func (d lspDiagnostic) CodeString() string {
	if len(d.Code) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(d.Code, &text); err == nil {
		return text
	}
	var number int
	if err := json.Unmarshal(d.Code, &number); err == nil {
		return strconv.Itoa(number)
	}
	return ""
}

type lspLocation struct {
	URI    string   `json:"uri"`
	Range  lspRange `json:"range"`
	Target string   `json:"targetUri,omitempty"`
}

type lspSymbolInformation struct {
	Name     string `json:"name"`
	Kind     int    `json:"kind"`
	Location struct {
		URI   string   `json:"uri"`
		Range lspRange `json:"range"`
	} `json:"location"`
}

func parseLSPLocations(workspaceRoot string, raw json.RawMessage, limit int) []domain.CodeLocation {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var many []lspLocation
	if err := json.Unmarshal(raw, &many); err != nil {
		var one lspLocation
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil
		}
		many = []lspLocation{one}
	}
	out := make([]domain.CodeLocation, 0, len(many))
	for _, location := range many {
		uri := nonEmpty(location.URI, location.Target)
		path, ok := pathFromFileURI(uri)
		if !ok {
			continue
		}
		rel, err := filepath.Rel(workspaceRoot, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		domainLocation := domain.CodeLocation{
			Path:     filepath.ToSlash(rel),
			Range:    location.Range.toDomain(),
			Language: symbolLanguage(strings.ToLower(filepath.Ext(path))),
			Preview:  previewLine(path, location.Range.Start.Line+1),
			Source:   "lsp",
		}
		out = append(out, domainLocation)
		if len(out) >= limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Range.Start.Line < out[j].Range.Start.Line
	})
	return out
}

func lspSymbolToDomain(workspaceRoot string, item lspSymbolInformation) (domain.CodeSymbol, bool) {
	path, ok := pathFromFileURI(item.Location.URI)
	if !ok {
		return domain.CodeSymbol{}, false
	}
	rel, err := filepath.Rel(workspaceRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return domain.CodeSymbol{}, false
	}
	return domain.CodeSymbol{
		Name:     item.Name,
		Kind:     lspSymbolKind(item.Kind),
		Path:     filepath.ToSlash(rel),
		Range:    item.Location.Range.toDomain(),
		Language: symbolLanguage(strings.ToLower(filepath.Ext(path))),
		Source:   "lsp",
	}, true
}

func codeSymbolPreviewLine(workspaceRoot string, symbol domain.CodeSymbol) string {
	return previewLine(filepath.Join(workspaceRoot, filepath.FromSlash(symbol.Path)), symbol.Range.Start.Line)
}

func previewLine(path string, line int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo == line {
			return strings.TrimSpace(scanner.Text())
		}
	}
	return ""
}

func cleanWorkspaceRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("workspace root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("workspace root must be a directory")
	}
	return abs, nil
}

func detectWorkspaceLSPLanguage(ctx context.Context, root string) (string, string, bool) {
	if fileExists(filepath.Join(root, "go.mod")) || hasWorkspaceFile(ctx, root, ".go") {
		return "go", "", true
	}
	if fileExists(filepath.Join(root, "tsconfig.json")) || fileExists(filepath.Join(root, "jsconfig.json")) || fileExists(filepath.Join(root, "package.json")) || hasWorkspaceFile(ctx, root, ".ts", ".tsx", ".js", ".jsx") {
		return "typescript", "", true
	}
	return "", "", false
}

func hasWorkspaceFile(ctx context.Context, root string, exts ...string) bool {
	extSet := map[string]bool{}
	for _, ext := range exts {
		extSet[ext] = true
	}
	ignore := loadWorkspaceIgnore(ctx, root)
	found := false
	_ = filepath.WalkDir(root, func(current string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if shouldSkipWorkspaceEntry(root, current, entry, ignore) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel := filepath.ToSlash(mustRel(root, current))
		if isSensitiveRelPath(rel) {
			return nil
		}
		if extSet[strings.ToLower(filepath.Ext(current))] {
			found = true
			return errStopWalk
		}
		return ctx.Err()
	})
	return found
}

func lspLanguageForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	default:
		return ""
	}
}

func languageID(language string, path string) string {
	switch language {
	case "go":
		return "go"
	case "javascript":
		if strings.EqualFold(filepath.Ext(path), ".jsx") {
			return "javascriptreact"
		}
		return "javascript"
	default:
		if strings.EqualFold(filepath.Ext(path), ".tsx") {
			return "typescriptreact"
		}
		return "typescript"
	}
}

func lspCommandForLanguage(language string) (string, []string, string, bool) {
	switch language {
	case "go":
		return "gopls", nil, "gopls", true
	case "typescript", "javascript":
		return "typescript-language-server", []string{"--stdio"}, "typescript-language-server", true
	default:
		return "", nil, "", false
	}
}

func lspReadyStatus(root string, language string, source string, startedAt time.Time, now time.Time) domain.CodeIntelligenceStatus {
	return domain.CodeIntelligenceStatus{
		WorkspaceRoot: root,
		Language:      language,
		Status:        domain.CodeIntelligenceStatusReady,
		Source:        source,
		Message:       fmt.Sprintf("language server ready; startup %dms", int(now.Sub(startedAt).Milliseconds())),
		TimeUpdated:   now.UTC().Format(time.RFC3339Nano),
	}
}

func lspUnavailableStatus(root string, language string, message string, detail string) domain.CodeIntelligenceStatus {
	if detail != "" {
		message = message + ": " + bounded(detail, 180)
	}
	return domain.CodeIntelligenceStatus{
		WorkspaceRoot: root,
		Language:      language,
		Status:        domain.CodeIntelligenceStatusUnavailable,
		Source:        "lsp",
		Message:       message,
		TimeUpdated:   time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func lspSeverity(severity int) string {
	switch severity {
	case 1:
		return domain.DiagnosticSeverityError
	case 2:
		return domain.DiagnosticSeverityWarning
	case 3:
		return domain.DiagnosticSeverityInformation
	case 4:
		return domain.DiagnosticSeverityHint
	default:
		return domain.DiagnosticSeverityInformation
	}
}

func lspSymbolKind(kind int) string {
	switch kind {
	case 2:
		return "module"
	case 3, 4, 5:
		return "class"
	case 6:
		return "method"
	case 7:
		return "property"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 12, 13:
		return "function"
	case 14:
		return "variable"
	case 15:
		return "constant"
	case 23:
		return "type"
	default:
		return "symbol"
	}
}

func fileURI(path string) string {
	abs, _ := filepath.Abs(path)
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

func pathFromFileURI(uri string) (string, bool) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return "", false
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", false
	}
	return filepath.FromSlash(path), true
}

func nonEmpty(first string, second string) string {
	if first != "" {
		return first
	}
	return second
}
