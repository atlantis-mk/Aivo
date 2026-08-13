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

const agentToolProtocolPrompt = `The Host gives this conversation four core execution primitives, any manually enabled tools, and one stable bounded automatic tool set. Treat Skill summaries as availability metadata and injected instructions/context as task context. Use only tools actually present in the request. When the visible tools cannot perform a concrete action required by the current task, call tool_resolve once with a concise description of the missing capability; it replaces the complete automatic tool set for the next model step and does not change manual tools. Do not call it to list hidden tools, speculate about optional capabilities, or accumulate more tools.`

type promptInjection struct {
	Scope promptInjectionScope
	Name  string
	Text  string
}

func agentPromptInjections(modeDef domain.AgentModeDefinition) []promptInjection {
	injections := []promptInjection{
		{
			Scope: promptInjectionScopeDefault,
			Name:  "agent_mode",
			Text:  "Agent mode: " + modeDef.DisplayName + "\n\n" + modeDef.Prompt,
		},
		{
			Scope: promptInjectionScopeGlobal,
			Name:  "tool_protocol",
			Text:  agentToolProtocolPrompt,
		},
	}
	if len(modeDef.Subagents) > 0 {
		quoted := make([]string, 0, len(modeDef.Subagents))
		for _, subagent := range modeDef.Subagents {
			quoted = append(quoted, fmt.Sprintf("`%s`", subagent))
		}
		injections = append(injections, promptInjection{
			Scope: promptInjectionScopeDefault,
			Name:  "associated_subagents",
			Text: "Associated subagents available to this mode: " + strings.Join(quoted, ", ") +
				". When the current task has a bounded independent part that benefits from one of these modes, you may call agent_delegate_task and then use its returned result. Decide based on the task; do not delegate routine work or fan out solely because associations exist. Never name an unlisted mode.",
		})
	}
	return injections
}

func buildAgentSystemPrompt(modeDef domain.AgentModeDefinition) string {
	injections := agentPromptInjections(modeDef)
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
	prompt := strings.TrimSpace(buildAgentSystemPrompt(modeDef))
	if prompt == "" {
		return append([]domain.ChatMessage(nil), messages...)
	}
	out := make([]domain.ChatMessage, 0, len(messages)+1)
	out = append(out, domain.ChatMessage{Role: domain.EventRoleSystem, Text: prompt})
	out = append(out, messages...)
	return out
}
