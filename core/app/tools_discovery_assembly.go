package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

const (
	ToolResolveName                     = "tool_resolve"
	ToolSearchName                      = "tool_search"
	ToolListName                        = "tool_list"
	ToolDetailName                      = "tool_detail"
	ToolCallName                        = "tool_call"
	providerDeclarationActivationPolicy = "provider_declaration"
	providerAccountActivationPolicy     = "provider_account"
)

type ToolAssemblyResult struct {
	Specs                 []domain.ToolSpec
	Activated             bool
	DeferredCount         int
	ExpectedRegistrations map[string]domain.ToolRegistrationIdentity
	Snapshot              domain.ToolSnapshot
}

func AssembleToolSpecs(registry *Registry, specs []domain.ToolSpec) ToolAssemblyResult {
	return AssembleToolSpecsWithSources(registry, specs, nil)
}

func AssembleToolSpecsWithActivated(registry *Registry, specs []domain.ToolSpec, activated map[string]bool) ToolAssemblyResult {
	sources := map[string]string{}
	for name, active := range activated {
		if active {
			sources[name] = "currentTurn"
		}
	}
	return AssembleToolSpecsWithSources(registry, specs, sources)
}

func AssembleToolSpecsWithSources(registry *Registry, specs []domain.ToolSpec, activationSources map[string]string) ToolAssemblyResult {
	allIdentities := map[string]domain.ToolRegistrationIdentity{}
	if registry != nil {
		for _, entry := range registry.CatalogEntries() {
			allIdentities[entry.Name] = domain.ToolRegistrationIdentity{
				Name: entry.Name, RegistrationID: entry.RegistrationID, SchemaHash: entry.SchemaHash, Source: entry.Source, SourceID: entry.SourceID, Version: entry.Version, ImplementationHash: entry.ImplementationHash,
			}
		}
	}
	visible := make([]domain.ToolSpec, 0, len(specs))
	deferredCount := 0
	for _, spec := range specs {
		if spec.Name == ToolResolveName {
			visible = append(visible, spec)
			continue
		}
		if isBridgeToolName(spec.Name) {
			continue
		}
		if spec.ActivationPolicy == providerDeclarationActivationPolicy {
			if activationSources[spec.Name] == "providerCapability" {
				visible = append(visible, spec)
			}
			continue
		}
		if spec.ActivationPolicy == providerAccountActivationPolicy {
			if activationSources[spec.Name] == "providerAccount" {
				visible = append(visible, spec)
			}
			continue
		}
		if isCoreVisibleToolSpec(spec) {
			if activationSources[spec.Name] != "disabled" {
				visible = append(visible, spec)
			}
			continue
		}
		if !isDeferrableToolSpec(spec, allIdentities[spec.Name]) {
			continue
		}
		deferredCount++
		if strings.TrimSpace(activationSources[spec.Name]) != "" {
			visible = append(visible, spec)
		}
	}
	identities := map[string]domain.ToolRegistrationIdentity{}
	entries := make([]domain.ToolSnapshotEntry, 0, len(visible))
	for _, spec := range visible {
		identity, ok := allIdentities[spec.Name]
		if !ok {
			continue
		}
		identities[spec.Name] = identity
		activationSource := firstNonEmpty(activationSources[spec.Name], "currentTurn")
		if isCoreVisibleToolSpec(spec) {
			activationSource = "core"
		} else if spec.Name == ToolResolveName {
			activationSource = "control"
		}
		entries = append(entries, domain.ToolSnapshotEntry{Name: spec.Name, RegistrationID: identity.RegistrationID, SchemaHash: toolSchemaHash(spec), SourceID: identity.SourceID, SourceVersion: identity.Version, ActivationSource: activationSource})
	}
	snapshotRaw, _ := json.Marshal(entries)
	sum := sha256.Sum256(snapshotRaw)
	snapshot := domain.ToolSnapshot{Revision: hex.EncodeToString(sum[:]), Tools: entries}
	activated := false
	for _, spec := range visible {
		if !isCoreVisibleToolSpec(spec) && spec.Name != ToolResolveName {
			activated = true
			break
		}
	}
	return ToolAssemblyResult{Specs: visible, Activated: activated, DeferredCount: deferredCount, ExpectedRegistrations: identities, Snapshot: snapshot}
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
	if isBridgeToolName(spec.Name) || spec.Name == SkillToolName {
		return false
	}
	_ = identity
	return true
}

func isCoreVisibleToolSpec(spec domain.ToolSpec) bool {
	switch spec.Name {
	case "read", "bash", "edit", "write", "update_plan", "ask_user":
		return true
	default:
		return false
	}
}
