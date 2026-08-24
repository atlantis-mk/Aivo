package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	symbolSearchMaxResults = 100
	symbolSearchMaxFiles   = 5000
	symbolSearchMaxBytes   = 1024 * 1024
)

type LSPDiagnosticsTool struct {
	workspaceRoot string
	intelligence  domain.CodeIntelligenceService
}

func NewLSPDiagnosticsTool(workspaceRoot string) *LSPDiagnosticsTool {
	return &LSPDiagnosticsTool{workspaceRoot: workspaceRoot, intelligence: codeIntelligenceService()}
}

func (t *LSPDiagnosticsTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "lsp_diagnostics", Description: "Return bounded source diagnostics for a workspace path. Uses language-server diagnostics when available and a safe local fallback otherwise.",
		Namespace: filesystemNamespace, NamespaceDescription: filesystemNamespaceDescription, Capability: "lsp.diagnostics", RiskLevel: "low", Category: "lsp",
		Toolsets: []string{"safe", "coding"}, RequiresWorkspace: true,
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "Optional workspace-relative file or directory."},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200},
		}},
	}
}

func (t *LSPDiagnosticsTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("lsp_diagnostics", err)
	}
	limit := input.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	target, err := safeJoin(root, input.Path)
	if err != nil {
		return toolError("lsp_diagnostics", err)
	}
	status := domain.CodeIntelligenceStatus{}
	if t.intelligence != nil {
		lspDiagnostics, lspStatus, lspErr := t.intelligence.Diagnostics(ctx, root, input.Path)
		status = lspStatus
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusReady {
			fallback, filesScanned, truncated := fallbackDiagnostics(ctx, root, target, limit-len(lspDiagnostics))
			diagnostics := append(lspDiagnostics, fallback...)
			return lspDiagnosticsResult("lsp_diagnostics", diagnostics, lspStatus, filesScanned, truncated)
		}
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusUnavailable && lspStatus.Language == "" {
			return lspUnavailableResult("lsp_diagnostics", lspStatus)
		}
	}
	diagnostics, filesScanned, truncated := fallbackDiagnostics(ctx, root, target, limit)
	if status.Status == "" {
		status = lspFallbackStatus(root, "language server unavailable; used bounded fallback scan")
	} else {
		status.Status = domain.CodeIntelligenceStatusFallback
		status.Source = "scan"
		status.Message = nonEmpty(status.Message, "language server unavailable") + "; used bounded fallback scan"
	}
	return lspDiagnosticsResult("lsp_diagnostics", diagnostics, status, filesScanned, truncated)
}

type LSPDefinitionTool struct {
	workspaceRoot string
	intelligence  domain.CodeIntelligenceService
}

func NewLSPDefinitionTool(workspaceRoot string) *LSPDefinitionTool {
	return &LSPDefinitionTool{workspaceRoot: workspaceRoot, intelligence: codeIntelligenceService()}
}

func (t *LSPDefinitionTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "lsp_definition", Description: "Find definition locations for the symbol at a workspace-relative file position. Falls back to bounded source symbol scanning when no language server is ready.",
		Namespace: filesystemNamespace, NamespaceDescription: filesystemNamespaceDescription, Capability: "lsp.definition", RiskLevel: "low", Category: "lsp",
		Toolsets: []string{"safe", "coding"}, RequiresWorkspace: true,
		InputSchema: lspPositionInputSchema(),
	}
}

func (t *LSPDefinitionTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := decodeLSPPositionInput(args)
	if err != nil {
		return toolError("lsp_definition", err)
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	symbol, err := symbolAtPosition(root, input.Path, input.Line, input.Character)
	if err != nil {
		return toolError("lsp_definition", err)
	}
	limit := boundedLookupLimit(input.Limit)
	status := domain.CodeIntelligenceStatus{}
	if t.intelligence != nil {
		locations, lspStatus, lspErr := t.intelligence.Definition(ctx, root, input.Path, domain.SourcePosition{Line: input.Line, Character: input.Character}, limit)
		status = lspStatus
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusReady && len(locations) > 0 {
			return lspLocationsResult("lsp_definition", symbol, locations, lspStatus, len(locations) >= limit)
		}
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusUnavailable && lspStatus.Language == "" {
			return lspUnavailableResult("lsp_definition", lspStatus)
		}
	}
	results, _, truncated, err := scanSymbolDefinitions(ctx, root, symbol, limit)
	if err != nil {
		return toolError("lsp_definition", err)
	}
	if status.Status == "" {
		status = lspFallbackStatus(root, "language server unavailable; used bounded fallback scan")
	} else {
		status.Status = domain.CodeIntelligenceStatusFallback
		status.Source = "scan"
		status.Message = nonEmpty(status.Message, "language server unavailable") + "; used bounded fallback scan"
	}
	return lspLocationsResult("lsp_definition", symbol, results, status, truncated)
}

type LSPReferencesTool struct {
	workspaceRoot string
	intelligence  domain.CodeIntelligenceService
}

func NewLSPReferencesTool(workspaceRoot string) *LSPReferencesTool {
	return &LSPReferencesTool{workspaceRoot: workspaceRoot, intelligence: codeIntelligenceService()}
}

func (t *LSPReferencesTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: "lsp_references", Description: "Find bounded references for the symbol at a workspace-relative file position. Falls back to safe source scanning when no language server is ready.",
		Namespace: filesystemNamespace, NamespaceDescription: filesystemNamespaceDescription, Capability: "lsp.references", RiskLevel: "low", Category: "lsp",
		Toolsets: []string{"safe", "coding"}, RequiresWorkspace: true,
		InputSchema: lspPositionInputSchema(),
	}
}

