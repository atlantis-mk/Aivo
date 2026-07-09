package app

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func mcpStderrWriter(serverID string) io.Writer {
	path, err := mcpLogPath(serverID)
	if err != nil {
		return io.Discard
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return io.Discard
	}
	return file
}

const (
	defaultMCPLogReadLimit = 16000
	maxMCPLogReadLimit     = 100000
)

func readMCPServerLog(ctx context.Context, input domain.MCPServerLogInput) (domain.MCPServerLogResult, error) {
	select {
	case <-ctx.Done():
		return domain.MCPServerLogResult{}, ctx.Err()
	default:
	}
	path, err := mcpLogPath(input.ServerID)
	if err != nil {
		return domain.MCPServerLogResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.MCPServerLogResult{ServerID: strings.TrimSpace(input.ServerID)}, nil
		}
		return domain.MCPServerLogResult{}, err
	}
	if info.IsDir() {
		return domain.MCPServerLogResult{}, errors.New("mcp log path is a directory")
	}
	size := int(info.Size())
	limit := input.Limit
	if limit <= 0 {
		limit = defaultMCPLogReadLimit
	}
	if limit > maxMCPLogReadLimit {
		limit = maxMCPLogReadLimit
	}
	offset := input.Offset
	if input.Tail {
		offset = size - limit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > size {
		offset = size
	}
	file, err := os.Open(path)
	if err != nil {
		return domain.MCPServerLogResult{}, err
	}
	defer file.Close()
	buffer := make([]byte, limit)
	n, err := file.ReadAt(buffer, int64(offset))
	if err != nil && n == 0 && offset < size {
		return domain.MCPServerLogResult{}, err
	}
	nextOffset := offset + n
	return domain.MCPServerLogResult{
		ServerID:   strings.TrimSpace(input.ServerID),
		Content:    string(buffer[:n]),
		Offset:     offset,
		NextOffset: nextOffset,
		Size:       size,
		Truncated:  nextOffset < size,
	}, nil
}

func mcpLogPath(serverID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cleanServerID := safeArtifactPart(strings.TrimSpace(serverID))
	if cleanServerID == "" {
		cleanServerID = "server"
	}
	return filepath.Join(home, ".aivo", "logs", "mcp-"+cleanServerID+".log"), nil
}
