package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

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
