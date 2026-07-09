package app

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func normalizeSandboxMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "", "foreground":
		return "foreground"
	case "background":
		return "background"
	case "pty":
		return "pty"
	default:
		return strings.TrimSpace(mode)
	}
}

func clampCommandTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultCommandTimeout
	}
	if timeout > maxCommandTimeout {
		return maxCommandTimeout
	}
	return timeout
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" && !isSecretEnvName("SHELL") {
		return shell
	}
	return "/bin/sh"
}

func SanitizedEnvironment(workspaceRoot string, allowlist []string, env map[string]string, overrides map[string]string) []string {
	allowed := map[string]bool{}
	for _, name := range defaultEnvAllowlist() {
		allowed[name] = true
	}
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = true
		}
	}
	envMap := map[string]string{}
	for _, item := range os.Environ() {
		name, value, ok := strings.Cut(item, "=")
		if !ok || !envNameAllowed(name, allowed) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	for name, value := range env {
		if !envNameAllowed(name, allowed) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	for name, value := range overrides {
		if !envOverrideKeyAllowed(name) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	envMap["AIVO_SANDBOX"] = "local"
	if _, ok := envMap["CI"]; !ok {
		envMap["CI"] = "1"
	}
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+envMap[key])
	}
	return out
}

func envOverrideKeyAllowed(name string) bool {
	switch strings.TrimSpace(name) {
	case "CI", "NODE_ENV", "GOFLAGS":
		return true
	case "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER":
		return true
	default:
		return false
	}
}

func defaultEnvAllowlist() []string {
	return []string{
		"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "TEMP", "TMP", "LANG", "TERM", "CI",
		"GOCACHE", "GOMODCACHE", "GOPATH", "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER",
	}
}

func envNameAllowed(name string, allowed map[string]bool) bool {
	if allowed[name] {
		return true
	}
	return strings.HasPrefix(name, "LC_")
}

func isSecretEnvName(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" {
		return false
	}
	for _, marker := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "COOKIE", "SESSION", "AUTH"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func isCacheEnvName(name string) bool {
	switch name {
	case "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER":
		return true
	default:
		return false
	}
}

func pathUnderHomeOrWorkspace(value string, workspaceRoot string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	target, err := filepath.Abs(value)
	if err != nil {
		return false
	}
	if home, err := os.UserHomeDir(); err == nil && pathHasPrefix(target, home) {
		return true
	}
	return pathHasPrefix(target, workspaceRoot)
}
