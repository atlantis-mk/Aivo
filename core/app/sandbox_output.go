package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func boundedSandboxOutput(request SandboxRequest, stream string, content string, maxChars int) (string, bool, string, int, error) {
	original := len(content)
	if original <= maxChars {
		return content, false, "", original, nil
	}
	ref, err := retainSandboxOutput(request, stream, content)
	if err != nil {
		return "", false, "", original, &SandboxError{Code: SandboxErrorOutputRetentionFailed, Message: err.Error(), Err: err}
	}
	return content[:maxChars] + fmt.Sprintf("\n\n[truncated: %s exceeded %d characters; full output retained at %s]", stream, maxChars, ref), true, ref, original, nil
}

func retainSandboxOutput(request SandboxRequest, stream string, content string) (string, error) {
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
	path := filepath.Join(dir, safeArtifactPart(stream)+".log")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
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
