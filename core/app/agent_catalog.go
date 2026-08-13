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
			ID:                   domain.AgentModeCode,
			DisplayName:          "Code",
			Description:          "Default primary coding agent with full workspace tools gated by runtime permissions.",
			Prompt:               "You are running as the primary coding agent. Use read for exact files, bash for repository search, Git, tests, builds, formatting, diagnostics, and other foreground CLI work, edit for exact atomic replacements, and write for complete file creation or overwrite. Optional capabilities appear only when the Host activates an extension before a model call. Be concise, direct, and to the point. Before a meaningful batch of tool calls, give one short progress update under 24 Chinese characters or 12 English words. Do not narrate routine reads, searches, or trivial edits. Answers should normally be fewer than 4 lines unless the work needs more detail. Inspect relevant files before answering, keep changes scoped, and report verification clearly.",
			Toolsets:             []string{"*"},
			FileWriteAccess:      true,
			CommandAccess:        true,
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeAssistant,
			DisplayName:          "Assistant",
			Description:          "General conversational mode with coding tools available when explicitly useful.",
			Prompt:               "You are running in assistant mode. Use read, bash, edit, and write according to runtime permissions; ordinary Git, search, test, build, formatting, and diagnostic work uses bash. Optional capabilities appear only when the Host activates an extension. Do not store secrets, credentials, transient chat, or raw private tool content.",
			Toolsets:             []string{"safe", "coding", "git", "web"},
			FileWriteAccess:      true,
			CommandAccess:        false,
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeBuild,
			DisplayName:          "Build",
			Description:          "Implements code changes, edits files, and runs approved project commands.",
			Prompt:               "You are running in build mode. Inspect with read and bash, make focused exact changes with edit or complete writes with write, and run the smallest useful verification through bash. Optional capabilities appear only when the Host activates an extension. Respect runtime permissions and avoid unrelated refactors.",
			Toolsets:             []string{"safe", "coding", "git", "web"},
			FileWriteAccess:      true,
			CommandAccess:        true,
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeExplore,
			DisplayName:          "Explore",
			Description:          "Reads code and answers questions without mutating files or running commands.",
			Prompt:               "You are running in explore mode. Investigate the repository, explain how things work, compare options, and surface risks. Do not mutate files, run shell commands, run tests, or create scheduled jobs. Use update_plan only when the investigation has multiple concrete steps.",
			Toolsets:             []string{"safe", "git", "web"},
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModePlan,
			DisplayName:          "Plan",
			Description:          "Plans implementation work without mutating files or running shell commands.",
			Prompt:               "You are running in plan mode. Inspect the repository, then produce concrete implementation plans. Use update_plan for multi-step planning work so the user can see progress; keep at most one step in_progress and do not batch completion updates. Do not mutate files, run shell commands, or create scheduler jobs.",
			Toolsets:             []string{"safe", "git", "web"},
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModePlanner,
			DisplayName:          "Planner",
			Description:          "Compatibility alias for Plan.",
			Prompt:               "You are running in planner mode. Inspect the repository, then produce concrete implementation plans. Use update_plan for multi-step planning work so the user can see progress; keep at most one step in_progress and do not batch completion updates. Do not mutate files, run shell commands, or create scheduler jobs.",
			Toolsets:             []string{"safe", "git", "web"},
			NetworkAccess:        true,
			Hidden:               true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeReview,
			DisplayName:          "Review",
			Description:          "Reviews code for bugs, regressions, risks, and missing tests without changing files.",
			Prompt:               "You are running in review mode. Take a code-review stance: prioritize concrete bugs, behavioral regressions, security or data-loss risks, and missing tests. Findings should come first, ordered by severity, with file and line references when available. Do not mutate files, run shell commands, run tests, or create scheduled jobs.",
			Toolsets:             []string{"safe", "git", "web"},
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeDebug,
			DisplayName:          "Debug",
			Description:          "Investigates failures and may run approved diagnostics, but cannot edit files.",
			Prompt:               "You are running in debug mode. Diagnose failures with read and approved foreground bash commands. Do not mutate files or create scheduled jobs. When you identify a likely fix, explain it clearly and switch to build mode only if the user wants implementation.",
			Toolsets:             []string{"safe", "coding", "git", "web"},
			CommandAccess:        true,
			NetworkAccess:        true,
			BackgroundTaskAccess: false,
		},
		{
			ID:                   domain.AgentModeSummary,
			DisplayName:          "Summary",
			Description:          "Hidden worker mode for conversation summaries and compaction.",
			Prompt:               "You are running in summary mode. Produce concise durable summaries, open tasks, decisions, facts, and changed files from the provided conversation context. Do not mutate files, run shell commands, or create scheduled jobs.",
			Toolsets:             []string{"safe"},
			Hidden:               true,
			BackgroundTaskAccess: true,
		},
		{
			ID:                   domain.AgentModeTitle,
			DisplayName:          "Title",
			Description:          "Hidden worker mode for session title generation.",
			Prompt:               "You are running in title mode. Generate a short single-line title for the session from the provided context. Do not use tools unless safe reads are explicitly needed. Do not mutate files, run shell commands, create scheduled jobs, or write memories.",
			Toolsets:             []string{"safe"},
			Hidden:               true,
			BackgroundTaskAccess: true,
		},
		{
			ID:                   domain.AgentModeSchedulerWorker,
			DisplayName:          "Scheduler Worker",
			Description:          "Hidden worker mode for scheduled jobs with bounded permissions.",
			Prompt:               "You are running in scheduler worker mode. Execute only the scheduled job prompt with the job's declared toolsets and permission scope. Keep context bounded and report concise results.",
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
