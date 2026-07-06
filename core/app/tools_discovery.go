package app

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode"

	"aivo/core/domain"
)

const (
	ToolResolveName = "tool_resolve"
	ToolSearchName  = "tool_search"
	ToolListName    = "tool_list"
	ToolDetailName  = "tool_detail"
	ToolCallName    = "tool_call"
)

type ToolAssemblyResult struct {
	Specs                 []domain.ToolSpec
	Activated             bool
	DeferredCount         int
	ExpectedRegistrations map[string]domain.ToolRegistrationIdentity
}

func AssembleToolSpecs(registry *Registry, specs []domain.ToolSpec) ToolAssemblyResult {
	return AssembleToolSpecsWithActivated(registry, specs, nil)
}

func AssembleToolSpecsWithActivated(registry *Registry, specs []domain.ToolSpec, activated map[string]bool) ToolAssemblyResult {
	identities := map[string]domain.ToolRegistrationIdentity{}
	if registry != nil {
		for _, entry := range registry.CatalogEntries() {
			identities[entry.Name] = domain.ToolRegistrationIdentity{
				Name: entry.Name, RegistrationID: entry.RegistrationID, Source: entry.Source, SourceID: entry.SourceID,
			}
		}
	}
	visible := make([]domain.ToolSpec, 0, len(specs))
	deferred := make([]domain.ToolSpec, 0)
	for _, spec := range specs {
		if isBridgeToolName(spec.Name) {
			if spec.Name == ToolResolveName {
				visible = append(visible, spec)
			}
			continue
		}
		if !isDeferrableToolSpec(spec, identities[spec.Name]) {
			visible = append(visible, spec)
			continue
		}
		if isDeferredToolActivated(spec, activated) {
			visible = append(visible, spec)
			continue
		}
		deferred = append(deferred, spec)
	}
	bridgeActivated := len(deferred) > 0
	if !bridgeActivated {
		return ToolAssemblyResult{Specs: specs, ExpectedRegistrations: identities, DeferredCount: len(deferred)}
	}
	visible = appendBridgeSpecsIfMissing(visible, len(deferred))
	return ToolAssemblyResult{Specs: visible, Activated: bridgeActivated, DeferredCount: len(deferred), ExpectedRegistrations: identities}
}

func appendBridgeSpecsIfMissing(specs []domain.ToolSpec, deferredCount int) []domain.ToolSpec {
	seen := map[string]bool{}
	for _, spec := range specs {
		seen[spec.Name] = true
	}
	if !seen[ToolResolveName] {
		specs = append(specs, toolResolveSpec(deferredCount))
	}
	return specs
}

func isBridgeToolName(name string) bool {
	switch name {
	case ToolResolveName, ToolSearchName, ToolListName, ToolDetailName, ToolCallName:
		return true
	default:
		return false
	}
}

func isDeferrableToolSpec(spec domain.ToolSpec, identity domain.ToolRegistrationIdentity) bool {
	if isCoreVisibleToolSpec(spec) {
		return false
	}
	if identity.Source == domain.ToolSourceMCP || identity.Source == domain.ToolSourcePlugin {
		return true
	}
	switch spec.Category {
	case "mcp", "plugin", "agent", "automation", "admin", "browser":
		return true
	}
	for _, toolset := range spec.Toolsets {
		if toolset == "mcp" || toolset == "plugin" || toolset == "admin" || toolset == "browser" ||
			strings.HasPrefix(toolset, "mcp:") || strings.HasPrefix(toolset, "plugin:") {
			return true
		}
	}
	return false
}

func isCoreVisibleToolSpec(spec domain.ToolSpec) bool {
	switch spec.Name {
	case ToolResolveName, ToolSearchName, ToolListName, ToolDetailName, ToolCallName,
		"read_file", "list_files", "glob", "search_files",
		"lsp_diagnostics", "lsp_definition", "lsp_references", "lsp_symbol_search",
		"web_fetch", "web_search",
		"git_status", "git_diff",
		"write_file", "edit_file", "apply_patch", "format_code",
		"read_diagnostics", "run_tests", "bash",
		"update_plan", "ask_user":
		return true
	default:
		return false
	}
}

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

type ToolActivateFunc func(context.Context, string, string) error

