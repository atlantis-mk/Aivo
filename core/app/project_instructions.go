package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	projectInstructionFileMaxChars = 32 * 1024
	projectInstructionsMaxChars    = 64 * 1024
)

var projectInstructionNames = []string{"AGENTS.md", "CLAUDE.md"}

var configuredInstructionHTTPClient = &http.Client{Timeout: 3 * time.Second}

type projectInstructionSource struct {
	Path    string
	Label   string
	Content string
}

// resolveProjectInstructions returns bounded instruction text ordered from
// global to repository to the most-specific target directory. Target paths may
// be absolute or workspace-relative; paths outside the workspace are ignored.
func resolveProjectInstructions(projectRoot string, targetPaths []string) string {
	sources := collectProjectInstructionSources(projectRoot, targetPaths)
	if len(sources) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sources))
	used := 0
	for _, source := range sources {
		content := strings.TrimSpace(source.Content)
		if content == "" || used >= projectInstructionsMaxChars {
			continue
		}
		block := fmt.Sprintf("Source: %s\n%s", source.Label, content)
		remaining := projectInstructionsMaxChars - used
		if len(block) > remaining {
			block = block[:remaining]
		}
		parts = append(parts, block)
		used += len(block)
	}
	return strings.Join(parts, "\n\n")
}

func resolveConfiguredRuntimeInstructions(ctx context.Context, projectRoot string) string {
	var sources []projectInstructionSource
	seen := map[string]bool{}
	remoteCount := 0
	for _, raw := range loadEffectiveRuntimeConfig(projectRoot).Config.Instructions {
		if len(sources) >= 16 {
			break
		}
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if parsed, err := url.Parse(value); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
			if remoteCount >= 4 {
				continue
			}
			remoteCount++
			if seen[value] {
				continue
			}
			seen[value] = true
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, value, nil)
			if err != nil {
				continue
			}
			response, err := configuredInstructionHTTPClient.Do(request)
			if err != nil {
				continue
			}
			body, readErr := io.ReadAll(io.LimitReader(response.Body, projectInstructionFileMaxChars+1))
			_ = response.Body.Close()
			if readErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 || len(body) > projectInstructionFileMaxChars {
				continue
			}
			sources = append(sources, projectInstructionSource{Path: value, Label: value, Content: string(body)})
			continue
		}
		pattern := value
		if strings.HasPrefix(pattern, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				pattern = filepath.Join(home, strings.TrimPrefix(pattern, "~/"))
			}
		} else if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(projectRoot, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if len(sources) >= 16 {
				break
			}
			clean, err := filepath.Abs(match)
			if err != nil || seen[clean] {
				continue
			}
			seen[clean] = true
			info, err := os.Lstat(clean)
			if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > projectInstructionFileMaxChars {
				continue
			}
			body, err := os.ReadFile(clean)
			if err == nil {
				sources = append(sources, projectInstructionSource{Path: clean, Label: clean, Content: string(body)})
			}
		}
	}
	return renderProjectInstructionSources(sources)
}

func renderProjectInstructionSources(sources []projectInstructionSource) string {
	var parts []string
	used := 0
	for _, source := range sources {
		content := strings.TrimSpace(source.Content)
		if content == "" || used >= projectInstructionsMaxChars {
			continue
		}
		block := fmt.Sprintf("Source: %s\n%s", source.Label, content)
		if remaining := projectInstructionsMaxChars - used; len(block) > remaining {
			block = block[:remaining]
		}
		parts = append(parts, block)
		used += len(block)
	}
	return strings.Join(parts, "\n\n")
}

func collectProjectInstructionSources(projectRoot string, targetPaths []string) []projectInstructionSource {
	seen := map[string]bool{}
	var sources []projectInstructionSource
	appendFile := func(path string, label string) {
		clean := filepath.Clean(path)
		if seen[clean] {
			return
		}
		seen[clean] = true
		info, err := os.Lstat(clean)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return
		}
		raw, err := os.ReadFile(clean)
		if err != nil {
			return
		}
		if len(raw) > projectInstructionFileMaxChars {
			raw = raw[:projectInstructionFileMaxChars]
		}
		sources = append(sources, projectInstructionSource{Path: clean, Label: label, Content: string(raw)})
	}

	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		for _, dir := range []string{filepath.Join(home, ".config", "aivo"), filepath.Join(home, ".aivo")} {
			for _, name := range projectInstructionNames {
				appendFile(filepath.Join(dir, name), filepath.ToSlash(filepath.Join("~", strings.TrimPrefix(dir, home), name)))
			}
		}
	}

	root, err := filepath.Abs(strings.TrimSpace(projectRoot))
	if err != nil || root == "" {
		return sources
	}
	for _, name := range projectInstructionNames {
		appendFile(filepath.Join(root, name), name)
	}

	dirs := map[string]bool{}
	for _, target := range targetPaths {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		absTarget, err := filepath.Abs(target)
		if err != nil || !pathWithinRoot(root, absTarget) {
			continue
		}
		dir := absTarget
		if info, err := os.Stat(absTarget); err == nil && !info.IsDir() {
			dir = filepath.Dir(absTarget)
		} else if filepath.Ext(absTarget) != "" {
			dir = filepath.Dir(absTarget)
		}
		for pathWithinRoot(root, dir) && dir != root {
			dirs[dir] = true
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	orderedDirs := make([]string, 0, len(dirs))
	for dir := range dirs {
		orderedDirs = append(orderedDirs, dir)
	}
	sort.Slice(orderedDirs, func(i, j int) bool {
		depthI := strings.Count(filepath.Clean(orderedDirs[i]), string(filepath.Separator))
		depthJ := strings.Count(filepath.Clean(orderedDirs[j]), string(filepath.Separator))
		if depthI == depthJ {
			return orderedDirs[i] < orderedDirs[j]
		}
		return depthI < depthJ
	})
	for _, dir := range orderedDirs {
		rel, _ := filepath.Rel(root, dir)
		for _, name := range projectInstructionNames {
			appendFile(filepath.Join(dir, name), filepath.ToSlash(filepath.Join(rel, name)))
		}
	}
	return sources
}

func pathWithinRoot(root string, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func nestedProjectInstructionsForTarget(projectRoot, targetPath string) string {
	root, err := filepath.Abs(strings.TrimSpace(projectRoot))
	if err != nil || root == "" {
		return ""
	}
	if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
		root = resolved
	}
	var parts []string
	for _, source := range collectProjectInstructionSources(root, []string{targetPath}) {
		if !pathWithinRoot(root, source.Path) || filepath.Dir(source.Path) == root {
			continue
		}
		parts = append(parts, fmt.Sprintf("Source: %s\n%s", source.Label, strings.TrimSpace(source.Content)))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}