func (t *LSPReferencesTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := decodeLSPPositionInput(args)
	if err != nil {
		return toolError("lsp_references", err)
	}
	root := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	symbol, err := symbolAtPosition(root, input.Path, input.Line, input.Character)
	if err != nil {
		return toolError("lsp_references", err)
	}
	limit := boundedLookupLimit(input.Limit)
	status := domain.CodeIntelligenceStatus{}
	if t.intelligence != nil {
		locations, lspStatus, lspErr := t.intelligence.References(ctx, root, input.Path, domain.SourcePosition{Line: input.Line, Character: input.Character}, limit)
		status = lspStatus
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusReady && len(locations) > 0 {
			return lspLocationsResult("lsp_references", symbol, locations, lspStatus, len(locations) >= limit)
		}
		if lspErr == nil && lspStatus.Status == domain.CodeIntelligenceStatusUnavailable && lspStatus.Language == "" {
			return lspUnavailableResult("lsp_references", lspStatus)
		}
	}
	results, truncated, err := scanSymbolReferences(ctx, root, symbol, limit)
	if err != nil {
		return toolError("lsp_references", err)
	}
	if status.Status == "" {
		status = lspFallbackStatus(root, "language server unavailable; used bounded fallback scan")
	} else {
		status.Status = domain.CodeIntelligenceStatusFallback
		status.Source = "scan"
		status.Message = nonEmpty(status.Message, "language server unavailable") + "; used bounded fallback scan"
	}
	return lspLocationsResult("lsp_references", symbol, results, status, truncated)
}

type LSPSymbolSearchTool struct {
	workspaceRoot string
	intelligence  domain.CodeIntelligenceService
}

func NewLSPSymbolSearchTool(workspaceRoot string) *LSPSymbolSearchTool {
	return &LSPSymbolSearchTool{workspaceRoot: workspaceRoot, intelligence: codeIntelligenceService()}
}

func (t *LSPSymbolSearchTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "lsp_symbol_search",
		Description:          "Search workspace source symbols by name using an LSP-style definition scan. Use this to find functions, classes, types, interfaces, variables, methods, and similar definitions before reading files. Generated directories, sensitive files, and paths ignored by .gitignore are skipped. Results are bounded and include path, line, kind, language, and signature.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: filesystemNamespaceDescription,
		Capability:           "lsp.symbol_search",
		RiskLevel:            "low",
		Category:             "lsp",
		Toolsets:             []string{"safe", "coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Symbol name or fragment to search for. Case-insensitive."},
				"path":  map[string]any{"type": "string", "description": "Optional workspace-relative file or directory to limit the search."},
				"kind":  map[string]any{"type": "string", "description": "Optional symbol kind filter such as function, method, class, type, interface, enum, variable, constant, or struct."},
				"limit": map[string]any{"type": "integer", "description": "Maximum results. Defaults to 50; max 100.", "minimum": 1, "maximum": symbolSearchMaxResults},
			},
			"required": []string{"query"},
		},
	}
}