type ToolResolveTool struct {
	registry *Registry
	resolve  ToolResolveFunc
	activate ToolActivateFunc
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

func NewToolResolveTool(registry *Registry, resolve ToolResolveFunc, activate ToolActivateFunc) *ToolResolveTool {
	return &ToolResolveTool{registry: registry, resolve: resolve, activate: activate}
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

func toolResolveSpec(deferredCount int) domain.ToolSpec {
	_ = deferredCount
	return domain.ToolSpec{
		Name: ToolResolveName, Description: "Resolve allowed deferred tools for one concise, specific missing capability. Use only when current tools cannot perform the required action. This does not call tools or bypass permissions.",
		Capability: "tool.resolve", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp", "plugin", "browser"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"intent":    map[string]any{"type": "string", "description": "Concise, specific missing capability. Describe the required action, not a guessed tool name, plan, or broad topic."},
			"required":  map[string]any{"type": "boolean", "description": "Whether the task cannot proceed without a matching tool. Defaults to true."},
			"maxTools":  map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "description": "Maximum number of tools to activate. Defaults to 8."},
			"source":    map[string]any{"type": "string", "description": "Optional source filter, such as mcp, plugin, builtin, or bridge."},
			"category":  map[string]any{"type": "string", "description": "Optional category filter, such as browser, mcp, plugin, automation, or filesystem."},
			"riskLevel": map[string]any{"type": "string", "description": "Optional risk filter, such as low, medium, or high."},
		}, "required": []string{"intent"}, "additionalProperties": false},
	}
}

func toolSearchSpec(deferredCount int) domain.ToolSpec {
	_ = deferredCount
	return domain.ToolSpec{
		Name: ToolSearchName, Description: "Search additional long-tail tools by capability keywords. Returns tool names and descriptions only; use tool_detail to inspect one tool's parameters, then use tool_call to invoke the selected deferred tool.",
		Capability: "tool.search", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Keywords describing the capability you need."},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "description": "Maximum number of matches. Defaults to 5."},
		}, "required": []string{"query"}, "additionalProperties": false},
	}
}

func toolListSpec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: ToolListName, Description: "List available tools with names and descriptions only. Use this for MCP/tool inventory, then call tool_detail for the selected tool's exact input schema.",
		Capability: "tool.list", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"source":          map[string]any{"type": "string", "description": "Optional source filter, such as mcp, plugin, builtin, or bridge."},
			"category":        map[string]any{"type": "string", "description": "Optional category filter, such as mcp, plugin, automation, or filesystem."},
			"query":           map[string]any{"type": "string", "description": "Optional substring filter over name, description, namespace, capability, category, and source."},
			"includeCore":     map[string]any{"type": "boolean", "description": "Include core visible tools such as file, shell, web, bridge, and planning tools. Defaults to true."},
			"includeLongTail": map[string]any{"type": "boolean", "description": "Include deferred long-tail tools from MCP, plugins, automation, and admin sources. Defaults to true."},
			"limit":           map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "description": "Maximum number of tools to return. Defaults to 100."},
			"offset":          map[string]any{"type": "integer", "minimum": 0, "description": "Pagination offset. Defaults to 0."},
		}, "additionalProperties": false},
	}
}

func toolDetailSpec(name string) domain.ToolSpec {
	description := "Load the full JSON schema and metadata for one tool returned by tool_list or tool_search."
	return domain.ToolSpec{
		Name: name, Description: description,
		Capability: "tool.describe", Category: "tool_discovery", RiskLevel: "low", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "Exact tool name returned by tool_list or tool_search."},
		}, "required": []string{"name"}, "additionalProperties": false},
	}
}

func toolCallSpec() domain.ToolSpec {
	return domain.ToolSpec{
		Name: ToolCallName, Description: "Invoke a deferred tool by name with arguments matching its schema. Permissions, toolset checks, and hooks run as for direct calls.",
		Capability: "tool.call", Category: "tool_discovery", RiskLevel: "medium", Toolsets: []string{"safe", "coding", "mcp"},
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{
			"name":      map[string]any{"type": "string", "description": "Exact underlying tool name."},
			"arguments": map[string]any{"type": "object", "description": "Arguments for the underlying tool."},
		}, "required": []string{"name", "arguments"}, "additionalProperties": false},
	}
}

