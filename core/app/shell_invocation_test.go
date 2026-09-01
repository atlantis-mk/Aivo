package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestShellCommandArgsUseNativeWindowsSyntax(t *testing.T) {
	command := "echo ready"
	tests := []struct {
		shell string
		want  []string
	}{
		{shell: "/bin/bash", want: []string{"-c", command}},
		{shell: `C:\Program Files\Git\bin\bash.exe`, want: []string{"-c", command}},
		{shell: "powershell.exe", want: []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}},
		{shell: "pwsh.exe", want: []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}},
		{shell: "cmd.exe", want: []string{"/d", "/s", "/c", command}},
	}
	for _, test := range tests {
		t.Run(test.shell, func(t *testing.T) {
			if got := shellCommandArgs(test.shell, command); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("shellCommandArgs(%q) = %#v, want %#v", test.shell, got, test.want)
			}
		})
	}
}

func TestShellCommandArgsUseLoginShellForPOSIXShells(t *testing.T) {
	command := "echo ready"
	if got := shellCommandArgs("/bin/zsh", command, true); !reflect.DeepEqual(got, []string{"-lc", command}) {
		t.Fatalf("shellCommandArgs login = %#v, want -lc", got)
	}
	if got := shellCommandArgs("powershell.exe", command, true); !reflect.DeepEqual(got, []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}) {
		t.Fatalf("PowerShell login args = %#v, want native command form", got)
	}
}

func TestResolveLoginShellDefaultsToLoginForPOSIXShells(t *testing.T) {
	if !resolveLoginShell(nil, "/bin/zsh") {
		t.Fatal("omitted login should default to login shell for zsh")
	}
	disabled := false
	if resolveLoginShell(&disabled, "/bin/zsh") {
		t.Fatal("explicit login=false should disable login shell")
	}
	enabled := true
	if !resolveLoginShell(&enabled, "/bin/zsh") {
		t.Fatal("explicit login=true should enable login shell")
	}
	if resolveLoginShell(nil, "cmd.exe") {
		t.Fatal("cmd.exe should not default to login shell")
	}
}

func TestWindowsShellCandidatesPreferPowerShellThenCommandPrompt(t *testing.T) {
	want := []string{"powershell.exe", "pwsh.exe", "cmd.exe"}
	if got := windowsShellCandidates(); !reflect.DeepEqual(got, want) {
		t.Fatalf("windowsShellCandidates() = %#v, want %#v", got, want)
	}
}

func TestResolveCommandShellRejectsUnsupportedExplicitShell(t *testing.T) {
	root := t.TempDir()
	unsupported := filepath.Join(root, "fish")
	if err := os.WriteFile(unsupported, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveCommandShell(root, unsupported); err == nil {
		t.Fatal("unsupported explicit shell should be rejected")
	}
}

func TestShellRuntimeInstructionUsesTheResolvedShellSemantics(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{shell: "powershell.exe", want: "PowerShell syntax only"},
		{shell: "cmd.exe", want: "cmd.exe syntax only"},
		{shell: "/bin/zsh", want: "unmatched globs fail before the command runs"},
		{shell: "/bin/bash", want: "Write Bash syntax"},
		{shell: "/bin/sh", want: "POSIX sh syntax"},
	}
	for _, test := range tests {
		t.Run(test.shell, func(t *testing.T) {
			if got := shellRuntimeInstructionForExecutable(test.shell); !strings.Contains(got, test.want) {
				t.Fatalf("shell instruction = %q, want %q", got, test.want)
			}
		})
	}
}