func (t *LSPSymbolSearchTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Query string `json:"query"`
		Path  string `json:"path"`
		Kind  string `json:"kind"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("lsp_symbol_search", err)
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolError("lsp_symbol_search", errors.New("query is required"))
	}
	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > symbolSearchMaxResults {
		limit = symbolSearchMaxResults
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	searchRoot, err := safeJoin(workspaceRoot, input.Path)
	if err != nil {
		return toolError("lsp_symbol_search", err)
	}
	info, err := os.Stat(searchRoot)
	if err != nil {
		return toolError("lsp_symbol_search", err)
	}
	var lspSymbols []domain.CodeSymbol
	lspStatus := domain.CodeIntelligenceStatus{}
	if t.intelligence != nil {
		symbols, status, lspErr := t.intelligence.Symbols(ctx, workspaceRoot, query, input.Path, kind, limit)
		lspStatus = status
		if lspErr == nil && status.Status == domain.CodeIntelligenceStatusReady {
			lspSymbols = symbols
		}
		if lspErr == nil && status.Status == domain.CodeIntelligenceStatusUnavailable && status.Language == "" {
			return lspUnavailableResult("lsp_symbol_search", status)
		}
	}
	var results []symbolSearchResult
	filesScanned := 0
	truncated := false
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	visit := func(current string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel := filepath.ToSlash(mustRel(workspaceRoot, current))
		if isSensitiveRelPath(rel) || !symbolFileSupported(rel) {
			return nil
		}
		if filesScanned >= symbolSearchMaxFiles {
			truncated = true
			return errStopWalk
		}
		filesScanned++
		matches, err := scanSymbolsInFile(ctx, workspaceRoot, current, query, kind, limit-len(results))
		if err != nil {
			return nil
		}
		results = append(results, matches...)
		if len(results) >= limit {
			truncated = true
			return errStopWalk
		}
		return nil
	}
	if info.IsDir() {
		err = filepath.WalkDir(searchRoot, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if shouldSkipWorkspaceEntry(workspaceRoot, current, entry, ignore) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			return visit(current)
		})
	} else {
		err = visit(searchRoot)
	}
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	if err != nil {
		return toolError("lsp_symbol_search", err)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Path != results[j].Path {
			return results[i].Path < results[j].Path
		}
		return results[i].Line < results[j].Line
	})
	if len(results) > limit {
		results = results[:limit]
		truncated = true
	}
	symbols := make([]domain.CodeSymbol, 0, len(lspSymbols)+len(results))
	seen := map[string]bool{}
	for _, result := range lspSymbols {
		key := result.Path + "\x00" + result.Name + "\x00" + fmt.Sprint(result.Range.Start.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		symbols = append(symbols, result)
	}
	for _, result := range results {
		if len(symbols) >= limit {
			truncated = true
			break
		}
		key := result.Path + "\x00" + result.Name + "\x00" + fmt.Sprint(result.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		symbols = append(symbols, domain.CodeSymbol{
			Name: result.Name, Kind: result.Kind, Path: result.Path, Language: result.Language, Signature: result.Signature, Source: "fallback-scan",
			Range: domain.SourceRange{Start: domain.SourcePosition{Line: result.Line, Character: 0}, End: domain.SourcePosition{Line: result.Line, Character: len(result.Signature)}},
		})
	}
	status := lspFallbackStatus(workspaceRoot, "language server unavailable; used bounded fallback scan")
	if lspStatus.Status == domain.CodeIntelligenceStatusReady {
		status = lspStatus
	}
	return lspSymbolsResult("lsp_symbol_search", query, kind, symbols, status, filesScanned, truncated)
}

type lspPositionInput struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
	Limit     int    `json:"limit"`
}

func lspPositionInputSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{
		"path":      map[string]any{"type": "string", "description": "Workspace-relative source file path."},
		"line":      map[string]any{"type": "integer", "description": "1-based line number.", "minimum": 1},
		"character": map[string]any{"type": "integer", "description": "0-based character offset.", "minimum": 0},
		"limit":     map[string]any{"type": "integer", "minimum": 1, "maximum": symbolSearchMaxResults},
	}, "required": []string{"path", "line"}}
}

func decodeLSPPositionInput(args json.RawMessage) (lspPositionInput, error) {
	var input lspPositionInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, err
	}
	if strings.TrimSpace(input.Path) == "" {
		return input, errors.New("path is required")
	}
	if input.Line <= 0 {
		return input, errors.New("line must be 1-based")
	}
	return input, nil
}

func boundedLookupLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > symbolSearchMaxResults {
		return symbolSearchMaxResults
	}
	return limit
}

func symbolAtPosition(workspaceRoot string, relPath string, lineNo int, character int) (string, error) {
	path, err := safeJoin(workspaceRoot, relPath)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(raw), "\n")
	if lineNo > len(lines) {
		return "", errors.New("line is outside file")
	}
	line := lines[lineNo-1]
	if character < 0 {
		character = 0
	}
	if character > len(line) {
		character = len(line)
	}
	start := character
	for start > 0 && isIdentByte(line[start-1]) {
		start--
	}
	end := character
	for end < len(line) && isIdentByte(line[end]) {
		end++
	}
	if start == end && character > 0 && character <= len(line) && isIdentByte(line[character-1]) {
		start = character - 1
		for start > 0 && isIdentByte(line[start-1]) {
			start--
		}
		end = character
		for end < len(line) && isIdentByte(line[end]) {
			end++
		}
	}
	symbol := strings.TrimSpace(line[start:end])
	if symbol == "" {
		return "", errors.New("no symbol at position")
	}
	return symbol, nil
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '$'
}
