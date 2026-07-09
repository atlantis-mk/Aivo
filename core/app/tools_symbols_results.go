package app

import (
	"fmt"
	"strings"

	"aivo/core/domain"
)

func lspSymbolsResult(toolName string, query string, kind string, symbols []domain.CodeSymbol, status domain.CodeIntelligenceStatus, filesScanned int, truncated bool) domain.ToolResult {
	structured := make([]map[string]any, 0, len(symbols))
	lines := make([]string, 0, len(symbols))
	for _, result := range symbols {
		item := map[string]any{
			"name":      result.Name,
			"kind":      result.Kind,
			"path":      result.Path,
			"line":      result.Range.Start.Line,
			"language":  result.Language,
			"signature": result.Signature,
			"source":    result.Source,
		}
		structured = append(structured, item)
		lines = append(lines, fmt.Sprintf("%s:%d %s %s [%s]\n  %s", result.Path, result.Range.Start.Line, result.Kind, result.Name, result.Language, result.Signature))
	}
	content := "No symbols found"
	if len(lines) > 0 {
		content = strings.Join(lines, "\n")
	}
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: showing first %d symbols]", len(symbols))
	}
	return domain.ToolResult{
		Name:         toolName,
		OK:           true,
		Content:      content,
		ModelContent: content,
		Structured: map[string]any{
			"status":       status,
			"query":        query,
			"kind":         kind,
			"results":      structured,
			"resultCount":  len(symbols),
			"filesScanned": filesScanned,
			"truncated":    truncated,
		},
		Truncated: truncated,
	}
}

func lspFallbackStatus(workspaceRoot string, message string) domain.CodeIntelligenceStatus {
	return domain.CodeIntelligenceStatus{WorkspaceRoot: workspaceRoot, Status: domain.CodeIntelligenceStatusFallback, Source: "scan", Message: message}
}

func lspUnavailableResult(toolName string, status domain.CodeIntelligenceStatus) domain.ToolResult {
	content := "Code intelligence unavailable"
	if status.Message != "" {
		content += ": " + status.Message
	}
	return domain.ToolResult{OK: true, Name: toolName, Content: content, ModelContent: content, Structured: map[string]any{
		"status": status, "resultCount": 0,
	}}
}

func lspDiagnosticsResult(toolName string, diagnostics []domain.CodeDiagnostic, status domain.CodeIntelligenceStatus, filesScanned int, truncated bool) domain.ToolResult {
	lines := []string{}
	for _, diagnostic := range diagnostics {
		lines = append(lines, fmt.Sprintf("%s:%d:%d %s %s", diagnostic.Path, diagnostic.Range.Start.Line, diagnostic.Range.Start.Character, diagnostic.Severity, diagnostic.Message))
	}
	content := "No diagnostics found"
	if len(lines) > 0 {
		content = strings.Join(lines, "\n")
	}
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: showing first %d diagnostics]", len(diagnostics))
	}
	return domain.ToolResult{OK: true, Name: toolName, Content: content, ModelContent: content, Structured: map[string]any{
		"status": status, "diagnostics": diagnostics, "resultCount": len(diagnostics), "filesScanned": filesScanned, "truncated": truncated,
	}, Truncated: truncated}
}

func lspLocationsResult(toolName string, symbol string, locations []domain.CodeLocation, status domain.CodeIntelligenceStatus, truncated bool) domain.ToolResult {
	lines := []string{}
	for _, location := range locations {
		lines = append(lines, fmt.Sprintf("%s:%d:%d %s", location.Path, location.Range.Start.Line, location.Range.Start.Character, location.Preview))
	}
	content := "No locations found"
	if len(lines) > 0 {
		content = strings.Join(lines, "\n")
	}
	if truncated {
		content += fmt.Sprintf("\n\n[truncated: showing first %d locations]", len(locations))
	}
	return domain.ToolResult{OK: true, Name: toolName, Content: content, ModelContent: content, Structured: map[string]any{
		"symbol":    symbol,
		"status":    status,
		"locations": locations, "resultCount": len(locations), "truncated": truncated,
	}, Truncated: truncated}
}
