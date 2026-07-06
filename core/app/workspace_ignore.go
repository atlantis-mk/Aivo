package app

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
)

type workspaceIgnore struct {
	root  string
	rules []workspaceIgnoreRule
}

type workspaceIgnoreRule struct {
	base     string
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool
	hasSlash bool
}

func loadWorkspaceIgnore(ctx context.Context, workspaceRoot string) workspaceIgnore {
	root, err := filepath.Abs(strings.TrimSpace(workspaceRoot))
	if err != nil || root == "" {
		return workspaceIgnore{}
	}
	root, _ = filepath.EvalSymlinks(root)
	ignore := workspaceIgnore{root: root}
	_ = filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if current != root && ignoredToolDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != ".gitignore" {
			return nil
		}
		base := filepath.ToSlash(mustRel(root, filepath.Dir(current)))
		if base == "." {
			base = ""
		}
		ignore.rules = append(ignore.rules, parseWorkspaceIgnoreFile(base, current)...)
		return nil
	})
	return ignore
}

func parseWorkspaceIgnoreFile(base string, path string) []workspaceIgnoreRule {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var rules []workspaceIgnoreRule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		negated := strings.HasPrefix(raw, "!")
		if negated {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "!"))
		}
		if raw == "" {
			continue
		}
		raw = filepath.ToSlash(raw)
		anchored := strings.HasPrefix(raw, "/")
		raw = strings.TrimPrefix(raw, "/")
		dirOnly := strings.HasSuffix(raw, "/")
		raw = strings.TrimSuffix(raw, "/")
		if raw == "" {
			continue
		}
		rules = append(rules, workspaceIgnoreRule{
			base:     filepath.ToSlash(strings.Trim(base, "/")),
			pattern:  raw,
			negated:  negated,
			dirOnly:  dirOnly,
			anchored: anchored,
			hasSlash: strings.Contains(raw, "/"),
		})
	}
	return rules
}

func (ignore workspaceIgnore) ignored(rel string, isDir bool) bool {
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if rel == "." || rel == "" || len(ignore.rules) == 0 {
		return false
	}
	ignored := false
	for _, rule := range ignore.rules {
		if rule.matches(rel, isDir) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func (rule workspaceIgnoreRule) matches(rel string, isDir bool) bool {
	local := rel
	if rule.base != "" {
		if rel == rule.base {
			local = ""
		} else if strings.HasPrefix(rel, rule.base+"/") {
			local = strings.TrimPrefix(rel, rule.base+"/")
		} else {
			return false
		}
	}
	if local == "" {
		return false
	}
	if rule.dirOnly && !isDir {
		return false
	}
	pattern := rule.pattern
	if rule.anchored || rule.hasSlash {
		return globMatch(pattern, local)
	}
	parts := strings.Split(local, "/")
	for _, part := range parts {
		if globMatch(pattern, part) {
			return true
		}
	}
	return false
}

func globMatch(pattern string, value string) bool {
	if ok, err := filepath.Match(pattern, value); err == nil && ok {
		return true
	}
	if strings.Contains(pattern, "**") {
		matcher, err := compileGlobMatcher(pattern)
		return err == nil && matcher.MatchString(value)
	}
	return false
}

func shouldSkipWorkspaceEntry(workspaceRoot string, current string, entry os.DirEntry, ignore workspaceIgnore) bool {
	if entry.IsDir() && ignoredToolDirs[entry.Name()] {
		return true
	}
	rel := filepath.ToSlash(mustRel(workspaceRoot, current))
	if rel == "." {
		return false
	}
	return ignore.ignored(rel, entry.IsDir())
}
