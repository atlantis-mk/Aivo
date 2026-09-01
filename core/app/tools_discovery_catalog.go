package app

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"aivo/core/domain"
)

func isDeferredToolActivated(spec domain.ToolSpec, activated map[string]bool) bool {
	if activated == nil {
		return false
	}
	name := strings.TrimSpace(spec.Name)
	if activated[name] {
		return true
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
		if entry.ActivationPolicy == providerDeclarationActivationPolicy || entry.ActivationPolicy == providerAccountActivationPolicy {
			return false
		}
		spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets, ActivationPolicy: entry.ActivationPolicy}
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
	if entry.ActivationPolicy == providerDeclarationActivationPolicy || entry.ActivationPolicy == providerAccountActivationPolicy {
		return false
	}
	spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets, ActivationPolicy: entry.ActivationPolicy}
	return !isBridgeToolName(entry.Name) && isDeferrableToolSpec(spec, domain.ToolRegistrationIdentity{Source: entry.Source})
}

func isStandaloneToolCatalogEntry(entry domain.ToolCatalogEntry) bool {
	if entry.ActivationPolicy == providerDeclarationActivationPolicy || entry.ActivationPolicy == providerAccountActivationPolicy {
		return false
	}
	return entry.SelectionGroup == nil && (entry.Source == domain.ToolSourceBuiltin ||
		(entry.Source == domain.ToolSourceExtension && strings.HasPrefix(entry.SourceID, "aivo.")))
}

func resourceResolveCandidates(registry *Registry, execCtx domain.ToolExecutionContext, source string, category string, riskLevel string) []domain.ToolCatalogEntry {
	if registry == nil {
		return nil
	}
	source = strings.ToLower(strings.TrimSpace(source))
	category = strings.ToLower(strings.TrimSpace(category))
	riskLevel = strings.ToLower(strings.TrimSpace(riskLevel))
	out := []domain.ToolCatalogEntry{}
	for _, entry := range registry.CatalogEntries() {
		if entry.ActivationPolicy == providerDeclarationActivationPolicy || entry.ActivationPolicy == providerAccountActivationPolicy {
			continue
		}
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

func toolAccessCatalogEntries(registry *Registry, execCtx domain.ToolExecutionContext) []domain.ToolCatalogEntry {
	if registry == nil {
		return nil
	}
	if execCtx.ToolSnapshot == nil {
		return registry.CatalogEntries()
	}
	allowed := map[string]bool{}
	for _, item := range execCtx.ToolSnapshot.Tools {
		name := strings.TrimSpace(item.Name)
		if name != "" && !isBridgeToolName(name) {
			allowed[name] = true
		}
	}
	out := make([]domain.ToolCatalogEntry, 0, len(allowed))
	for _, entry := range registry.CatalogEntries() {
		if allowed[entry.Name] {
			out = append(out, entry)
		}
	}
	return out
}

func validateResourceResolveToolSelection(candidates []domain.ToolCatalogEntry, names []string) []domain.ToolCatalogEntry {
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
	}
	return selected
}

func localResourceResolve(_ context.Context, request ResourceResolveRequest) (ResourceResolveDecision, error) {
	if request.Mode == string(hostResourceSelectionInspect) {
		matches := request.Candidates
		if !isToolInventoryIntent(request.Intent) {
			matches = searchToolCatalog(request.Candidates, request.Intent, 0)
		}
		groups := hostToolGroupCandidates(matches)
		_, names, _ := expandHostToolGroups(groups, nil, false)
		return ResourceResolveDecision{
			Names:     names,
			Resources: hostResourceSelectionResources(groups, normalizeDeferredToolNames(names)),
			Reason:    "matched by local catalog inspection",
		}, nil
	}
	matches := searchToolCatalog(request.Candidates, request.Intent, 0)
	names := make([]string, 0, len(matches))
	for _, entry := range matches {
		names = append(names, entry.Name)
	}
	return ResourceResolveDecision{Names: names, Reason: "matched by local catalog search"}, nil
}

func resourceResolveNoAvailable(callID string, intent string, required bool, reason string) domain.ToolResult {
	code := "no_available_resource"
	message := "no available resource matches requested capability: " + strings.TrimSpace(intent)
	if strings.TrimSpace(reason) != "" {
		message += " (" + strings.TrimSpace(reason) + ")"
	}
	result := toolFailure(callID, ResourceResolveName, code, message)
	result.Structured = map[string]any{"status": code, "intent": intent, "required": required, "reason": reason}
	return result
}

func deferrableCatalogEntries(registry *Registry) []domain.ToolCatalogEntry {
	if registry == nil {
		return nil
	}
	return deferrableCatalogEntriesFrom(registry.CatalogEntries())
}

func deferrableCatalogEntriesFrom(entries []domain.ToolCatalogEntry) []domain.ToolCatalogEntry {
	out := []domain.ToolCatalogEntry{}
	for _, entry := range entries {
		if entry.ActivationPolicy == providerDeclarationActivationPolicy || entry.ActivationPolicy == providerAccountActivationPolicy {
			continue
		}
		spec := domain.ToolSpec{Name: entry.Name, Category: entry.Category, Toolsets: entry.Toolsets, ActivationPolicy: entry.ActivationPolicy}
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
	if limit <= 0 || limit > len(scoredEntries) {
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
