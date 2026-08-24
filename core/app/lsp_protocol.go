package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		rel, err := filepath.Rel(canonicalLSPPath(workspaceRoot), canonicalLSPPath(path))
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
	rel, err := filepath.Rel(canonicalLSPPath(workspaceRoot), canonicalLSPPath(path))
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

func canonicalLSPPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
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
	for _, resolved := range resolvedLSPCatalog(root) {
		if resolved.Definition.Disabled || resolved.Definition.StrictRoot {
			continue
		}
		// Root markers select a server root after a language has been detected;
		// generic markers such as .git must not classify every repository as shell.
		if hasWorkspaceLSPFile(ctx, root, resolved.Definition) {
			language := firstLSPDefinitionLanguage(resolved.Definition)
			if language != "" {
				return language, resolved.Name, true
			}
		}
	}
	return "", "", false
}

func hasWorkspaceLSPFile(ctx context.Context, root string, definition domain.LanguageServerDefinition) bool {
	extensions := map[string]bool{}
	filenames := map[string]bool{}
	for _, extension := range definition.Extensions {
		extensions[strings.ToLower(extension)] = true
	}
	for _, filename := range definition.Filenames {
		filenames[strings.ToLower(filename)] = true
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
		if !isSensitiveRelPath(rel) && (extensions[strings.ToLower(filepath.Ext(current))] || filenames[strings.ToLower(entry.Name())]) {
			found = true
			return errStopWalk
		}
		return ctx.Err()
	})
	return found
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
	if resolved, ok := resolveLSPDefinitionForPath(filepath.Dir(path), path); ok {
		return languageIDForLSPDefinition(resolved.Definition, path)
	}
	return builtinLSPLanguageForPath(path)
}

func builtinLSPLanguageForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py", ".pyi":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".lua":
		return "lua"
	case ".sh", ".bash", ".zsh":
		return "shellscript"
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
	case "typescript", "typescriptreact":
		if strings.EqualFold(filepath.Ext(path), ".tsx") {
			return "typescriptreact"
		}
		return "typescript"
	default:
		return language
	}
}

func lspCommandForLanguage(language string) (string, []string, string, bool) {
	if resolved, ok := resolveLSPDefinitionForLanguage("", language); ok {
		return resolved.Definition.Command, resolved.Definition.Args, resolved.Name, true
	}
	return "", nil, "", false
}

type resolvedLSPDefinition struct {
	Name       string
	Definition domain.LanguageServerDefinition
	Revision   string
}

func resolvedLSPCatalog(root string) []resolvedLSPDefinition {
	definitions := builtInLSPDefinitions()
	if strings.TrimSpace(root) != "" {
		for name, definition := range loadEffectiveRuntimeConfig(root).Config.LanguageServers {
			definitions[name] = definition
		}
	}
	names := make([]string, 0, len(definitions))
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]resolvedLSPDefinition, 0, len(names))
	for _, name := range names {
		definition := definitions[name]
		raw, _ := json.Marshal(definition)
		sum := sha256.Sum256(raw)
		out = append(out, resolvedLSPDefinition{Name: name, Definition: definition, Revision: hex.EncodeToString(sum[:])})
	}
	return out
}

