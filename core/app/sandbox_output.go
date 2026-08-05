package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type boundedOutputCapture struct {
	file      *os.File
	path      string
	tail      []byte
	maxBytes  int
	total     int
	emitter   *shellOutputEmitter
	stream    string
	finalized bool
}

func newBoundedOutputCapture(request SandboxRequest, stream string, maxBytes int, emitter *shellOutputEmitter) (*boundedOutputCapture, error) {
	if maxBytes <= 0 {
		maxBytes = defaultStreamMaxChars
	}
	path, err := retainedOutputPath(request, stream)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	return &boundedOutputCapture{file: file, path: path, maxBytes: maxBytes, emitter: emitter, stream: stream}, nil
}

func (c *boundedOutputCapture) Write(p []byte) (int, error) {
	if c == nil || c.file == nil {
		return 0, os.ErrInvalid
	}
	n, err := c.file.Write(p)
	if n > 0 {
		c.total += n
		c.tail = append(c.tail, p[:n]...)
		if len(c.tail) > c.maxBytes {
			c.tail = append([]byte(nil), c.tail[len(c.tail)-c.maxBytes:]...)
		}
		if c.emitter != nil {
			c.emitter.emit(c.stream, string(p[:n]))
		}
	}
	return n, err
}

func (c *boundedOutputCapture) Finish() (string, bool, string, int, error) {
	if c == nil || c.file == nil {
		return "", false, "", 0, os.ErrInvalid
	}
	if err := c.file.Sync(); err != nil {
		_ = c.file.Close()
		return "", false, "", c.total, err
	}
	if err := c.file.Close(); err != nil {
		return "", false, "", c.total, err
	}
	c.file = nil
	c.finalized = true
	tail := bytes.ToValidUTF8(c.tail, []byte("�"))
	if c.total <= c.maxBytes {
		_ = os.Remove(c.path)
		_ = os.Remove(filepath.Dir(c.path))
		return string(tail), false, "", c.total, nil
	}
	return fmt.Sprintf("[truncated: %s exceeded %d characters; full output retained at %s; showing tail]\n\n", c.stream, c.maxBytes, c.path) + string(tail), true, c.path, c.total, nil
}

func (c *boundedOutputCapture) Abort() {
	if c == nil || c.finalized {
		return
	}
	if c.file != nil {
		_ = c.file.Close()
		c.file = nil
	}
	_ = os.Remove(c.path)
}

func boundedSandboxOutput(request SandboxRequest, stream string, content string, maxChars int) (string, bool, string, int, error) {
	original := len(content)
	if original <= maxChars {
		return content, false, "", original, nil
	}
	ref, err := retainSandboxOutput(request, stream, content)
	if err != nil {
		return "", false, "", original, &SandboxError{Code: SandboxErrorOutputRetentionFailed, Message: err.Error(), Err: err}
	}
	tail := content[len(content)-maxChars:]
	return fmt.Sprintf("[truncated: %s exceeded %d characters; full output retained at %s; showing tail]\n\n", stream, maxChars, ref) + tail, true, ref, original, nil
}

func retainSandboxOutput(request SandboxRequest, stream string, content string) (string, error) {
	path, err := retainedOutputPath(request, stream)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func retainedOutputPath(request SandboxRequest, stream string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	sessionID := safeArtifactPart(firstNonEmpty(request.SessionID, "session"))
	toolCallID := safeArtifactPart(firstNonEmpty(request.ToolCallID, fmt.Sprintf("%d", time.Now().UnixNano())))
	dir := filepath.Join(base, "aivo", "command-artifacts", sessionID, toolCallID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, safeArtifactPart(stream)+".log"), nil
}

func safeArtifactPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "artifact"
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

func redactCommandOutput(value string) string {
	for _, marker := range []string{"OPENAI_API_KEY=", "ANTHROPIC_API_KEY=", "GITHUB_TOKEN=", "GOOGLE_API_KEY="} {
		value = redactAfterMarker(value, marker)
	}
	return value
}

func redactAfterMarker(value string, marker string) string {
	for {
		idx := strings.Index(value, marker)
		if idx < 0 {
			return value
		}
		start := idx + len(marker)
		end := start
		for end < len(value) && value[end] != '\n' && value[end] != ' ' && value[end] != '\t' {
			end++
		}
		value = value[:start] + "[redacted]" + value[end:]
	}
}
