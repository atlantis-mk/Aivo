package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func pathWithin(root string, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func firstNonEmptyApp(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeExternalToolResult(name string, result map[string]any) domain.ToolResult {
	if result == nil {
		return domain.ToolResult{Name: name, OK: true}
	}
	ok := true
	if rawOK, exists := result["ok"].(bool); exists {
		ok = rawOK
	}
	content, _ := result["content"].(string)
	if content == "" {
		content, _ = result["output"].(string)
	}
	if content == "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		content = string(raw)
	}
	errorText, _ := result["error"].(string)
	return domain.ToolResult{Name: name, OK: ok, Content: content, ModelContent: content, Structured: result, Error: errorText}
}
