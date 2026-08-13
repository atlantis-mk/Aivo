package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"aivo/core/domain"
)

const expandedCommandMaxChars = 64 * 1024

func (s *Service) ListCommandCatalog(ctx context.Context, input domain.CommandCatalogInput) ([]domain.CommandCatalogEntry, error) {
	effective := loadEffectiveRuntimeConfig(input.ProjectPath)
	entries := []domain.CommandCatalogEntry{
		{ID: "builtin:init", Name: "init", Description: "Analyze the project and create or improve its AGENTS.md instructions.", Source: "builtin", SourceID: "init", Agent: domain.AgentModeBuild},
		{ID: "builtin:review", Name: "review", Description: "Review current changes, a commit, or a branch in an isolated child session.", Source: "builtin", SourceID: "review", Agent: domain.AgentModeReview, Subtask: true},
	}
	for name, command := range effective.Config.Commands {
		entries = append(entries, domain.CommandCatalogEntry{
			ID: "config:" + name, Name: name, Description: command.Description, Source: "config", SourceID: name,
			Arguments: command.Arguments, Agent: command.Agent, Model: command.Model, Toolsets: command.Toolsets,
			Subtask: command.Subtask,
		})
	}
	if skills, err := s.ListSkills(ctx, domain.SkillListInput{WorkspaceRoot: input.ProjectPath}); err == nil {
		for _, skill := range skills.Entries {
			if skill.Enabled {
				entries = append(entries, domain.CommandCatalogEntry{ID: "skill:" + skill.ID, Name: skill.Name, Description: skill.Description, Source: "skill", SourceID: skill.ID})
			}
		}
	}
	if servers, err := s.ListMCPServers(ctx, domain.MCPServerListInput{}); err == nil {
		for _, server := range servers {
			for _, prompt := range server.Prompts {
				args := make([]domain.CommandArgument, 0, len(prompt.Arguments))
				for _, argument := range prompt.Arguments {
					args = append(args, domain.CommandArgument{Name: argument.Name, Description: argument.Description, Required: argument.Required})
				}
				entries = append(entries, domain.CommandCatalogEntry{
					ID: "mcp:" + prompt.ServerID + ":" + prompt.Name, Name: prompt.Name, Description: prompt.Description,
					Source: "mcp", SourceID: prompt.ServerID, Arguments: args,
				})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s *Service) InvokeCommand(ctx context.Context, input domain.InvokeCommandInput) (domain.InvokeCommandResult, error) {
	commandID := strings.TrimSpace(input.CommandID)
	if commandID == "" {
		return domain.InvokeCommandResult{}, errors.New("commandId is required")
	}
	projectPath := strings.TrimSpace(input.ProjectPath)
	if projectPath == "" && strings.TrimSpace(input.SessionID) != "" {
		if session, err := s.store.GetRuntimeSession(ctx, input.SessionID); err == nil {
			projectPath = session.ProjectPath
		}
	}
	switch {
	case strings.HasPrefix(commandID, "builtin:"):
		name := strings.TrimPrefix(commandID, "builtin:")
		var result domain.InvokeCommandResult
		switch name {
		case "init":
			result = domain.InvokeCommandResult{CommandID: commandID, Source: "builtin", SourceID: name, Agent: domain.AgentModeBuild, Prompt: "Inspect this repository, then create or improve AGENTS.md with concise project-specific structure, build/test commands, coding conventions, safety constraints, and verification guidance. Preserve useful existing instructions and do not invent commands that are not present in the project."}
		case "review":
			target := strings.TrimSpace(input.Arguments["ARGUMENTS"])
			if target == "" {
				target = "the current uncommitted changes"
			}
			result = domain.InvokeCommandResult{CommandID: commandID, Source: "builtin", SourceID: name, Agent: domain.AgentModeReview, Subtask: true, Prompt: "Review " + target + ". Prioritize correctness bugs, regressions, security or data-loss risks, and missing tests. Report findings first with concrete file references; do not modify files."}
		default:
			return domain.InvokeCommandResult{}, fmt.Errorf("built-in command %q not found", name)
		}
		return s.finishCommandInvocation(ctx, input, result)
	case strings.HasPrefix(commandID, "config:"):
		name := strings.TrimPrefix(commandID, "config:")
		definition, ok := loadEffectiveRuntimeConfig(projectPath).Config.Commands[name]
		if !ok {
			return domain.InvokeCommandResult{}, fmt.Errorf("configured command %q not found", name)
		}
		prompt, err := expandCommandTemplate(definition, input.Arguments)
		if err != nil {
			return domain.InvokeCommandResult{}, err
		}
		return s.finishCommandInvocation(ctx, input, domain.InvokeCommandResult{
			CommandID: commandID, Source: "config", SourceID: name, Prompt: prompt, Agent: definition.Agent,
			Model: definition.Model, Toolsets: definition.Toolsets, Subtask: definition.Subtask, Provenance: map[string]string{"projectPath": projectPath},
		})
	case strings.HasPrefix(commandID, "skill:"):
		if strings.TrimSpace(input.SessionID) == "" {
			return domain.InvokeCommandResult{}, errors.New("sessionId is required to invoke a skill command")
		}
		skillID := strings.TrimPrefix(commandID, "skill:")
		loaded, err := s.loadOrImportSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: input.SessionID, SkillID: skillID, Reason: "invoked from command catalog"})
		if err != nil {
			return domain.InvokeCommandResult{}, err
		}
		return s.finishCommandInvocation(ctx, input, domain.InvokeCommandResult{CommandID: commandID, Source: "skill", SourceID: loaded.Skill.ID, Prompt: loaded.ModelOutput, Provenance: map[string]string{"skillName": loaded.Skill.Name}})
	case strings.HasPrefix(commandID, "mcp:"):
		serverID, name, ok := splitMCPCommandID(commandID)
		if !ok {
			return domain.InvokeCommandResult{}, errors.New("invalid MCP command id")
		}
		result, err := s.GetMCPPrompt(ctx, domain.MCPPromptGetInput{ServerID: serverID, Name: name, Arguments: input.Arguments})
		if err != nil {
			return domain.InvokeCommandResult{}, err
		}
		return s.finishCommandInvocation(ctx, input, domain.InvokeCommandResult{CommandID: commandID, Source: "mcp", SourceID: serverID, Prompt: bounded(result.Content, expandedCommandMaxChars), Provenance: map[string]string{"promptName": name}})
	default:
		return domain.InvokeCommandResult{}, fmt.Errorf("unsupported command source in %q", commandID)
	}
}

func (s *Service) finishCommandInvocation(ctx context.Context, input domain.InvokeCommandInput, result domain.InvokeCommandResult) (domain.InvokeCommandResult, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return result, nil
	}
	payload := map[string]any{
		"commandId": result.CommandID, "source": result.Source, "sourceId": result.SourceID,
		"agent": result.Agent, "model": result.Model, "toolsets": result.Toolsets, "arguments": input.Arguments,
		"provenance": result.Provenance,
	}
	if _, err := s.AppendEvent(ctx, domain.AppendEventRequest{
		SessionID: input.SessionID, Type: domain.EventTypeSystemNote, Role: domain.EventRoleSystem,
		Visibility: domain.EventVisibilityInternal, Content: "Invoked command " + result.CommandID, Payload: payload,
	}); err != nil {
		return domain.InvokeCommandResult{}, fmt.Errorf("record command invocation: %w", err)
	}
	if result.Subtask {
		return s.executeCommandSubtask(ctx, input, result)
	}
	return result, nil
}

