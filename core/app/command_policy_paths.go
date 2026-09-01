package app

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func appendUniqueStrings(values []string, next ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values)+len(next))
	for _, value := range append(values, next...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func commandPathHints(tokens []string, cwd string, workspaceRoot string) ([]string, []string) {
	seenPaths := map[string]bool{}
	seenExternal := map[string]bool{}
	root, _ := filepath.Abs(strings.TrimSpace(workspaceRoot))
	base := cwd
	if strings.TrimSpace(base) == "" {
		base = root
	}
	for _, token := range tokens[1:] {
		if !looksLikePathToken(token) {
			continue
		}
		path := token
		if strings.Contains(path, "=") {
			continue
		}
		var abs string
		if filepath.IsAbs(path) {
			abs = filepath.Clean(path)
		} else {
			abs = filepath.Clean(filepath.Join(base, path))
		}
		if pathHasPrefix(abs, root) {
			rel, err := filepath.Rel(root, abs)
			if err == nil && rel != "." {
				seenPaths[filepath.ToSlash(rel)] = true
			}
			continue
		}
		if filepath.IsAbs(path) || strings.HasPrefix(filepath.Clean(path), "..") {
			seenExternal[filepath.ToSlash(path)] = true
		}
	}
	return sortedMapKeys(seenPaths), sortedMapKeys(seenExternal)
}

func looksLikePathToken(token string) bool {
	if strings.TrimSpace(token) == "" || strings.HasPrefix(token, "-") {
		return false
	}
	return strings.Contains(token, "/") || strings.HasPrefix(token, ".")
}

func commandApprovalKey(workspaceRoot string, cwd string, command string, argv []string, toolName string, backend string, sandboxProfile string, networkPolicy string, category string, riskLevel string, shell string, login bool, capabilities []string) string {
	caps := append([]string(nil), capabilities...)
	sort.Strings(caps)
	parts := []string{
		"workspace=" + normalizeStoredPathForKey(workspaceRoot),
		"cwd=" + normalizeStoredPathForKey(cwd),
		"command=" + command,
		"argv=" + strings.Join(argv, "\x00"),
		"tool=" + toolName,
		"backend=" + backend,
		"sandbox=" + firstNonEmpty(sandboxProfile, "default"),
		"shell=" + normalizeStoredPathForKey(shell),
		"login=" + strconv.FormatBool(login),
		"network=" + networkPolicy,
		"category=" + category,
		"risk=" + riskLevel,
		"capabilities=" + strings.Join(caps, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return "shell:" + hex.EncodeToString(sum[:])
}

func normalizeStoredPathForKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(value))
	}
	return filepath.ToSlash(filepath.Clean(abs))
}

func sortedMapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
