package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"aivo/core/domain"
)

type ToolResolveRequest struct {
	Intent     string
	Required   bool
	MaxTools   int
	Source     string
	Category   string
	RiskLevel  string
	SessionID  string
	TurnID     string
	AgentMode  string
	Candidates []domain.ToolCatalogEntry
}

type ToolResolveDecision struct {
	Names  []string
	Reason string
}

type ToolResolveFunc func(context.Context, ToolResolveRequest) (ToolResolveDecision, error)

type ToolReplaceFunc func(context.Context, string, []string) error

type ToolResolveTool struct {
	registry *Registry
	resolve  ToolResolveFunc
	replace  ToolReplaceFunc
}
type ToolSearchTool struct{ registry *Registry }
type ToolListTool struct{ registry *Registry }
type ToolDetailTool struct {
	registry *Registry
	name     string
}
type ToolCallTool struct {
	registry *Registry
	runtime  func() *ToolRuntime
}

func NewToolResolveTool(registry *Registry, resolve ToolResolveFunc, replace ToolReplaceFunc) *ToolResolveTool {
	return &ToolResolveTool{registry: registry, resolve: resolve, replace: replace}
}
func NewToolSearchTool(registry *Registry) *ToolSearchTool {
	return &ToolSearchTool{registry: registry}
}
func NewToolListTool(registry *Registry) *ToolListTool {
	return &ToolListTool{registry: registry}
}
func NewToolDetailTool(registry *Registry) *ToolDetailTool {
	return &ToolDetailTool{registry: registry, name: ToolDetailName}
}
func NewToolCallTool(registry *Registry, runtime func() *ToolRuntime) *ToolCallTool {
	return &ToolCallTool{registry: registry, runtime: runtime}
}

func (t *ToolResolveTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Intent    string `json:"intent"`
		Required  *bool  `json:"required"`
		MaxTools  int    `json:"maxTools"`
		Source    string `json:"source"`
		Category  string `json:"category"`
		RiskLevel string `json:"riskLevel"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(ToolResolveName, errors.New("invalid tool_resolve arguments"))
	}
	input.Intent = strings.TrimSpace(input.Intent)
	if input.Intent == "" {
		return toolError(ToolResolveName, errors.New("intent is required"))
	}
	required := true
	if input.Required != nil {
		required = *input.Required
	}
	if input.MaxTools <= 0 {
		input.MaxTools = 8
	}
	if input.MaxTools > 20 {
		input.MaxTools = 20
	}
	candidates := toolResolveCandidates(t.registry, execCtx, input.Source, input.Category, input.RiskLevel)
	if len(candidates) == 0 {
		if !required {
			return t.replaceWithNoAutomaticTools(ctx, execCtx, "no allowed deferred tools match the requested filters")
		}
		return toolResolveNoAvailable(execCtx.ToolCallID, input.Intent, required, "no allowed deferred tools match the requested filters")
	}
	resolver := t.resolve
	if resolver == nil {
		resolver = localToolResolve
	}
	decision, err := resolver(ctx, ToolResolveRequest{
		Intent: input.Intent, Required: required, MaxTools: input.MaxTools, Source: input.Source, Category: input.Category,
		RiskLevel: input.RiskLevel, SessionID: execCtx.SessionID, TurnID: execCtx.TurnID, AgentMode: execCtx.AgentMode,
		Candidates: candidates,
	})
	if err != nil {
		return toolFailure(execCtx.ToolCallID, ToolResolveName, "tool_resolve_failed", err.Error())
	}
	selected := validateToolResolveSelection(candidates, decision.Names, hostExpandedToolLimit)
	if len(selected) == 0 {
		if !required {
			return t.replaceWithNoAutomaticTools(ctx, execCtx, decision.Reason)
		}
		return toolResolveNoAvailable(execCtx.ToolCallID, input.Intent, required, firstNonEmpty(decision.Reason, "no candidate tool satisfied the requested capability"))
	}
	items := make([]map[string]any, 0, len(selected))
	names := make([]string, 0, len(selected))
	for _, entry := range selected {
		names = append(names, entry.Name)
		items = append(items, toolCatalogListItem(entry))
	}
	if t.replace != nil {
		if err := t.replace(ctx, execCtx.SessionID, names); err != nil {
			return toolFailure(execCtx.ToolCallID, ToolResolveName, "tool_activation_failed", err.Error())
		}
	}
	structured := map[string]any{
		"status":          "replaced",
		"tools":           items,
		"count":           len(items),
		"reason":          strings.TrimSpace(decision.Reason),
		"appliesNextStep": true,
	}
	raw, _ := json.MarshalIndent(structured, "", "  ")
	return domain.ToolResult{Name: ToolResolveName, CallID: execCtx.ToolCallID, OK: true, Content: string(raw), Structured: structured}
}

func (t *ToolResolveTool) replaceWithNoAutomaticTools(ctx context.Context, execCtx domain.ToolExecutionContext, reason string) domain.ToolResult {
	if t.replace != nil {
		if err := t.replace(ctx, execCtx.SessionID, nil); err != nil {
			return toolFailure(execCtx.ToolCallID, ToolResolveName, "tool_activation_failed", err.Error())
		}
	}
	structured := map[string]any{"status": "replaced", "tools": []map[string]any{}, "count": 0, "reason": strings.TrimSpace(reason), "appliesNextStep": true}
	raw, _ := json.MarshalIndent(structured, "", "  ")
	return domain.ToolResult{Name: ToolResolveName, CallID: execCtx.ToolCallID, OK: true, Content: string(raw), Structured: structured}
}

func (t *ToolSearchTool) Execute(_ context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(ToolSearchName, errors.New("invalid tool_search arguments"))
	}
	input.Query = strings.TrimSpace(input.Query)
	if input.Query == "" {
		return toolError(ToolSearchName, errors.New("query is required"))
	}
	if input.Limit <= 0 || input.Limit > 20 {
		input.Limit = 5
	}
	entries := deferrableCatalogEntries(t.registry)
	matches := searchToolCatalog(entries, input.Query, input.Limit)
	sourceCounts := countDeferredCatalogSources(entries)
	items := make([]map[string]any, 0, len(matches))
	for _, entry := range matches {
		items = append(items, toolCatalogListItem(entry))
	}
	structured := map[string]any{"matches": items, "count": len(items), "availableDeferredCount": len(entries), "sourceCounts": sourceCounts}
	raw, _ := json.MarshalIndent(structured, "", "  ")
	return domain.ToolResult{Name: ToolSearchName, OK: true, Content: string(raw), Structured: structured}
}

func (t *ToolListTool) Execute(_ context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	if t.registry == nil {
		return toolFailure("", ToolListName, "registry_unavailable", "tool registry is unavailable")
	}
	var input struct {
		Source          string `json:"source"`
		Category        string `json:"category"`
		Query           string `json:"query"`
		IncludeCore     *bool  `json:"includeCore"`
		IncludeLongTail *bool  `json:"includeLongTail"`
		Limit           int    `json:"limit"`
		Offset          int    `json:"offset"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return toolError(ToolListName, errors.New("invalid tool_list arguments"))
		}
	}
	includeCore := true
	if input.IncludeCore != nil {
		includeCore = *input.IncludeCore
	}
	includeLongTail := true
	if input.IncludeLongTail != nil {
		includeLongTail = *input.IncludeLongTail
	}
	if input.Limit <= 0 {
		input.Limit = 100
	}
	if input.Limit > 200 {
		input.Limit = 200
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	source := strings.ToLower(strings.TrimSpace(input.Source))
	category := strings.ToLower(strings.TrimSpace(input.Category))
	query := strings.ToLower(strings.TrimSpace(input.Query))
	entries := t.registry.CatalogEntries()
	filtered := make([]domain.ToolCatalogEntry, 0, len(entries))
	sourceCounts := map[string]int{}
	for _, entry := range entries {
		deferred := isToolCatalogEntryDeferrable(entry)
		if deferred && !includeLongTail {
			continue
		}
		if !deferred && !includeCore {
			continue
		}
		if source != "" && strings.ToLower(strings.TrimSpace(entry.Source)) != source {
			continue
		}
		if category != "" && strings.ToLower(strings.TrimSpace(entry.Category)) != category {
			continue
		}
		if query != "" && !strings.Contains(toolCatalogSearchText(entry), query) {
			continue
		}
		filtered = append(filtered, entry)
		sourceKey := strings.TrimSpace(entry.Source)
		if sourceKey == "" {
			sourceKey = "unknown"
		}
		sourceCounts[sourceKey]++
	}
	total := len(filtered)
	start := input.Offset
	if start > total {
		start = total
	}
	end := start + input.Limit
	if end > total {
		end = total
	}
	items := make([]map[string]any, 0, end-start)
	for _, entry := range filtered[start:end] {
		items = append(items, toolCatalogListItem(entry))
	}
	structured := map[string]any{
		"tools": items, "count": len(items), "total": total, "offset": start, "limit": input.Limit,
		"truncated": end < total, "sourceCounts": sourceCounts,
	}
	if end < total {
		structured["nextOffset"] = end
	}
	raw, _ := json.MarshalIndent(structured, "", "  ")
	return domain.ToolResult{Name: ToolListName, OK: true, Content: string(raw), Structured: structured}
}

