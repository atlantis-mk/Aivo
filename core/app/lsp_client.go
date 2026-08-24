package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type boundedLSPClient struct {
	root      string
	language  string
	source    string
	revision  string
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
	return startBoundedLSPClientWithDefinition(ctx, workspaceRoot, language, resolvedLSPDefinition{
		Name: source, Definition: domain.LanguageServerDefinition{Command: command, Args: args, LanguageIDs: []string{language}},
	})
}

func startBoundedLSPClientWithDefinition(ctx context.Context, workspaceRoot string, language string, resolved resolvedLSPDefinition) (*boundedLSPClient, error) {
	processCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processCtx, resolved.Definition.Command, resolved.Definition.Args...)
	cmd.Dir = workspaceRoot
	cmd.Env = append(os.Environ(), resolvedLSPEnvironment(resolved.Definition.Env)...)
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
		root: workspaceRoot, language: language, source: resolved.Name, revision: resolved.Revision, cmd: cmd, stdin: stdin, cancel: cancel,
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
	initializeParams := map[string]any{
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
	}
	if len(resolved.Definition.InitializationOptions) > 0 {
		initializeParams["initializationOptions"] = resolved.Definition.InitializationOptions
	}
	if err := client.request(initCtx, "initialize", initializeParams, &result); err != nil {
		client.close()
		return nil, err
	}
	_ = client.notify("initialized", map[string]any{})
	return client, nil
}

func resolvedLSPEnvironment(configured map[string]string) []string {
	keys := make([]string, 0, len(configured))
	for key := range configured {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.TrimSpace(configured[key])
		reference := strings.TrimPrefix(value, "$")
		reference = strings.TrimPrefix(reference, "{")
		reference = strings.TrimSuffix(reference, "}")
		if strings.HasPrefix(value, "$") {
			value = os.Getenv(reference)
		}
		out = append(out, key+"="+value)
	}
	return out
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
