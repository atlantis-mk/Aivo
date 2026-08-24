package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"aivo/core/domain"
)

type AgentCatalog struct {
	modes map[string]domain.AgentModeDefinition
	order []string
}

func NewAgentCatalog() *AgentCatalog {
	items := []domain.AgentModeDefinition{
		{
			ID:                   domain.AgentModeAssistant,
			DisplayName:          "Assistant",
			Description:          "Default general Agent with full workspace tools gated by runtime permissions.",
			PromptID:             "agent.assistant",
			Prompt:               builtinPromptBody("agent.assistant"),
			Toolsets:             []string{"*"},
			FileWriteAccess:      true,
			CommandAccess:        true,
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeSummary,
			DisplayName:          "Summary",
			Description:          "Hidden worker mode for conversation summaries and compaction.",
			PromptID:             "agent.summary",
			Prompt:               builtinPromptBody("agent.summary"),
			Toolsets:             []string{"safe"},
			Hidden:               true,
			BackgroundTaskAccess: true,
		},
		{
			ID:                   domain.AgentModeTitle,
			DisplayName:          "Title",
			Description:          "Hidden worker mode for session title generation.",
			PromptID:             "agent.title",
			Prompt:               builtinPromptBody("agent.title"),
			Toolsets:             []string{"safe"},
			Hidden:               true,
			BackgroundTaskAccess: true,
		},
		{
			ID:                   domain.AgentModeSchedulerWorker,
			DisplayName:          "Scheduler Worker",
			Description:          "Hidden worker mode for scheduled jobs with bounded permissions.",
			PromptID:             "agent.scheduler_worker",
			Prompt:               builtinPromptBody("agent.scheduler_worker"),
			Toolsets:             []string{"safe", "personal", "web"},
			NetworkAccess:        true,
			Hidden:               true,
			BackgroundTaskAccess: true,
		},
	}
	c := &AgentCatalog{modes: map[string]domain.AgentModeDefinition{}, order: make([]string, 0, len(items))}
	for _, item := range items {
		if item.Mode == "" {
			item.Mode = "all"
		}
		item.Source = "builtin"
		item.BuiltIn = true
		c.modes[item.ID] = item
		c.order = append(c.order, item.ID)
	}
	return c
}

func NewAgentCatalogWithRuntime(runtime domain.RuntimeConfig) *AgentCatalog {
	catalog := NewAgentCatalog()
	catalog.ApplyRuntime(runtime)
	return catalog
}

func NewAgentCatalogWithDefinitions(definitions []domain.AgentModeDefinition) *AgentCatalog {
	catalog := NewAgentCatalog()
	for _, definition := range definitions {
		id, err := domain.NormalizeAgentMode(definition.ID)
		if err != nil {
			continue
		}
		if isRetiredBuiltInAgentMode(id) {
			continue
		}
		builtInDefinition, builtIn := catalog.modes[id]
		definition.ID = id
		if builtIn {
			definition.Toolsets = append([]string{}, builtInDefinition.Toolsets...)
		} else {
			definition.Toolsets = []string{"safe"}
		}
		definition.BuiltIn = builtIn
		definition.Overridden = builtIn
		definition.Source = "user"
		if definition.Mode == "" {
			definition.Mode = "all"
		}
		definition.FileWriteAccess = definition.PermissionScope != "read_only" && toolsetMayProvide(definition.Toolsets, "coding")
		definition.CommandAccess = definition.PermissionScope != "read_only" && toolsetMayProvide(definition.Toolsets, "coding")
		definition.NetworkAccess = toolsetMayProvide(definition.Toolsets, "web")
		definition.Revision = agentModeRevision(definition)
		if !builtIn {
			catalog.order = append(catalog.order, id)
		}
		catalog.modes[id] = definition
	}
	return catalog
}

func (catalog *AgentCatalog) ApplyPromptSnapshot(snapshot PromptSnapshot) {
	if catalog == nil {
		return
	}
	for id, definition := range catalog.modes {
		promptID := strings.TrimSpace(definition.PromptID)
		if promptID == "" {
			promptID = "agent." + id
		}
		if body := snapshot.Body(promptID); body != "" {
			definition.PromptID = promptID
			definition.Prompt = body
			if !definition.BuiltIn {
				definition.Revision = agentModeRevision(definition)
			}
			catalog.modes[id] = definition
		}
	}
}

