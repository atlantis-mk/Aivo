package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aivo/core/domain"
)

const qualityNamespaceDescription = "Code quality tools. Prefer read_diagnostics for declared diagnostics and format_code for supported formatter-backed file rewrites before falling back to exec_command."

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
		Description:          "Preferred tool for declared diagnostics. Run one diagnostics command and return parsed problems plus bounded command output. Use this after edits or while debugging to collect compiler, test, lint, or build issues without inventing shell commands or falling back to exec_command.",
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
	workspaceRoot := toolWorkspaceRoot(t.workspaceRoot, execCtx)
	shell, err := resolveCommandShell(workspaceRoot, "")
	if err != nil {
		return primitiveError("read_diagnostics", "bash_unavailable", err)
	}
	prepared, err := prepareShellCommand(workspaceRoot, execCtx, "read_diagnostics", command, "", input.TimeoutSeconds, "deny", "foreground", "", shell, false, nil)
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
		Description:          "Preferred tool for formatter-backed source rewrites. Format supported source files in place using project-local or standard formatters instead of falling back to exec_command. Supports Go/gofmt, TypeScript/JavaScript/CSS/Markdown/JSON/YAML/HTML via Prettier, optional project-local ESLint --fix for JavaScript/TypeScript, Rust/rustfmt, Python/black, and shell/shfmt. Returns changed file metadata and diffs.",
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
	shell, err := resolveCommandShell(workspaceRoot, "")
	if err != nil {
		return primitiveError("format_code", "bash_unavailable", err)
	}
	var toolResult domain.ToolResult
	commands := make([]map[string]any, 0, len(plans))
	var content strings.Builder
	for index, plan := range plans {
		prepared, err := prepareShellCommand(workspaceRoot, execCtx, "format_code", plan.Command, "", input.TimeoutSeconds, "deny", "foreground", "", shell, false, nil)
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

type formatCodeInput struct {
	Paths          []string `json:"paths"`
	ESLintFix      bool     `json:"eslintFix"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

type formatCommandPlan struct {
	Formatter string
	Command   string
	Paths     []string
}