func (t *ToolResolveTool) Spec() domain.ToolSpec { return toolResolveSpec(0) }
func (t *ToolSearchTool) Spec() domain.ToolSpec  { return toolSearchSpec(0) }
func (t *ToolListTool) Spec() domain.ToolSpec    { return toolListSpec() }
func (t *ToolDetailTool) Spec() domain.ToolSpec {
	return toolDetailSpec(firstNonEmpty(t.name, ToolDetailName))
}
func (t *ToolCallTool) Spec() domain.ToolSpec { return toolCallSpec() }

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
	selected := validateToolResolveSelection(candidates, decision.Names, input.MaxTools)
	if len(selected) == 0 {
		return toolResolveNoAvailable(execCtx.ToolCallID, input.Intent, required, firstNonEmpty(decision.Reason, "no candidate tool satisfied the requested capability"))
	}
	items := make([]map[string]any, 0, len(selected))
	for _, entry := range selected {
		if t.activate != nil {
			if err := t.activate(ctx, execCtx.SessionID, entry.Name); err != nil {
				return toolFailure(execCtx.ToolCallID, ToolResolveName, "tool_activation_failed", err.Error())
			}
		}
		items = append(items, toolCatalogListItem(entry))
	}
	structured := map[string]any{
		"status":          "activated",
		"tools":           items,
		"count":           len(items),
		"reason":          strings.TrimSpace(decision.Reason),
		"appliesNextStep": true,
	}
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

func isDeferredToolActivated(spec domain.ToolSpec, activated map[string]bool) bool {
	if activated == nil {
		return false
	}
	name := strings.TrimSpace(spec.Name)
	if activated[name] {
		return true
	}
	if isBrowserToolSpec(spec) && activatedHasBrowserToolGroup(activated) {
		return true
	}
	return false
}

const deferredBrowserToolGroupKey = "toolset:browser"

func activatedHasBrowserToolGroup(activated map[string]bool) bool {
	if activated[deferredBrowserToolGroupKey] || activated["browser"] {
		return true
	}
	for name, ok := range activated {
		if ok && strings.HasPrefix(strings.TrimSpace(name), "browser_") {
			return true
		}
	}
	return false
}

func isBrowserToolSpec(spec domain.ToolSpec) bool {
	if spec.Category == "browser" || strings.HasPrefix(strings.TrimSpace(spec.Name), "browser_") {
		return true
	}
	for _, toolset := range spec.Toolsets {
		if strings.TrimSpace(toolset) == "browser" {
			return true
		}
	}
	return false
}

func deferredToolNameUsedByCall(registry *Registry, call domain.ChatToolCall, result domain.ToolResult) string {
	if registry == nil {
		return ""
	}
	for _, name := range []string{result.Name, call.Name} {
		name = strings.TrimSpace(name)
		if name == "" || isBridgeToolName(name) {
			continue
		}
		if isRegisteredDeferrableTool(registry, name) {
			if strings.HasPrefix(name, "browser_") {
				return deferredBrowserToolGroupKey
			}
			return name
		}
	}
	return ""
}

func isRegisteredDeferrableTool(registry *Registry, name string) bool {
	name = strings.TrimSpace(name)
	if registry == nil || name == "" || isBridgeToolName(name) {
		return false
	}
	for _, entry := range registry.CatalogEntries() {
		if entry.Name != name {
			continue
		}
		spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets}
		return isDeferrableToolSpec(spec, domain.ToolRegistrationIdentity{Source: entry.Source})
	}
	return false
}

func toolCatalogListItem(entry domain.ToolCatalogEntry) map[string]any {
	return map[string]any{
		"name":        entry.Name,
		"description": bounded(entry.Description, 400),
		"source":      entry.Source,
		"sourceId":    entry.SourceID,
		"category":    entry.Category,
		"namespace":   entry.Namespace,
		"capability":  entry.Capability,
		"riskLevel":   entry.RiskLevel,
		"deferred":    isToolCatalogEntryDeferrable(entry),
	}
}

func isToolCatalogEntryDeferrable(entry domain.ToolCatalogEntry) bool {
	spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets}
	return !isBridgeToolName(entry.Name) && isDeferrableToolSpec(spec, domain.ToolRegistrationIdentity{Source: entry.Source})
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

