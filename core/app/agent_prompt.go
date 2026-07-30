package app

import (
	"strings"

	"aivo/core/domain"
)

type promptInjectionScope string

const (
	promptInjectionScopeDefault promptInjectionScope = "default"
	promptInjectionScopeGlobal  promptInjectionScope = "global"
)

const agentToolProtocolPrompt = `When specialized workflow instructions may help, first call the skill tool with mode=discover and a concise intent. Privately filtered skill names and descriptions will be returned. Review them and call mode=activate with only the exact applicable names before continuing; never treat discovery as activation and never guess names. Use mode=list only when the user asks which skills are available. If current tools cannot perform a required action, call tool_resolve with a concise, specific missing capability. Do not use tool_resolve for convenience, exploration, planning, or guessing tool names. If no allowed tool matches, stop with a local no_available_tool error.`

type promptInjection struct {
	Scope promptInjectionScope
	Name  string
	Text  string
}

func agentPromptInjections(modeDef domain.AgentModeDefinition) []promptInjection {
	return []promptInjection{
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