func (s *Service) executeCommandSubtask(ctx context.Context, input domain.InvokeCommandInput, result domain.InvokeCommandResult) (domain.InvokeCommandResult, error) {
	parent, err := s.store.GetRuntimeSession(ctx, input.SessionID)
	if err != nil {
		return domain.InvokeCommandResult{}, err
	}
	mode := firstNonEmpty(strings.TrimSpace(result.Agent), parent.AgentMode)
	catalog, err := s.agentCatalogForProject(ctx, parent.ProjectPath)
	if err != nil {
		return domain.InvokeCommandResult{}, err
	}
	definition, err := catalog.Get(mode)
	if err != nil || definition.ID == domain.AgentModeSummary || definition.ID == domain.AgentModeTitle || definition.ID == domain.AgentModeSchedulerWorker {
		return domain.InvokeCommandResult{}, errors.New("command subtask agent is unavailable")
	}
	if definition.Mode == "primary" {
		return domain.InvokeCommandResult{}, errors.New("command subtask agent is primary-only")
	}
	if definition.Revision != "" {
		if err := s.validateAgentToolsets(parent.ProjectPath, definition); err != nil {
			return domain.InvokeCommandResult{}, err
		}
	}
	child, err := s.store.ForkRuntimeSession(ctx, parent, domain.ForkSessionRequest{Title: "Command: " + result.SourceID, Goal: result.Prompt})
	if err != nil {
		return domain.InvokeCommandResult{}, err
	}
	child, err = s.store.SetRuntimeSessionAgentMode(ctx, child.ID, definition.ID)
	if err != nil {
		return domain.InvokeCommandResult{}, err
	}
	run, err := s.store.SaveAgentRun(ctx, domain.AgentRun{ParentSessionID: parent.ID, SessionID: child.ID, Mode: definition.ID, Status: domain.AgentRunStatusRunning, Prompt: result.Prompt, Metadata: map[string]string{"commandId": result.CommandID}})
	if err != nil {
		return domain.InvokeCommandResult{}, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.registerActiveAgentRun(run.ID, cancel)
	prepared, runErr := s.SubmitSessionMessage(runCtx, domain.SubmitSessionMessageRequest{SessionID: child.ID, Text: result.Prompt, AgentMode: definition.ID, Model: result.Model, Toolsets: result.Toolsets})
	runCtxErr := runCtx.Err()
	s.unregisterActiveAgentRun(run.ID)
	cancel()
	run.Status = domain.AgentRunStatusCompleted
	if runCtxErr != nil {
		run.Status, run.Error = domain.AgentRunStatusCancelled, runCtxErr.Error()
	} else if runErr != nil {
		run.Status, run.Error = domain.AgentRunStatusFailed, runErr.Error()
	} else if prepared.AssistantEvent != nil {
		run.Result = prepared.AssistantEvent.Content
	}
	run, _ = s.store.SaveAgentRun(ctx, run)
	result.ChildSessionID = child.ID
	result.AgentRunID = run.ID
	result.Response = run.Result
	_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: parent.ID, Type: domain.EventTypeUserMessage, Role: domain.EventRoleUser, Visibility: domain.EventVisibilityNormal, Content: "Invoked /" + result.SourceID, Payload: map[string]any{"commandId": result.CommandID, "arguments": input.Arguments}})
	if strings.TrimSpace(run.Result) != "" {
		_, _ = s.AppendEvent(ctx, domain.AppendEventRequest{SessionID: parent.ID, Type: domain.EventTypeAssistantMessage, Role: domain.EventRoleAssistant, Visibility: domain.EventVisibilityNormal, Content: run.Result, Payload: map[string]any{"commandId": result.CommandID, "agentRunId": run.ID, "childSessionId": child.ID}})
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

func expandCommandTemplate(definition domain.CommandTemplateDefinition, values map[string]string) (string, error) {
	values = cloneStringMap(values)
	ordered := make([]string, 0, len(definition.Arguments))
	allowed := map[string]bool{}
	allowed["ARGUMENTS"] = true
	for index, argument := range definition.Arguments {
		name := strings.TrimSpace(argument.Name)
		if name == "" {
			return "", errors.New("command argument name is required")
		}
		allowed[name] = true
		value := values[name]
		if value == "" {
			value = argument.Default
		}
		if argument.Required && strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("command argument %q is required", name)
		}
		values[name] = value
		ordered = append(ordered, value)
		values[strconv.Itoa(index+1)] = value
	}
	for name := range values {
		if _, err := strconv.Atoi(name); err == nil {
			continue
		}
		if !allowed[name] {
			return "", fmt.Errorf("unknown command argument %q", name)
		}
	}
	prompt := definition.Template
	for name, value := range values {
		prompt = strings.ReplaceAll(prompt, "{{"+name+"}}", value)
		prompt = strings.ReplaceAll(prompt, "$"+name, value)
	}
	rawArguments := values["ARGUMENTS"]
	if rawArguments == "" {
		rawArguments = strings.Join(ordered, " ")
	}
	prompt = strings.ReplaceAll(prompt, "$ARGUMENTS", rawArguments)
	if strings.Contains(prompt, "{{") {
		return "", errors.New("command template contains an unresolved placeholder")
	}
	if len(prompt) > expandedCommandMaxChars {
		return "", fmt.Errorf("expanded command exceeds %d characters", expandedCommandMaxChars)
	}
	return strings.TrimSpace(prompt), nil
}

func splitMCPCommandID(commandID string) (string, string, bool) {
	rest := strings.TrimPrefix(commandID, "mcp:")
	index := strings.LastIndex(rest, ":")
	if index <= 0 || index == len(rest)-1 {
		return "", "", false
	}
	return rest[:index], rest[index+1:], true
}

func cloneStringMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
