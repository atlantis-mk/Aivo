package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

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