func (catalog *AgentCatalog) ApplyRuntime(runtime domain.RuntimeConfig) {
	if catalog == nil {
		return
	}
	names := make([]string, 0, len(runtime.Agents))
	for name := range runtime.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		configured := runtime.Agents[name]
		id, err := domain.NormalizeAgentMode(name)
		if err != nil {
			continue
		}
		definition, exists := catalog.modes[id]
		if configured.Disabled {
			if exists {
				delete(catalog.modes, id)
				catalog.order = removeAgentCatalogID(catalog.order, id)
			}
			continue
		}
		if !exists {
			definition = domain.AgentModeDefinition{
				ID: id, DisplayName: name, Description: configured.Description, Toolsets: []string{"safe"},
				PermissionScope: "read_only", Mode: "all",
			}
			catalog.order = append(catalog.order, id)
		}
		if strings.TrimSpace(configured.DisplayName) != "" {
			definition.DisplayName = strings.TrimSpace(configured.DisplayName)
		}
		if strings.TrimSpace(configured.Description) != "" {
			definition.Description = strings.TrimSpace(configured.Description)
		}
		if strings.TrimSpace(configured.Prompt) != "" {
			definition.Prompt = strings.TrimSpace(configured.Prompt)
		}
		if configured.Model != nil {
			definition.Model = configured.Model
		}
		if configured.Temperature != nil {
			definition.Temperature = configured.Temperature
		}
		if configured.TopP != nil {
			definition.TopP = configured.TopP
		}
		if configured.MaxSteps > 0 {
			definition.MaxSteps = configured.MaxSteps
		}
		if len(configured.Toolsets) > 0 {
			definition.Toolsets = append([]string{}, configured.Toolsets...)
		}
		if strings.TrimSpace(configured.PermissionScope) != "" {
			definition.PermissionScope = strings.TrimSpace(configured.PermissionScope)
		}
		if strings.TrimSpace(configured.Mode) != "" {
			definition.Mode = strings.TrimSpace(configured.Mode)
		}
		if configured.Subagents != nil {
			definition.Subagents = make([]string, 0, len(configured.Subagents))
			for _, candidate := range configured.Subagents {
				normalized, err := domain.NormalizeAgentMode(candidate)
				if err != nil {
					definition.Subagents = append(definition.Subagents, candidate)
					continue
				}
				definition.Subagents = append(definition.Subagents, normalized)
			}
		}
		definition.Variant = strings.TrimSpace(configured.Variant)
		definition.Options = cloneAnyMap(configured.Options)
		definition.Hidden = configured.Hidden
		definition.Source = "project"
		definition.BuiltIn = exists && definition.BuiltIn
		definition.Overridden = exists
		definition.FileWriteAccess = definition.PermissionScope != "read_only" && toolsetMayProvide(definition.Toolsets, "coding")
		definition.CommandAccess = definition.PermissionScope != "read_only" && toolsetMayProvide(definition.Toolsets, "coding")
		definition.NetworkAccess = toolsetMayProvide(definition.Toolsets, "web")
		definition.Revision = agentModeRevision(configured)
		catalog.modes[id] = definition
	}
}

func agentModeRevision(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func removeAgentCatalogID(ids []string, wanted string) []string {
	out := ids[:0]
	for _, id := range ids {
		if id != wanted {
			out = append(out, id)
		}
	}
	return out
}

func toolsetMayProvide(toolsets []string, wanted string) bool {
	for _, toolset := range toolsets {
		if strings.TrimSpace(toolset) == "*" || strings.TrimSpace(toolset) == wanted {
			return true
		}
	}
	return false
}

func (c *AgentCatalog) List(includeHidden bool) []domain.AgentModeDefinition {
	if c == nil {
		c = NewAgentCatalog()
	}
	out := make([]domain.AgentModeDefinition, 0, len(c.order))
	for _, id := range c.order {
		mode := c.modes[id]
		if mode.Hidden && !includeHidden {
			continue
		}
		out = append(out, mode)
	}
	return out
}

func (c *AgentCatalog) Get(mode string) (domain.AgentModeDefinition, error) {
	if c == nil {
		c = NewAgentCatalog()
	}
	id, err := domain.NormalizeAgentMode(mode)
	if err != nil {
		return domain.AgentModeDefinition{}, err
	}
	def, ok := c.modes[id]
	if !ok {
		return domain.AgentModeDefinition{}, fmt.Errorf("agent mode %q is not configured", id)
	}
	return def, nil
}

func toolSpecInToolsets(spec domain.ToolSpec, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	for _, allowedToolset := range allowed {
		if strings.TrimSpace(allowedToolset) == "*" {
			return true
		}
	}
	for _, toolset := range spec.Toolsets {
		for _, allowedToolset := range allowed {
			if strings.TrimSpace(toolset) == strings.TrimSpace(allowedToolset) {
				return true
			}
		}
	}
	return false
}
