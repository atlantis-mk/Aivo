package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

type mcpStdioClient struct {
	server           domain.MCPServerConfig
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	lines            *bufio.Reader
	mu               sync.Mutex
	toolsListChanged bool
}

func startMCPStdio(ctx context.Context, server domain.MCPServerConfig) (*mcpStdioClient, error) {
	if strings.TrimSpace(server.Command) == "" {
		return nil, errors.New("stdio MCP server command is required")
	}
	command := server.Command
	cmd := exec.CommandContext(ctx, command, server.Args...)
	if server.CWD != "" {
		cmd.Dir = server.CWD
	} else if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	cmd.Env = SanitizedEnvironment(firstNonEmptyApp(cmd.Dir, "."), defaultEnvAllowlist(), server.Env, nil)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = mcpStderrWriter(server.ID)
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcpStdioClient{server: server, cmd: cmd, stdin: stdin, lines: bufio.NewReader(stdout)}, nil
}

func (c *mcpStdioClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return nil, err
	}
	timeout := time.Duration(c.server.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	deadline := time.After(timeout)
	for {
		line, err := c.readLine(ctx, deadline)
		if err != nil {
			return nil, err
		}
		var resp struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result map[string]any  `json:"result"`
			Error  any             `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		if resp.Method != "" {
			_ = c.handleServerMessage(resp.ID, resp.Method)
			continue
		}
		var responseID string
		_ = json.Unmarshal(resp.ID, &responseID)
		if responseID != id {
			continue
		}
		if resp.Error != nil {
			return nil, mcpRPCError(resp.Error)
		}
		return resp.Result, nil
	}
}

func (c *mcpStdioClient) readLine(ctx context.Context, deadline <-chan time.Time) (string, error) {
	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := c.lines.ReadString('\n')
		ch <- readResult{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		c.close()
		return "", ctx.Err()
	case <-deadline:
		c.close()
		return "", errors.New("mcp request timed out")
	case result := <-ch:
		return result.line, result.err
	}
}

func (c *mcpStdioClient) handleServerMessage(id json.RawMessage, method string) error {
	if len(id) == 0 || string(id) == "null" {
		c.handleServerNotification(method)
		return nil
	}
	return c.handleServerRequest(id, method)
}

func (c *mcpStdioClient) handleServerNotification(method string) {
	switch method {
	case "notifications/tools/list_changed", "tools/list_changed":
		c.toolsListChanged = true
	}
}

func (c *mcpStdioClient) consumeToolsListChanged() bool {
	if c == nil || !c.toolsListChanged {
		return false
	}
	c.toolsListChanged = false
	return true
}

func (c *mcpStdioClient) handleServerRequest(id json.RawMessage, method string) error {
	var result map[string]any
	switch method {
	case "roots/list":
		result = map[string]any{"roots": mcpRootEntries(c.server)}
	default:
		raw, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"error":   map[string]any{"code": -32601, "message": "method not found"},
		})
		_, err := c.stdin.Write(append(raw, '\n'))
		return err
	}
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id), "result": result})
	_, err := c.stdin.Write(append(raw, '\n'))
	return err
}

func (c *mcpStdioClient) close() {
	if c == nil {
		return
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
}
