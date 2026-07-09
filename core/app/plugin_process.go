package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func startPluginProcess(ctx context.Context, plugin domain.PluginInstall) (*pluginProcessClient, error) {
	entry := plugin.Manifest.Entrypoint
	if strings.TrimSpace(entry.Command) == "" {
		return nil, errors.New("plugin entrypoint.command is required")
	}
	root := plugin.RootPath
	command := entry.Command
	if !filepath.IsAbs(command) && strings.Contains(command, string(os.PathSeparator)) {
		command = filepath.Join(root, command)
	}
	cmd := exec.CommandContext(ctx, command, entry.Args...)
	cmd.Dir = firstNonEmptyApp(entry.CWD, root)
	if !filepath.IsAbs(cmd.Dir) {
		cmd.Dir = filepath.Join(root, cmd.Dir)
	}
	if !pathWithin(root, cmd.Dir) {
		return nil, errors.New("plugin entrypoint cwd escapes plugin root")
	}
	cmd.Env = SanitizedEnvironment(root, defaultEnvAllowlist(), entry.Env, nil)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = pluginStderrWriter(plugin.ID)
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &pluginProcessClient{cmd: cmd, stdin: stdin, lines: bufio.NewReader(stdout)}, nil
}

func (c *pluginProcessClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil {
		return nil, errors.New("plugin process is not running")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	id := uuid.NewString()
	request := map[string]any{"id": id, "method": method, "params": params}
	raw, _ := json.Marshal(request)
	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return nil, err
	}
	type response struct {
		ID     string         `json:"id"`
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	deadline := time.After(20 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, errors.New("plugin request timed out")
		default:
			line, err := c.lines.ReadString('\n')
			if err != nil {
				return nil, err
			}
			var resp response
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}
			if resp.ID != id {
				continue
			}
			if resp.Error != nil {
				return nil, fmt.Errorf("%v", resp.Error)
			}
			return resp.Result, nil
		}
	}
}

func (c *pluginProcessClient) close() error {
	if c == nil {
		return nil
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
	return nil
}

func pluginStderrWriter(pluginID string) io.Writer {
	home, err := os.UserHomeDir()
	if err != nil {
		return io.Discard
	}
	dir := filepath.Join(home, ".aivo", "logs")
	_ = os.MkdirAll(dir, 0o755)
	file, err := os.OpenFile(filepath.Join(dir, "plugins-"+pluginID+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return io.Discard
	}
	return file
}
