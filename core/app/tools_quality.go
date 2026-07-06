package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"aivo/core/domain"
)

const qualityNamespaceDescription = "Code quality tools. Prefer read_diagnostics for declared diagnostics and format_code for supported formatter-backed file rewrites before falling back to bash."

type ReadDiagnosticsTool struct {
	workspaceRoot string
	runner        SandboxRunner
	outputSink    ShellOutputSink
}

func NewReadDiagnosticsTool(workspaceRoot string, runner SandboxRunner, outputSink ...ShellOutputSink) *ReadDiagnosticsTool {
	if runner == nil {
		runner = NewLocalSandboxRunner()
	}
	var sink ShellOutputSink
	if len(outputSink) > 0 {
		sink = outputSink[0]
	}
	return &ReadDiagnosticsTool{workspaceRoot: workspaceRoot, runner: runner, outputSink: sink}
}

func (t *ReadDiagnosticsTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "read_diagnostics",
		Description:          "Preferred tool for declared diagnostics. Run one diagnostics command and return parsed problems plus bounded command output. Use this after edits or while debugging to collect compiler, test, lint, or build issues without inventing shell commands or falling back to bash.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: qualityNamespaceDescription,
		Capability:           "shell.test",
		RiskLevel:            "medium",
		Category:             "diagnostics",
		Toolsets:             []string{"coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target":         map[string]any{"type": "string", "enum": []string{"core", "desktop", "all"}, "description": "Project target to diagnose. core runs Go tests, desktop runs lint or build, and all runs the repository diagnostics script."},
				"kind":           map[string]any{"type": "string", "enum": []string{"auto", "test", "lint", "build"}, "description": "Diagnostics kind. auto maps core to test and desktop to lint."},
				"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": int(maxCommandTimeout.Seconds())},
			},
		},
	}
}

func (t *ReadDiagnosticsTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseDiagnosticsArgs(args)
	if err != nil {
		return toolError("read_diagnostics", err)
	}
	command, err := diagnosticsCommand(input)
	if err != nil {
		return toolError("read_diagnostics", err)
	}
	prepared, err := prepareShellCommand(toolWorkspaceRoot(t.workspaceRoot, execCtx), execCtx, "read_diagnostics", command, "", input.TimeoutSeconds, "deny", "foreground", "", nil)
	if err != nil {
		return commandToolError("read_diagnostics", prepared, err)
	}
	prepared.request.OutputSink = t.outputSink
	result, runErr := t.runner.Run(ctx, prepared.request)
	toolResult := commandToolResult("read_diagnostics", prepared, result, runErr)
	problems := parseDiagnosticsProblems(result.Stdout + "\n" + result.Stderr)
	toolResult.Structured["target"] = input.Target
	toolResult.Structured["kind"] = input.Kind
	toolResult.Structured["problemCount"] = len(problems)
	toolResult.Structured["problems"] = problems
	if len(problems) > 0 {
		toolResult.Content += "\n\nProblems:\n" + diagnosticsProblemsContent(problems, 50)
		toolResult.ModelContent = toolResult.Content
	}
	return toolResult
}

type FormatCodeTool struct {
	workspaceRoot string
	runner        SandboxRunner
	outputSink    ShellOutputSink
}

func NewFormatCodeTool(workspaceRoot string, runner SandboxRunner, outputSink ...ShellOutputSink) *FormatCodeTool {
	if runner == nil {
		runner = NewLocalSandboxRunner()
	}
	var sink ShellOutputSink
	if len(outputSink) > 0 {
		sink = outputSink[0]
	}
	return &FormatCodeTool{workspaceRoot: workspaceRoot, runner: runner, outputSink: sink}
}

