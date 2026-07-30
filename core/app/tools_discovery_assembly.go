package app

import (
	"strings"

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
	case "mcp", "plugin", "agent", "automation", "admin":
		return true
	}
	for _, toolset := range spec.Toolsets {
		if toolset == "mcp" || toolset == "plugin" || toolset == "admin" ||
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
		SkillToolName,
		"git_status", "git_diff",
		"write_file", "edit_file", "apply_patch", "format_code",
		"read_diagnostics", "run_tests", "bash",
		"update_plan", "ask_user":
		return true
	default:
		return false
	}
}