func builtInLSPDefinitions() map[string]domain.LanguageServerDefinition {
	return map[string]domain.LanguageServerDefinition{
		"astro-ls":                   {LanguageIDs: []string{"astro"}, Extensions: []string{".astro"}, RootMarkers: []string{"astro.config.mjs", "astro.config.ts", "package.json"}, Command: "astro-ls", Args: []string{"--stdio"}},
		"bash-language-server":       {LanguageIDs: []string{"shellscript"}, Extensions: []string{".sh", ".bash", ".zsh"}, RootMarkers: []string{".git"}, Command: "bash-language-server", Args: []string{"start"}},
		"basedpyright":               {LanguageIDs: []string{"python"}, Extensions: []string{".py", ".pyi"}, RootMarkers: []string{"pyproject.toml", "setup.py", "requirements.txt"}, Command: "basedpyright-langserver", Args: []string{"--stdio"}},
		"clangd":                     {LanguageIDs: []string{"c", "cpp"}, Extensions: []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx"}, RootMarkers: []string{"compile_commands.json", "CMakeLists.txt"}, Command: "clangd"},
		"clojure-lsp":                {LanguageIDs: []string{"clojure"}, Extensions: []string{".clj", ".cljs", ".cljc", ".edn"}, RootMarkers: []string{"deps.edn", "project.clj", "shadow-cljs.edn"}, Command: "clojure-lsp"},
		"csharp-ls":                  {LanguageIDs: []string{"csharp"}, Extensions: []string{".cs"}, RootMarkers: []string{"*.sln", "*.csproj"}, Command: "csharp-ls"},
		"dart-language-server":       {LanguageIDs: []string{"dart"}, Extensions: []string{".dart"}, RootMarkers: []string{"pubspec.yaml"}, Command: "dart", Args: []string{"language-server", "--protocol=lsp"}},
		"deno":                       {LanguageIDs: []string{"typescript", "typescriptreact", "javascript", "javascriptreact"}, Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs"}, RootMarkers: []string{"deno.json", "deno.jsonc"}, StrictRoot: true, Command: "deno", Args: []string{"lsp"}},
		"dockerfile-language-server": {LanguageIDs: []string{"dockerfile"}, Filenames: []string{"Dockerfile", "Containerfile"}, RootMarkers: []string{".git"}, Command: "docker-langserver", Args: []string{"--stdio"}},
		"elixir-ls":                  {LanguageIDs: []string{"elixir"}, Extensions: []string{".ex", ".exs"}, RootMarkers: []string{"mix.exs"}, Command: "elixir-ls"},
		"fsautocomplete":             {LanguageIDs: []string{"fsharp"}, Extensions: []string{".fs", ".fsi", ".fsx", ".fsscript"}, RootMarkers: []string{"*.sln", "*.fsproj"}, Command: "fsautocomplete", Args: []string{"--adaptive-lsp-server-enabled"}},
		"gleam":                      {LanguageIDs: []string{"gleam"}, Extensions: []string{".gleam"}, RootMarkers: []string{"gleam.toml"}, Command: "gleam", Args: []string{"lsp"}},
		"gopls":                      {LanguageIDs: []string{"go"}, Extensions: []string{".go"}, RootMarkers: []string{"go.mod", "go.work"}, Command: "gopls"},
		"haskell-language-server":    {LanguageIDs: []string{"haskell"}, Extensions: []string{".hs", ".lhs"}, RootMarkers: []string{"hie.yaml", "stack.yaml", "cabal.project", "*.cabal"}, Command: "haskell-language-server-wrapper", Args: []string{"--lsp"}},
		"intelephense":               {LanguageIDs: []string{"php"}, Extensions: []string{".php"}, RootMarkers: []string{"composer.json"}, Command: "intelephense", Args: []string{"--stdio"}},
		"jdtls":                      {LanguageIDs: []string{"java"}, Extensions: []string{".java"}, RootMarkers: []string{"pom.xml", "build.gradle", "build.gradle.kts"}, Command: "jdtls"},
		"julia-language-server":      {LanguageIDs: []string{"julia"}, Extensions: []string{".jl"}, RootMarkers: []string{"Project.toml", "JuliaProject.toml"}, Command: "julia", Args: []string{"--startup-file=no", "--history-file=no", "-e", "using LanguageServer; runserver()"}},
		"kotlin-language-server":     {LanguageIDs: []string{"kotlin"}, Extensions: []string{".kt", ".kts"}, RootMarkers: []string{"settings.gradle", "settings.gradle.kts", "pom.xml"}, Command: "kotlin-language-server"},
		"lua-language-server":        {LanguageIDs: []string{"lua"}, Extensions: []string{".lua"}, RootMarkers: []string{".luarc.json", ".luarc.jsonc"}, Command: "lua-language-server"},
		"nixd":                       {LanguageIDs: []string{"nix"}, Extensions: []string{".nix"}, RootMarkers: []string{"flake.nix", "shell.nix", ".git"}, Command: "nixd"},
		"ocamllsp":                   {LanguageIDs: []string{"ocaml"}, Extensions: []string{".ml", ".mli"}, RootMarkers: []string{"dune-project", "dune-workspace", "*.opam"}, Command: "ocamllsp"},
		"prisma-language-server":     {LanguageIDs: []string{"prisma"}, Extensions: []string{".prisma"}, RootMarkers: []string{"schema.prisma", "package.json"}, Command: "prisma-language-server", Args: []string{"--stdio"}},
		"razor-language-server":      {LanguageIDs: []string{"razor"}, Extensions: []string{".razor", ".cshtml"}, RootMarkers: []string{"*.sln", "*.csproj"}, Command: "rzls"},
		"ruby-lsp":                   {LanguageIDs: []string{"ruby"}, Extensions: []string{".rb"}, RootMarkers: []string{"Gemfile", ".ruby-version"}, Command: "ruby-lsp"},
		"rust-analyzer":              {LanguageIDs: []string{"rust"}, Extensions: []string{".rs"}, RootMarkers: []string{"Cargo.toml"}, Command: "rust-analyzer"},
		"sourcekit-lsp":              {LanguageIDs: []string{"swift"}, Extensions: []string{".swift"}, RootMarkers: []string{"Package.swift", "*.xcodeproj", "*.xcworkspace"}, Command: "sourcekit-lsp"},
		"svelte-language-server":     {LanguageIDs: []string{"svelte"}, Extensions: []string{".svelte"}, RootMarkers: []string{"svelte.config.js", "svelte.config.ts", "package.json"}, Command: "svelteserver", Args: []string{"--stdio"}},
		"terraform-ls":               {LanguageIDs: []string{"terraform"}, Extensions: []string{".tf", ".tfvars"}, RootMarkers: []string{".terraform", ".git"}, Command: "terraform-ls", Args: []string{"serve"}},
		"texlab":                     {LanguageIDs: []string{"latex"}, Extensions: []string{".tex", ".bib"}, RootMarkers: []string{".latexmkrc", "latexmkrc", ".git"}, Command: "texlab"},
		"tinymist":                   {LanguageIDs: []string{"typst"}, Extensions: []string{".typ"}, RootMarkers: []string{"typst.toml", ".git"}, Command: "tinymist"},
		"typescript-language-server": {LanguageIDs: []string{"typescript", "typescriptreact", "javascript", "javascriptreact"}, Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".mts", ".cts"}, RootMarkers: []string{"tsconfig.json", "jsconfig.json", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb"}, Command: "typescript-language-server", Args: []string{"--stdio"}},
		"vue-language-server":        {LanguageIDs: []string{"vue"}, Extensions: []string{".vue"}, RootMarkers: []string{"package.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock"}, Command: "vue-language-server", Args: []string{"--stdio"}},
		"yaml-language-server":       {LanguageIDs: []string{"yaml"}, Extensions: []string{".yaml", ".yml"}, RootMarkers: []string{".git"}, Command: "yaml-language-server", Args: []string{"--stdio"}},
		"zls":                        {LanguageIDs: []string{"zig"}, Extensions: []string{".zig", ".zon"}, RootMarkers: []string{"build.zig", "build.zig.zon"}, Command: "zls"},
	}
}

func resolveLSPDefinitionForLanguage(root string, language string) (resolvedLSPDefinition, bool) {
	for _, resolved := range resolvedLSPCatalog(root) {
		if resolved.Definition.Disabled || resolved.Definition.StrictRoot {
			continue
		}
		for _, candidate := range resolved.Definition.LanguageIDs {
			if candidate == language || (language == "javascript" && candidate == "typescript") {
				return resolved, true
			}
		}
	}
	return resolvedLSPDefinition{}, false
}

func resolveLSPDefinitionForPath(root string, path string) (resolvedLSPDefinition, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	filename := strings.ToLower(filepath.Base(path))
	var fallback resolvedLSPDefinition
	var best resolvedLSPDefinition
	bestDepth := -1
	for _, resolved := range resolvedLSPCatalog(root) {
		if resolved.Definition.Disabled {
			continue
		}
		matchedFile := false
		for _, candidate := range resolved.Definition.Extensions {
			if strings.EqualFold(candidate, ext) {
				matchedFile = true
				break
			}
		}
		if !matchedFile {
			for _, candidate := range resolved.Definition.Filenames {
				if strings.EqualFold(candidate, filename) {
					matchedFile = true
					break
				}
			}
		}
		if !matchedFile {
			continue
		}
		if strings.TrimSpace(root) != "" {
			if markerRoot, ok := nearestLSPRootMatch(root, path, resolved.Definition.RootMarkers); ok {
				depth := len(strings.Split(filepath.Clean(markerRoot), string(os.PathSeparator)))
				if depth > bestDepth {
					best, bestDepth = resolved, depth
				}
				continue
			}
		}
		if !resolved.Definition.StrictRoot && fallback.Name == "" {
			fallback = resolved
		}
	}
	if best.Name != "" {
		return best, true
	}
	if fallback.Name != "" {
		return fallback, true
	}
	return resolvedLSPDefinition{}, false
}

func firstLSPDefinitionLanguage(definition domain.LanguageServerDefinition) string {
	if len(definition.LanguageIDs) == 0 {
		return ""
	}
	return definition.LanguageIDs[0]
}

func languageIDForLSPDefinition(definition domain.LanguageServerDefinition, path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".cc" || ext == ".cpp" || ext == ".cxx" || ext == ".hh" || ext == ".hpp" || ext == ".hxx" {
		for _, language := range definition.LanguageIDs {
			if language == "cpp" {
				return language
			}
		}
	}
	for _, language := range definition.LanguageIDs {
		if ext == ".tsx" && language == "typescriptreact" || ext == ".jsx" && language == "javascriptreact" {
			return language
		}
	}
	return firstLSPDefinitionLanguage(definition)
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