func toolResolveCandidates(registry *Registry, execCtx domain.ToolExecutionContext, source string, category string, riskLevel string) []domain.ToolCatalogEntry {
	if registry == nil {
		return nil
	}
	source = strings.ToLower(strings.TrimSpace(source))
	category = strings.ToLower(strings.TrimSpace(category))
	riskLevel = strings.ToLower(strings.TrimSpace(riskLevel))
	out := []domain.ToolCatalogEntry{}
	for _, entry := range registry.CatalogEntries() {
		spec := domain.ToolSpec{
			Name: entry.Name, Description: entry.Description, InputSchema: entry.InputSchema,
			Namespace: entry.Namespace, Capability: entry.Capability, RiskLevel: entry.RiskLevel,
			Category: entry.Category, Toolsets: entry.Toolsets,
		}
		if isBridgeToolName(entry.Name) || !isDeferrableToolSpec(spec, domain.ToolRegistrationIdentity{Source: entry.Source}) {
			continue
		}
		if len(execCtx.AllowedToolsets) > 0 && !toolSpecInToolsets(spec, execCtx.AllowedToolsets) {
			continue
		}
		if len(visibleToolSpecsForMode(execCtx.AgentMode, []domain.ToolSpec{spec})) == 0 {
			continue
		}
		if source != "" && strings.ToLower(strings.TrimSpace(entry.Source)) != source {
			continue
		}
		if category != "" && strings.ToLower(strings.TrimSpace(entry.Category)) != category {
			continue
		}
		if riskLevel != "" && strings.ToLower(strings.TrimSpace(entry.RiskLevel)) != riskLevel {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func validateToolResolveSelection(candidates []domain.ToolCatalogEntry, names []string, limit int) []domain.ToolCatalogEntry {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	byName := map[string]domain.ToolCatalogEntry{}
	for _, entry := range candidates {
		byName[entry.Name] = entry
	}
	selected := []domain.ToolCatalogEntry{}
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		entry, ok := byName[name]
		if !ok {
			continue
		}
		selected = append(selected, entry)
		seen[name] = true
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func localToolResolve(_ context.Context, request ToolResolveRequest) (ToolResolveDecision, error) {
	matches := searchToolCatalog(request.Candidates, request.Intent, request.MaxTools)
	names := make([]string, 0, len(matches))
	for _, entry := range matches {
		names = append(names, entry.Name)
	}
	return ToolResolveDecision{Names: names, Reason: "matched by local catalog search"}, nil
}

func toolResolveNoAvailable(callID string, intent string, required bool, reason string) domain.ToolResult {
	code := "no_available_tool"
	message := "no available tool matches requested capability: " + strings.TrimSpace(intent)
	if strings.TrimSpace(reason) != "" {
		message += " (" + strings.TrimSpace(reason) + ")"
	}
	result := toolFailure(callID, ToolResolveName, code, message)
	result.Structured = map[string]any{"status": code, "intent": intent, "required": required, "reason": reason}
	return result
}

func deferrableCatalogEntries(registry *Registry) []domain.ToolCatalogEntry {
	if registry == nil {
		return nil
	}
	out := []domain.ToolCatalogEntry{}
	for _, entry := range registry.CatalogEntries() {
		spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets}
		if isBridgeToolName(entry.Name) || !isDeferrableToolSpec(spec, domain.ToolRegistrationIdentity{Source: entry.Source}) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func searchToolCatalog(entries []domain.ToolCatalogEntry, query string, limit int) []domain.ToolCatalogEntry {
	tokens := tokenizeToolQuery(query)
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		entry domain.ToolCatalogEntry
		score int
	}
	scoredEntries := []scored{}
	for _, entry := range entries {
		text := toolCatalogSearchText(entry)
		score := 0
		for _, token := range tokens {
			if strings.Contains(text, token) {
				score++
			}
		}
		if score > 0 {
			scoredEntries = append(scoredEntries, scored{entry: entry, score: score})
		}
	}
	sort.SliceStable(scoredEntries, func(i, j int) bool {
		if scoredEntries[i].score == scoredEntries[j].score {
			return scoredEntries[i].entry.Name < scoredEntries[j].entry.Name
		}
		return scoredEntries[i].score > scoredEntries[j].score
	})
	if limit > len(scoredEntries) {
		limit = len(scoredEntries)
	}
	out := make([]domain.ToolCatalogEntry, 0, limit)
	for _, item := range scoredEntries[:limit] {
		out = append(out, item.entry)
	}
	return out
}

func toolCatalogSearchText(entry domain.ToolCatalogEntry) string {
	parts := []string{
		entry.Name,
		strings.ReplaceAll(entry.Name, "_", " "),
		entry.Description,
		entry.Namespace,
		entry.Capability,
		entry.Category,
		entry.Source,
		entry.SourceID,
		strings.Join(entry.Toolsets, " "),
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func countDeferredCatalogSources(entries []domain.ToolCatalogEntry) map[string]int {
	counts := map[string]int{}
	for _, entry := range entries {
		source := strings.TrimSpace(entry.Source)
		if source == "" {
			source = "unknown"
		}
		counts[source]++
	}
	return counts
}

func tokenizeToolQuery(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			out = append(out, field)
		}
	}
	return out
}
