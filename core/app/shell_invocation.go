package app

import (
	"path/filepath"
	"runtime"
	"strings"
)

// shellCommandArgs uses each supported shell's native non-interactive command form.
// The command stays a single argv item so it is never re-parsed by the Go process launcher.
func shellCommandArgs(shell string, command string, login ...bool) []string {
	useLogin := len(login) > 0 && login[0]
	name := shellExecutableName(shell)
	switch name {
	case "cmd", "cmd.exe":
		return []string{"/d", "/s", "/c", command}
	case "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}
	default:
		if useLogin {
			return []string{"-lc", command}
		}
		return []string{"-c", command}
	}
}

func shellExecutableName(shell string) string {
	return strings.ToLower(filepath.Base(strings.ReplaceAll(strings.TrimSpace(shell), "\\", "/")))
}

func resolveLoginShell(input *bool, shell string) bool {
	if input != nil {
		return *input
	}
	return runtime.GOOS != "windows" && shellSupportsLogin(shell)
}

func shellSupportsLogin(shell string) bool {
	switch shellExecutableName(shell) {
	case "zsh", "zsh.exe", "bash", "bash.exe", "sh", "sh.exe":
		return true
	default:
		return false
	}
}
