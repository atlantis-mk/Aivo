package app

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"aivo/core/domain"
)

type symbolSearchResult struct {
	Name      string
	Kind      string
	Path      string
	Line      int
	Language  string
	Signature string
	Score     int
}

type symbolPattern struct {
	Kind        string
	Expression  *regexp.Regexp
	NameIndex   int
	ReceiverIdx int
}

var symbolPatternsByExt = map[string][]symbolPattern{
	".go": {
		{Kind: "function", Expression: regexp.MustCompile(`^\s*func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), NameIndex: 1},
		{Kind: "method", Expression: regexp.MustCompile(`^\s*func\s+\(([^)]*)\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(`), NameIndex: 2, ReceiverIdx: 1},
		{Kind: "type", Expression: regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`), NameIndex: 1},
		{Kind: "constant", Expression: regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "variable", Expression: regexp.MustCompile(`^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
	},
	".ts":  tsLikeSymbolPatterns(),
	".tsx": tsLikeSymbolPatterns(),
	".js":  tsLikeSymbolPatterns(),
	".jsx": tsLikeSymbolPatterns(),
	".py": {
		{Kind: "function", Expression: regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), NameIndex: 1},
		{Kind: "class", Expression: regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "variable", Expression: regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*=`), NameIndex: 1},
	},
	".rs": {
		{Kind: "function", Expression: regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\s*`), NameIndex: 1},
		{Kind: "struct", Expression: regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "enum", Expression: regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "trait", Expression: regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
	},
	".java": jvmLikeSymbolPatterns(),
	".cs":   jvmLikeSymbolPatterns(),
}

func tsLikeSymbolPatterns() []symbolPattern {
	return []symbolPattern{
		{Kind: "function", Expression: regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`), NameIndex: 1},
		{Kind: "class", Expression: regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`), NameIndex: 1},
		{Kind: "interface", Expression: regexp.MustCompile(`^\s*(?:export\s+)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`), NameIndex: 1},
		{Kind: "type", Expression: regexp.MustCompile(`^\s*(?:export\s+)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`), NameIndex: 1},
		{Kind: "enum", Expression: regexp.MustCompile(`^\s*(?:export\s+)?enum\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`), NameIndex: 1},
		{Kind: "variable", Expression: regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:[:=]|=>)`), NameIndex: 1},
		{Kind: "method", Expression: regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+|static\s+|async\s+)*([A-Za-z_$][A-Za-z0-9_$]*)\s*\([^)]*\)\s*[:{]`), NameIndex: 1},
	}
}

func jvmLikeSymbolPatterns() []symbolPattern {
	return []symbolPattern{
		{Kind: "class", Expression: regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+|abstract\s+|final\s+)*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "interface", Expression: regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)*interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "enum", Expression: regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)*enum\s+([A-Za-z_][A-Za-z0-9_]*)\b`), NameIndex: 1},
		{Kind: "method", Expression: regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+|static\s+|async\s+|final\s+)*[A-Za-z_<>\[\], ?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`), NameIndex: 1},
	}
}

func symbolFileSupported(path string) bool {
	_, ok := symbolPatternsByExt[strings.ToLower(filepath.Ext(path))]
	return ok
}

func scanSymbolsInFile(ctx context.Context, workspaceRoot string, path string, query string, kind string, remaining int) ([]symbolSearchResult, error) {
	if remaining <= 0 {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > symbolSearchMaxBytes {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(path))
	patterns := symbolPatternsByExt[ext]
	language := symbolLanguage(ext)
	queryLower := strings.ToLower(query)
	var results []symbolSearchResult
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		lineNo++
		line := scanner.Text()
		for _, pattern := range patterns {
			if kind != "" && pattern.Kind != kind {
				continue
			}
			match := pattern.Expression.FindStringSubmatch(line)
			if len(match) <= pattern.NameIndex {
				continue
			}
			name := strings.TrimSpace(match[pattern.NameIndex])
			if name == "" || !strings.Contains(strings.ToLower(name), queryLower) {
				continue
			}
			displayName := name
			if pattern.ReceiverIdx > 0 && len(match) > pattern.ReceiverIdx {
				receiver := strings.TrimSpace(match[pattern.ReceiverIdx])
				if receiver != "" {
					displayName = receiver + "." + name
				}
			}
			results = append(results, symbolSearchResult{
				Name:      displayName,
				Kind:      pattern.Kind,
				Path:      filepath.ToSlash(mustRel(workspaceRoot, path)),
				Line:      lineNo,
				Language:  language,
				Signature: strings.TrimSpace(line),
				Score:     symbolMatchScore(name, query),
			})
			if len(results) >= remaining {
				return results, scanner.Err()
			}
		}
	}
	return results, scanner.Err()
}

func symbolLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}

func symbolMatchScore(name string, query string) int {
	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)
	if nameLower == queryLower {
		return 100
	}
	if strings.HasPrefix(nameLower, queryLower) {
		return 75
	}
	return 50
}

func fallbackDiagnostics(ctx context.Context, workspaceRoot string, target string, limit int) ([]domain.CodeDiagnostic, int, bool) {
	diagnostics := []domain.CodeDiagnostic{}
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
		filesScanned++
		fileDiagnostics := scanTodoDiagnostics(ctx, workspaceRoot, current, limit-len(diagnostics))
		diagnostics = append(diagnostics, fileDiagnostics...)
		if len(diagnostics) >= limit {
			truncated = true
			return errStopWalk
		}
		return nil
	}
	info, err := os.Stat(target)
	if err != nil {
		return diagnostics, filesScanned, false
	}
	if info.IsDir() {
		err = filepath.WalkDir(target, func(current string, entry os.DirEntry, walkErr error) error {
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
		err = visit(target)
	}
	if err != nil && !errors.Is(err, errStopWalk) {
		return diagnostics, filesScanned, truncated
	}
	return diagnostics, filesScanned, truncated
}

func scanTodoDiagnostics(ctx context.Context, workspaceRoot string, path string, remaining int) []domain.CodeDiagnostic {
	if remaining <= 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > symbolSearchMaxBytes {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var out []domain.CodeDiagnostic
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return out
		}
		lineNo++
		line := scanner.Text()
		upper := strings.ToUpper(line)
		index := strings.Index(upper, "TODO")
		if index < 0 {
			index = strings.Index(upper, "FIXME")
		}
		if index < 0 {
			continue
		}
		out = append(out, domain.CodeDiagnostic{
			Path:     filepath.ToSlash(mustRel(workspaceRoot, path)),
			Range:    domain.SourceRange{Start: domain.SourcePosition{Line: lineNo, Character: index}, End: domain.SourcePosition{Line: lineNo, Character: len(line)}},
			Severity: domain.DiagnosticSeverityInformation, Message: strings.TrimSpace(line), Source: "fallback-scan",
		})
		if len(out) >= remaining {
			break
		}
	}
	return out
}

func scanSymbolDefinitions(ctx context.Context, workspaceRoot string, symbol string, limit int) ([]domain.CodeLocation, int, bool, error) {
	results := []domain.CodeLocation{}
	filesScanned := 0
	truncated := false
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	err := filepath.WalkDir(workspaceRoot, func(current string, entry os.DirEntry, walkErr error) error {
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
		rel := filepath.ToSlash(mustRel(workspaceRoot, current))
		if isSensitiveRelPath(rel) || !symbolFileSupported(rel) {
			return nil
		}
		filesScanned++
		matches, _ := scanSymbolsInFile(ctx, workspaceRoot, current, symbol, "", limit-len(results))
		for _, match := range matches {
			if strings.EqualFold(match.Name, symbol) || strings.HasSuffix(strings.ToLower(match.Name), "."+strings.ToLower(symbol)) {
				results = append(results, symbolResultLocation(match, "fallback-scan"))
			}
		}
		if len(results) >= limit {
			truncated = true
			return errStopWalk
		}
		return ctx.Err()
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Path != results[j].Path {
			return results[i].Path < results[j].Path
		}
		return results[i].Range.Start.Line < results[j].Range.Start.Line
	})
	return results, filesScanned, truncated, err
}

func scanSymbolReferences(ctx context.Context, workspaceRoot string, symbol string, limit int) ([]domain.CodeLocation, bool, error) {
	results := []domain.CodeLocation{}
	truncated := false
	ignore := loadWorkspaceIgnore(ctx, workspaceRoot)
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	err := filepath.WalkDir(workspaceRoot, func(current string, entry os.DirEntry, walkErr error) error {
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
		rel := filepath.ToSlash(mustRel(workspaceRoot, current))
		if isSensitiveRelPath(rel) || !symbolFileSupported(rel) {
			return nil
		}
		fileResults := scanReferencesInFile(ctx, workspaceRoot, current, pattern, limit-len(results))
		results = append(results, fileResults...)
		if len(results) >= limit {
			truncated = true
			return errStopWalk
		}
		return ctx.Err()
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Path != results[j].Path {
			return results[i].Path < results[j].Path
		}
		return results[i].Range.Start.Line < results[j].Range.Start.Line
	})
	return results, truncated, err
}

func scanReferencesInFile(ctx context.Context, workspaceRoot string, path string, pattern *regexp.Regexp, remaining int) []domain.CodeLocation {
	if remaining <= 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > symbolSearchMaxBytes {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	language := symbolLanguage(strings.ToLower(filepath.Ext(path)))
	var out []domain.CodeLocation
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return out
		}
		lineNo++
		line := scanner.Text()
		indexes := pattern.FindAllStringIndex(line, -1)
		for _, index := range indexes {
			out = append(out, domain.CodeLocation{
				Path:     filepath.ToSlash(mustRel(workspaceRoot, path)),
				Range:    domain.SourceRange{Start: domain.SourcePosition{Line: lineNo, Character: index[0]}, End: domain.SourcePosition{Line: lineNo, Character: index[1]}},
				Language: language, Preview: strings.TrimSpace(line), Source: "fallback-scan",
			})
			if len(out) >= remaining {
				return out
			}
		}
	}
	return out
}

func symbolResultLocation(result symbolSearchResult, source string) domain.CodeLocation {
	return domain.CodeLocation{
		Path:     result.Path,
		Range:    domain.SourceRange{Start: domain.SourcePosition{Line: result.Line, Character: 0}, End: domain.SourcePosition{Line: result.Line, Character: len(result.Signature)}},
		Language: result.Language, Preview: result.Signature, Source: source,
	}
}
