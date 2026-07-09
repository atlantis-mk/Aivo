package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
)

func normalizeTerminalSize(rows int, cols int) (int, int) {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	if rows < 4 {
		rows = 4
	}
	if rows > 200 {
		rows = 200
	}
	if cols < 20 {
		cols = 20
	}
	if cols > 400 {
		cols = 400
	}
	return rows, cols
}

func terminalWorkspaceRoot(workspaceRoot string) (string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return realRoot, nil
}

func normalizeTerminalCWD(workspaceRoot string, cwd string) (string, string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root != "" {
		return normalizeSandboxCWD(root, cwd, false)
	}
	base, err := defaultTerminalCWD()
	if err != nil {
		return "", "", err
	}
	cleanCWD := strings.TrimSpace(cwd)
	if cleanCWD == "" || cleanCWD == "." {
		return "", base, nil
	}
	target := cleanCWD
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, filepath.Clean(target))
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", "", err
	}
	return "", realTarget, nil
}

func defaultTerminalCWD() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		realHome, evalErr := filepath.EvalSymlinks(home)
		if evalErr == nil {
			return realHome, nil
		}
		return filepath.Abs(home)
	}
	return os.Getwd()
}

func terminalEnvironment(workspaceRoot string, env map[string]string) []string {
	safe := SanitizedEnvironment(workspaceRoot, append(defaultEnvAllowlist(), "AIVO_TERMINAL"), nil, env)
	safe = append(safe, "AIVO_TERMINAL=1", "TERM=xterm-256color")
	return safe
}

func startTerminalPTY(preferredShell string, cwd string, env []string, rows int, cols int) (*exec.Cmd, *os.File, string, error) {
	shells := []string{strings.TrimSpace(preferredShell)}
	if runtimeShell := strings.TrimSpace(os.Getenv("SHELL")); runtimeShell != "" {
		shells = append(shells, runtimeShell)
	}
	shells = append(shells, "/bin/zsh", "/bin/sh")
	shells = uniqueNonEmptyStrings(shells)
	var failures []string
	cmd, ptmx, shell, err := startTerminalPTYInCWD(shells, cwd, env, rows, cols, &failures)
	if err == nil {
		return cmd, ptmx, shell, nil
	}
	if terminalStartMayBeCWDPermission(failures) {
		if home, homeErr := os.UserHomeDir(); homeErr == nil && home != "" && home != cwd {
			cmd, ptmx, shell, err = startTerminalPTYInCWD(shells, home, env, rows, cols, &failures)
			if err == nil {
				_, _ = ptmx.Write([]byte("cd " + shellSingleQuote(cwd) + "\n"))
				return cmd, ptmx, shell, nil
			}
		}
	}
	if len(failures) == 0 {
		failures = append(failures, "no terminal shell candidates")
	}
	return nil, nil, "", fmt.Errorf("start terminal shell failed; attempted %s", strings.Join(failures, "; "))
}

func startTerminalPTYInCWD(shells []string, cwd string, env []string, rows int, cols int, failures *[]string) (*exec.Cmd, *os.File, string, error) {
	var lastErr error
	for _, shell := range shells {
		if err := executableShellAvailable(shell); err != nil {
			*failures = append(*failures, fmt.Sprintf("%s in %s: %v", shell, cwd, err))
			lastErr = err
			continue
		}
		cmd := exec.Command(shell)
		cmd.Dir = cwd
		cmd.Env = env
		ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		if err == nil {
			return cmd, ptmx, shell, nil
		}
		lastErr = err
		*failures = append(*failures, fmt.Sprintf("%s in %s: %v", shell, cwd, err))
	}
	if lastErr == nil {
		lastErr = errors.New("no terminal shell candidates")
	}
	return nil, nil, "", lastErr
}

func terminalStartMayBeCWDPermission(failures []string) bool {
	for _, failure := range failures {
		if strings.Contains(strings.ToLower(failure), "operation not permitted") {
			return true
		}
	}
	return false
}

func executableShellAvailable(shell string) error {
	info, err := os.Stat(shell)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("is not executable")
	}
	return nil
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func killTerminalProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	groupErr := killProcessGroup(process)
	directErr := process.Kill()
	if groupErr != nil {
		return groupErr
	}
	if directErr != nil && !errors.Is(directErr, os.ErrProcessDone) {
		return directErr
	}
	return nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
