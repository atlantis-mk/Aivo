package app

import (
	"fmt"
	"strings"

	"aivo/core/domain"
)

type promptInjectionScope string

const (
	promptInjectionScopeDefault promptInjectionScope = "default"
	promptInjectionScopeGlobal  promptInjectionScope = "global"
)

type promptInjection struct {
	Scope promptInjectionScope
	Name  string
	Text  string
}

func agentPromptInjections(modeDef domain.AgentModeDefinition) []promptInjection {
	return agentPromptInjectionsWithSnapshot(modeDef, PromptSnapshot{})
}

func agentPromptInjectionsWithSnapshot(modeDef domain.AgentModeDefinition, snapshot PromptSnapshot) []promptInjection {
	toolProtocol := snapshot.Body("protocol.tool")
	if toolProtocol == "" {
		toolProtocol = builtinPromptBody("protocol.tool")
	}
	injections := []promptInjection{
		{
			Scope: promptInjectionScopeDefault,
			Name:  "agent_mode",
			Text:  "Agent mode: " + modeDef.DisplayName + "\n\n" + modeDef.Prompt,
		},
		{
			Scope: promptInjectionScopeGlobal,
			Name:  "tool_protocol",
			Text:  toolProtocol,
		},
	}
	if len(modeDef.Subagents) > 0 {
		quoted := make([]string, 0, len(modeDef.Subagents))
		for _, subagent := range modeDef.Subagents {
			quoted = append(quoted, fmt.Sprintf("`%s`", subagent))
		}
		subagents := strings.Join(quoted, ", ")
		injections = append(injections, promptInjection{
			Scope: promptInjectionScopeDefault,
			Name:  "associated_subagents",
			Text:  associatedSubagentsInstruction(subagents),
		})
	}
	return injections
}

func associatedSubagentsInstruction(subagents string) string {
	return "Associated subagents available to this mode: " + subagents + ". When the current task has a bounded independent part that benefits from one of these modes, you may call agent_delegate_task and then use its returned result. Decide based on the task; do not delegate routine work or fan out solely because associations exist. Never name an unlisted mode."
}

func buildAgentSystemPrompt(modeDef domain.AgentModeDefinition) string {
	return buildAgentSystemPromptWithSnapshot(modeDef, PromptSnapshot{})
}

func buildAgentSystemPromptWithSnapshot(modeDef domain.AgentModeDefinition, snapshot PromptSnapshot) string {
	return buildAgentSystemPromptWithSnapshotAndShell(modeDef, snapshot, "")
}

func buildAgentSystemPromptWithSnapshotAndShell(modeDef domain.AgentModeDefinition, snapshot PromptSnapshot, shellInstruction string) string {
	injections := agentPromptInjectionsWithSnapshot(modeDef, snapshot)
	if text := strings.TrimSpace(shellInstruction); text != "" {
		injections = append(injections, promptInjection{
			Scope: promptInjectionScopeGlobal,
			Name:  "shell_runtime",
			Text:  text,
		})
	}
	parts := make([]string, 0, len(injections))
	for _, injection := range injections {
		text := strings.TrimSpace(injection.Text)
		if text == "" {
			continue
		}
		parts = append(parts, renderPromptInjection(injection.Scope, injection.Name, text))
	}
	if len(parts) == 0 {
		return ""
	}
	return "<agent_instructions>\n" + strings.Join(parts, "\n\n") + "\n</agent_instructions>"
}

func renderPromptInjection(scope promptInjectionScope, name string, text string) string {
	var builder strings.Builder
	builder.WriteString("<")
	builder.WriteString(string(scope))
	if strings.TrimSpace(name) != "" {
		builder.WriteString(` name="`)
		builder.WriteString(strings.TrimSpace(name))
		builder.WriteString(`"`)
	}
	builder.WriteString(">\n")
	builder.WriteString(strings.TrimSpace(text))
	builder.WriteString("\n</")
	builder.WriteString(string(scope))
	builder.WriteString(">")
	return builder.String()
}

func prependAgentSystemPrompt(messages []domain.ChatMessage, modeDef domain.AgentModeDefinition) []domain.ChatMessage {
	return prependAgentSystemPromptWithSnapshot(messages, modeDef, PromptSnapshot{})
}

func prependAgentSystemPromptWithSnapshot(messages []domain.ChatMessage, modeDef domain.AgentModeDefinition, snapshot PromptSnapshot) []domain.ChatMessage {
	return prependAgentSystemPromptWithSnapshotAndShell(messages, modeDef, snapshot, "")
}

func prependAgentSystemPromptWithSnapshotAndShell(messages []domain.ChatMessage, modeDef domain.AgentModeDefinition, snapshot PromptSnapshot, shellInstruction string) []domain.ChatMessage {
	prompt := strings.TrimSpace(buildAgentSystemPromptWithSnapshotAndShell(modeDef, snapshot, shellInstruction))
	if prompt == "" {
		return append([]domain.ChatMessage(nil), messages...)
	}
	out := make([]domain.ChatMessage, 0, len(messages)+1)
	out = append(out, domain.ChatMessage{Role: domain.EventRoleSystem, Text: prompt})
	out = append(out, messages...)
	return out
}
