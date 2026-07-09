package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"aivo/core/domain"
)

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
