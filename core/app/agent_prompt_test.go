package app

import (
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestAgentSystemPromptInjectsResolvedShellRuntime(t *testing.T) {
	prompt := buildAgentSystemPromptWithSnapshotAndShell(domain.AgentModeDefinition{
		DisplayName: "Assistant",
		Prompt:      "Use the available tools.",
	}, PromptSnapshot{}, "The `exec_command` tool executes through PowerShell in this environment. Write PowerShell syntax only.")
	if !strings.Contains(prompt, `<global name="shell_runtime">`) || !strings.Contains(prompt, "PowerShell syntax only") {
		t.Fatalf("system prompt missing shell runtime instruction: %q", prompt)
	}
}

func TestAssistantBuiltinPromptRequiresCompleteFilePaths(t *testing.T) {
	prompt := builtinPromptBody("agent.assistant")
	for _, want := range []string{
		"group workspace-relative paths separately from host-native absolute paths",
		"complete workspace-relative path",
		"do not omit parent directories for sibling files",
		"bare basename unless the file is actually at that path root",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("assistant prompt missing %q: %q", want, prompt)
		}
	}
}

func TestAssistantBuiltinPromptConstrainsWorkspaceSearch(t *testing.T) {
	prompt := builtinPromptBody("agent.assistant")
	for _, want := range []string{
		"Treat the current workspace as the default filesystem boundary",
		"Do not generate or run commands that traverse outside the workspace",
		"find ..",
		"search from `.` or from the known workspace root",
		"Do not use parent-directory search as a convenience fallback",
		"find . -path '*/videospec/config.json' -print -quit",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("assistant prompt missing %q: %q", want, prompt)
		}
	}
}