func (t *FormatCodeTool) Spec() domain.ToolSpec {
	return domain.ToolSpec{
		Name:                 "format_code",
		Description:          "Preferred tool for formatter-backed source rewrites. Format supported source files in place using project-local or standard formatters instead of falling back to bash. Supports Go/gofmt, TypeScript/JavaScript/CSS/Markdown/JSON/YAML/HTML via Prettier, optional project-local ESLint --fix for JavaScript/TypeScript, Rust/rustfmt, Python/black, and shell/shfmt. Returns changed file metadata and diffs.",
		Namespace:            filesystemNamespace,
		NamespaceDescription: qualityNamespaceDescription,
		Capability:           "filesystem.write",
		RiskLevel:            "medium",
		Category:             "formatter",
		Toolsets:             []string{"coding"},
		RequiresWorkspace:    true,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"paths":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Workspace-relative file paths to format. Must be supported text source files."},
				"eslintFix":      map[string]any{"type": "boolean", "description": "Also run project-local ESLint --fix for supported JavaScript/TypeScript files after Prettier. Uses node_modules/.bin/eslint or npx --no-install eslint only."},
				"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": int(maxCommandTimeout.Seconds())},
			},
			"required": []string{"paths"},
		},
	}
}

func (t *FormatCodeTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	input, err := parseFormatCodeArgs(args)
	if err != nil {
		return toolError("format_code", err)
	}
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	before := map[string]string{}
	targets := map[string]string{}
	for _, rel := range input.Paths {
		target, err := safeTargetForWrite(workspaceRoot, rel)
		if err != nil {
			return toolError("format_code", err)
		}
		if !formatPathSupported(rel) {
			return toolError("format_code", fmt.Errorf("unsupported formatter target %q", rel))
		}
		raw, err := os.ReadFile(target)
		if err != nil {
			return toolError("format_code", err)
		}
		before[rel] = string(raw)
		targets[rel] = target
	}
	plans := formatCommandPlans(workspaceRoot, input.Paths, input.ESLintFix)
	var toolResult domain.ToolResult
	commands := make([]map[string]any, 0, len(plans))
	var content strings.Builder
	for index, plan := range plans {
		prepared, err := prepareShellCommand(workspaceRoot, execCtx, "format_code", plan.Command, "", input.TimeoutSeconds, "deny", "foreground", "", nil)
		if err != nil {
			return commandToolError("format_code", prepared, err)
		}
		prepared.request.OutputSink = t.outputSink
		result, runErr := t.runner.Run(ctx, prepared.request)
		commandResult := commandToolResult("format_code", prepared, result, runErr)
		commands = append(commands, map[string]any{"formatter": plan.Formatter, "paths": plan.Paths, "command": plan.Command, "ok": commandResult.OK})
		if index > 0 {
			content.WriteString("\n\n")
		}
		content.WriteString(commandResult.Content)
		toolResult = commandResult
		if runErr != nil || result.ExitCode != 0 || result.TimedOut || result.Cancelled {
			toolResult.Structured["formatterCommands"] = commands
			toolResult.Content = content.String()
			toolResult.ModelContent = toolResult.Content
			return toolResult
		}
	}
	toolResult.Content = content.String()
	files := make([]domain.ToolResultFile, 0, len(input.Paths))
	for _, rel := range input.Paths {
		target := targets[rel]
		raw, err := os.ReadFile(target)
		if err != nil {
			return toolError("format_code", err)
		}
		after := string(raw)
		if before[rel] == after {
			continue
		}
		additions, deletions := countLineDelta(before[rel], after)
		currentHash, _, _ := fileHashIfExists(target)
		files = append(files, domain.ToolResultFile{
			Path:        rel,
			FullPath:    filepath.ToSlash(target),
			Type:        "format",
			Additions:   additions,
			Deletions:   deletions,
			Diff:        simpleFileDiff(rel, rel, before[rel], after),
			CurrentHash: currentHash,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	toolResult.Files = files
	toolResult.Structured["files"] = files
	toolResult.Structured["formattedPaths"] = input.Paths
	toolResult.Structured["formatterCommands"] = commands
	toolResult.Structured["changedFileCount"] = len(files)
	if len(files) == 0 {
		toolResult.Content += "\n\nNo formatting changes."
	} else {
		toolResult.Content += fmt.Sprintf("\n\nFormatted %d file(s).", len(files))
	}
	toolResult.ModelContent = toolResult.Content
	return toolResult
}

type diagnosticsInput struct {
	Target         string `json:"target"`
	Kind           string `json:"kind"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func parseDiagnosticsArgs(args json.RawMessage) (diagnosticsInput, error) {
	var input diagnosticsInput
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return input, errors.New("invalid read_diagnostics arguments")
		}
	}
	input.Target = firstNonEmpty(strings.TrimSpace(input.Target), "core")
	input.Kind = firstNonEmpty(strings.TrimSpace(input.Kind), "auto")
	switch input.Target {
	case "core", "desktop", "all":
	default:
		return input, fmt.Errorf("unsupported diagnostics target %q", input.Target)
	}
	switch input.Kind {
	case "auto", "test", "lint", "build":
	default:
		return input, fmt.Errorf("unsupported diagnostics kind %q", input.Kind)
	}
	return input, nil
}

func diagnosticsCommand(input diagnosticsInput) (string, error) {
	target := input.Target
	kind := input.Kind
	if kind == "auto" {
		if target == "all" {
			kind = "auto"
		} else if target == "desktop" {
			kind = "lint"
		} else {
			kind = "test"
		}
	}
	switch target + ":" + kind {
	case "core:test":
		return "npm run test:core", nil
	case "desktop:lint":
		return "npm run lint", nil
	case "desktop:build":
		return "npm run build", nil
	case "all:auto":
		return "npm run diagnostics", nil
	default:
		return "", fmt.Errorf("unsupported diagnostics combination target=%s kind=%s", target, kind)
	}
}

type formatCodeInput struct {
	Paths          []string `json:"paths"`
	ESLintFix      bool     `json:"eslintFix"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

func parseFormatCodeArgs(args json.RawMessage) (formatCodeInput, error) {
	var input formatCodeInput
	if err := json.Unmarshal(args, &input); err != nil {
		return input, errors.New("invalid format_code arguments")
	}
	seen := map[string]bool{}
	paths := make([]string, 0, len(input.Paths))
	for _, path := range input.Paths {
		clean := cleanPatchPath(path)
		if clean == "" || clean == "." {
			continue
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		paths = append(paths, clean)
	}
	if len(paths) == 0 {
		return input, errors.New("paths are required")
	}
	if len(paths) > 50 {
		return input, errors.New("format_code supports at most 50 paths per call")
	}
	input.Paths = paths
	return input, nil
}

func formatPathSupported(path string) bool {
	return formatterForPath(path) != ""
}

type formatCommandPlan struct {
	Formatter string
	Command   string
	Paths     []string
}

func formatCommandPlans(workspaceRoot string, paths []string, eslintFix bool) []formatCommandPlan {
	grouped := map[string][]string{}
	order := []string{}
	for _, path := range paths {
		formatter := formatterForPath(path)
		if formatter == "" {
			continue
		}
		if len(grouped[formatter]) == 0 {
			order = append(order, formatter)
		}
		grouped[formatter] = append(grouped[formatter], path)
	}
	plans := make([]formatCommandPlan, 0, len(order))
	for _, formatter := range order {
		formatterPaths := grouped[formatter]
		plans = append(plans, formatCommandPlan{
			Formatter: formatter,
			Command:   formatCommand(workspaceRoot, formatter, formatterPaths),
			Paths:     formatterPaths,
		})
	}
	if eslintFix {
		eslintPaths := eslintFixPaths(paths)
		if len(eslintPaths) > 0 {
			plans = append(plans, formatCommandPlan{
				Formatter: "eslint",
				Command:   formatCommand(workspaceRoot, "eslint", eslintPaths),
				Paths:     eslintPaths,
			})
		}
	}
	return plans
}

func formatterForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "gofmt"
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".json", ".jsonc", ".css", ".scss", ".md", ".mdx", ".yaml", ".yml", ".html", ".vue", ".svelte":
		return "prettier"
	case ".rs":
		return "rustfmt"
	case ".py":
		return "black"
	case ".sh", ".bash", ".zsh":
		return "shfmt"
	default:
		return ""
	}
}

func formatCommand(workspaceRoot string, formatter string, paths []string) string {
	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		quoted = append(quoted, shellQuote(path))
	}
	pathArgs := strings.Join(quoted, " ")
	switch formatter {
	case "gofmt":
		return "gofmt -w " + pathArgs
	case "prettier":
		if bin := workspaceLocalBin(workspaceRoot, "prettier"); bin != "" {
			return shellQuote(bin) + " --write " + pathArgs
		}
		return "npx --no-install prettier --write " + pathArgs
	case "rustfmt":
		return "rustfmt " + pathArgs
	case "black":
		if bin := workspaceVirtualEnvBin(workspaceRoot, "black"); bin != "" {
			return shellQuote(bin) + " " + pathArgs
		}
		return "python -m black " + pathArgs
	case "shfmt":
		return "shfmt -w " + pathArgs
	case "eslint":
		if bin := workspaceLocalBin(workspaceRoot, "eslint"); bin != "" {
			return shellQuote(bin) + " --fix " + pathArgs
		}
		return "npx --no-install eslint --fix " + pathArgs
	default:
		return ""
	}
}

func eslintFixPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if eslintFixPathSupported(path) {
			out = append(out, path)
		}
	}
	return out
}

func eslintFixPathSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".vue", ".svelte":
		return true
	default:
		return false
	}
}

func workspaceLocalBin(workspaceRoot string, name string) string {
	candidates := []string{
		filepath.Join(workspaceRoot, "node_modules", ".bin", name),
		filepath.Join(workspaceRoot, "node_modules", ".bin", name+".cmd"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			rel, err := filepath.Rel(workspaceRoot, candidate)
			if err == nil {
				return filepath.ToSlash(rel)
			}
		}
	}
	return ""
}

func workspaceVirtualEnvBin(workspaceRoot string, name string) string {
	candidates := []string{
		filepath.Join(workspaceRoot, ".venv", "bin", name),
		filepath.Join(workspaceRoot, "venv", "bin", name),
		filepath.Join(workspaceRoot, ".venv", "Scripts", name+".exe"),
		filepath.Join(workspaceRoot, "venv", "Scripts", name+".exe"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			rel, err := filepath.Rel(workspaceRoot, candidate)
			if err == nil {
				return filepath.ToSlash(rel)
			}
		}
	}
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func parseDiagnosticsProblems(output string) []map[string]any {
	patterns := []struct {
		re           *regexp.Regexp
		fileIndex    int
		lineIndex    int
		columnIndex  int
		messageIndex int
	}{
		{regexp.MustCompile(`(?m)([A-Za-z0-9_./\\-]+\.[A-Za-z0-9]+):(\d+)(?::(\d+))?:\s*(.+)`), 1, 2, 3, 4},
		{regexp.MustCompile(`(?m)([^:\n]+?\.[A-Za-z0-9]+)\((\d+),(\d+)\):\s*(.+)`), 1, 2, 3, 4},
	}
	problems := make([]map[string]any, 0)
	seen := map[string]bool{}
	for _, pattern := range patterns {
		matches := pattern.re.FindAllStringSubmatch(output, -1)
		for _, match := range matches {
			if len(match) <= pattern.messageIndex {
				continue
			}
			file := filepath.ToSlash(strings.TrimSpace(match[pattern.fileIndex]))
			line := atoiDefault(match[pattern.lineIndex], 0)
			column := atoiDefault(match[pattern.columnIndex], 0)
			message := strings.TrimSpace(match[pattern.messageIndex])
			if file == "" || line <= 0 || message == "" {
				continue
			}
			key := fmt.Sprintf("%s:%d:%d:%s", file, line, column, message)
			if seen[key] {
				continue
			}
			seen[key] = true
			problem := map[string]any{"file": file, "line": line, "message": message, "severity": diagnosticsSeverity(message)}
			if column > 0 {
				problem["column"] = column
			}
			problems = append(problems, problem)
			if len(problems) >= 200 {
				return problems
			}
		}
	}
	return problems
}

func diagnosticsSeverity(message string) string {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		return "warning"
	}
	return "error"
}

func diagnosticsProblemsContent(problems []map[string]any, limit int) string {
	if limit <= 0 || limit > len(problems) {
		limit = len(problems)
	}
	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		file, _ := problems[i]["file"].(string)
		message, _ := problems[i]["message"].(string)
		line, _ := problems[i]["line"].(int)
		column, _ := problems[i]["column"].(int)
		location := fmt.Sprintf("%s:%d", file, line)
		if column > 0 {
			location = fmt.Sprintf("%s:%d:%d", file, line, column)
		}
		lines = append(lines, location+" "+message)
	}
	if len(problems) > limit {
		lines = append(lines, fmt.Sprintf("... %d more problem(s)", len(problems)-limit))
	}
	return strings.Join(lines, "\n")
}

func atoiDefault(value string, fallback int) int {
	var out int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &out); err != nil {
		return fallback
	}
	return out
}