func (t *ToolDetailTool) Execute(_ context.Context, args json.RawMessage, _ domain.ToolExecutionContext) domain.ToolResult {
	toolName := firstNonEmpty(t.name, ToolDetailName)
	if t.registry == nil {
		return toolFailure("", toolName, "registry_unavailable", "tool registry is unavailable")
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(toolName, errors.New("invalid "+toolName+" arguments"))
	}
	name := strings.TrimSpace(input.Name)
	for _, entry := range t.registry.CatalogEntries() {
		if entry.Name == name {
			raw, _ := json.MarshalIndent(entry, "", "  ")
			return domain.ToolResult{Name: toolName, OK: true, Content: string(raw), Structured: map[string]any{"tool": entry}}
		}
	}
	return toolFailure("", toolName, "tool_not_found", "tool is not available: "+name)
}

func (t *ToolCallTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if execCtx.BridgeCallDepth > 0 {
		return toolFailure(execCtx.ToolCallID, ToolCallName, "recursive_tool_call", "tool_call cannot invoke another deferred bridge call")
	}
	var input struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError(ToolCallName, errors.New("invalid tool_call arguments"))
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return toolError(ToolCallName, errors.New("name is required"))
	}
	if isBridgeToolName(name) {
		return toolFailure(execCtx.ToolCallID, ToolCallName, "invalid_tool_call", "tool_call cannot invoke bridge tools")
	}
	if len(input.Arguments) == 0 {
		input.Arguments = json.RawMessage(`{}`)
	}
	runtime := t.runtime()
	if runtime == nil {
		return toolFailure(execCtx.ToolCallID, name, "runtime_unavailable", "tool runtime is unavailable")
	}
	execCtx.BridgeCallDepth++
	call := domain.ChatToolCall{ID: execCtx.ToolCallID, Name: name, Arguments: input.Arguments}
	return runtime.ExecuteWithContext(ctx, call, execCtx)
}
