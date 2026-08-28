package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

type ToolRegistry interface {
	Register(tool domain.Tool) error
	Get(name string) (domain.Tool, bool)
	Specs() []domain.ToolSpec
	SpecsForToolsets(toolsets []string) []domain.ToolSpec
}

type registeredTool struct {
	tool               domain.Tool
	registrationID     string
	schemaHash         string
	source             string
	sourceID           string
	version            string
	implementationHash string
	enabled            bool
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string][]registeredTool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string][]registeredTool{}, order: []string{}}
}

func (r *Registry) Register(tool domain.Tool) error {
	if tool == nil {
		return errors.New("tool is required")
	}
	spec := tool.Spec()
	name := spec.Name
	if err := validateCanonicalToolName(name); err != nil {
		return err
	}
	if err := validateToolSelectionGroup(spec.SelectionGroup); err != nil {
		return fmt.Errorf("tool %q: %w", name, err)
	}
	return r.RegisterScoped(tool, domain.ToolSourceBuiltin, "", "")
}

func (r *Registry) RegisterScoped(tool domain.Tool, source string, sourceID string, version string) error {
	if tool == nil {
		return errors.New("tool is required")
	}
	spec := tool.Spec()
	name := spec.Name
	if err := validateCanonicalToolName(name); err != nil {
		return err
	}
	if err := validateToolSelectionGroup(spec.SelectionGroup); err != nil {
		return fmt.Errorf("tool %q: %w", name, err)
	}
	if source == "" {
		source = domain.ToolSourceBuiltin
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if isReservedCoreToolName(name) && source != domain.ToolSourceBuiltin {
		return fmt.Errorf("tool %q is reserved by the core execution environment", name)
	}
	if isBridgeToolName(name) && source != domain.ToolSourceBridge {
		return fmt.Errorf("tool %q is reserved by the Host control plane", name)
	}
	selectionGroups := map[string]domain.ToolSelectionGroup{}
	if spec.SelectionGroup != nil {
		selectionGroups[spec.SelectionGroup.ID] = *spec.SelectionGroup
	}
	if err := r.validateSelectionGroupsLocked(selectionGroups, source, sourceID, map[string]bool{name: true}); err != nil {
		return err
	}
	if existing := r.tools[name]; len(existing) > 0 {
		current := existing[len(existing)-1]
		if current.source != source || current.sourceID != sourceID || source == domain.ToolSourceBuiltin {
			return fmt.Errorf("tool %q is already registered", name)
		}
		registrationID := toolRegistrationID(spec, source, sourceID, version)
		if current.registrationID == registrationID {
			return fmt.Errorf("tool %q registration is unchanged", name)
		}
		r.tools[name] = append(r.tools[name], registeredTool{tool: tool, registrationID: registrationID, schemaHash: toolSchemaHash(spec), source: source, sourceID: sourceID, version: version, implementationHash: spec.ImplementationHash, enabled: true})
		return nil
	}
	r.order = append(r.order, name)
	r.tools[name] = append(r.tools[name], registeredTool{
		tool: tool, registrationID: toolRegistrationID(spec, source, sourceID, version), schemaHash: toolSchemaHash(spec),
		source: source, sourceID: sourceID, version: version, enabled: true,
		implementationHash: spec.ImplementationHash,
	})
	return nil
}

func (r *Registry) RegisterScopedBatch(tools []domain.Tool, source, sourceID, version string) error {
	if len(tools) == 0 {
		return nil
	}
	if source == "" {
		source = domain.ToolSourceBuiltin
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := map[string]bool{}
	selectionGroups := map[string]domain.ToolSelectionGroup{}
	for _, tool := range tools {
		if tool == nil {
			return errors.New("tool is required")
		}
		name := tool.Spec().Name
		if err := validateCanonicalToolName(name); err != nil {
			return err
		}
		if err := validateToolSelectionGroup(tool.Spec().SelectionGroup); err != nil {
			return fmt.Errorf("tool %q: %w", name, err)
		}
		if group := tool.Spec().SelectionGroup; group != nil {
			if existing, ok := selectionGroups[group.ID]; ok && existing != *group {
				return fmt.Errorf("selection group %q has inconsistent metadata", group.ID)
			}
			selectionGroups[group.ID] = *group
		}
		if seen[name] {
			return fmt.Errorf("tool %q is duplicated in the registration batch", name)
		}
		seen[name] = true
		if isReservedCoreToolName(name) && source != domain.ToolSourceBuiltin {
			return fmt.Errorf("tool %q is reserved by the core execution environment", name)
		}
		if isBridgeToolName(name) && source != domain.ToolSourceBridge {
			return fmt.Errorf("tool %q is reserved by the Host control plane", name)
		}
		if existing := r.tools[name]; len(existing) > 0 {
			current := existing[len(existing)-1]
			if current.source != source || current.sourceID != sourceID || source == domain.ToolSourceBuiltin {
				return fmt.Errorf("tool %q is already registered", name)
			}
			if current.registrationID == toolRegistrationID(tool.Spec(), source, sourceID, version) {
				return fmt.Errorf("tool %q registration is unchanged", name)
			}
		}
	}
	if err := r.validateSelectionGroupsLocked(selectionGroups, source, sourceID, seen); err != nil {
		return err
	}
	for _, tool := range tools {
		spec := tool.Spec()
		name := strings.TrimSpace(spec.Name)
		if len(r.tools[name]) == 0 {
			r.order = append(r.order, name)
		}
		r.tools[name] = append(r.tools[name], registeredTool{
			tool: tool, registrationID: toolRegistrationID(spec, source, sourceID, version), schemaHash: toolSchemaHash(spec),
			source: source, sourceID: sourceID, version: version, implementationHash: spec.ImplementationHash, enabled: true,
		})
	}
	return nil
}

func (r *Registry) validateSelectionGroupsLocked(groups map[string]domain.ToolSelectionGroup, source, sourceID string, replacing map[string]bool) error {
	if len(groups) == 0 {
		return nil
	}
	for name := range r.tools {
		if replacing[name] {
			continue
		}
		registration, ok := r.effectiveLocked(name)
		if !ok {
			continue
		}
		group := registration.tool.Spec().SelectionGroup
		if group == nil {
			continue
		}
		candidate, exists := groups[group.ID]
		if !exists {
			continue
		}
		if registration.source != source || registration.sourceID != sourceID {
			return fmt.Errorf("selection group %q is already registered by another source", group.ID)
		}
		if candidate != *group {
			return fmt.Errorf("selection group %q has inconsistent metadata", group.ID)
		}
	}
	return nil
}

func (r *Registry) GetRegisteredForSnapshot(name, registrationID string) (domain.Tool, domain.ToolRegistrationIdentity, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name = strings.TrimSpace(name)
	registrationID = strings.TrimSpace(registrationID)
	if registrationID == "" {
		reg, ok := r.effectiveLocked(name)
		if !ok {
			return nil, domain.ToolRegistrationIdentity{}, false
		}
		return reg.tool, identityForRegisteredTool(name, reg), true
	}
	stack := r.tools[name]
	for index := len(stack) - 1; index >= 0; index-- {
		reg := stack[index]
		if reg.enabled && reg.registrationID == registrationID {
			return reg.tool, identityForRegisteredTool(name, reg), true
		}
	}
	return nil, domain.ToolRegistrationIdentity{}, false
}

func (r *Registry) Get(name string) (domain.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.effectiveLocked(strings.TrimSpace(name))
	if !ok {
		return nil, false
	}
	return reg.tool, true
}

func (r *Registry) GetRegistered(name string) (domain.Tool, domain.ToolRegistrationIdentity, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.effectiveLocked(strings.TrimSpace(name))
	if !ok {
		return nil, domain.ToolRegistrationIdentity{}, false
	}
	return reg.tool, identityForRegisteredTool(name, reg), true
}

func (r *Registry) IdentityFor(name string) (domain.ToolRegistrationIdentity, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.effectiveLocked(strings.TrimSpace(name))
	if !ok {
		return domain.ToolRegistrationIdentity{}, false
	}
	return identityForRegisteredTool(name, reg), true
}

func (r *Registry) effectiveLocked(name string) (registeredTool, bool) {
	stack := r.tools[name]
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].enabled && stack[i].tool != nil {
			return stack[i], true
		}
	}
	return registeredTool{}, false
}

func (r *Registry) Specs() []domain.ToolSpec {
	return r.specsForFilter(nil)
}

func (r *Registry) SpecsForToolsets(toolsets []string) []domain.ToolSpec {
	return r.specsForFilter(func(spec domain.ToolSpec) bool {
		return toolSpecInToolsets(spec, toolsets)
	})
}

func (r *Registry) specsForFilter(allow func(domain.ToolSpec) bool) []domain.ToolSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := r.orderedNamesLocked()
	specs := make([]domain.ToolSpec, 0, len(names))
	for _, name := range names {
		reg, ok := r.effectiveLocked(name)
		if !ok {
			continue
		}
		spec := reg.tool.Spec()
		if allow != nil && !allow(spec) {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func (r *Registry) CatalogEntries() []domain.ToolCatalogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := r.orderedNamesLocked()
	out := make([]domain.ToolCatalogEntry, 0, len(names))
	for _, name := range names {
		reg, ok := r.effectiveLocked(name)
		if !ok {
			continue
		}
		spec := reg.tool.Spec()
		out = append(out, domain.ToolCatalogEntry{
			Name: spec.Name, Description: spec.Description, InputSchema: spec.InputSchema,
			Namespace: spec.Namespace, NamespaceDescription: spec.NamespaceDescription, Capability: spec.Capability, RiskLevel: spec.RiskLevel,
			Category: spec.Category, Toolsets: spec.Toolsets, Source: reg.source, SourceID: reg.sourceID,
			RegistrationID: reg.registrationID, SchemaHash: reg.schemaHash, Version: reg.version, ImplementationHash: reg.implementationHash, Enabled: reg.enabled, ActivationPolicy: spec.ActivationPolicy,
			SelectionGroup: cloneToolSelectionGroup(spec.SelectionGroup),
		})
	}
	return out
}

func (r *Registry) orderedNamesLocked() []string {
	preferred := []string{"read", "bash", "edit", "write", "update_plan", "ask_user", "grep", "find", "ls"}
	seen := map[string]bool{}
	names := make([]string, 0, len(r.tools))
	for _, name := range preferred {
		if _, ok := r.tools[name]; ok {
			names = append(names, name)
			seen[name] = true
		}
	}
	remaining := make([]string, 0, len(r.tools))
	for _, name := range r.order {
		if !seen[name] {
			remaining = append(remaining, name)
			seen[name] = true
		}
	}
	for name := range r.tools {
		if !seen[name] {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	return append(names, remaining...)
}

func isReservedCoreToolName(name string) bool {
	switch strings.TrimSpace(name) {
	case "read", "bash", "edit", "write", "update_plan", "ask_user":
		return true
	default:
		return false
	}
}

func toolRegistrationID(spec domain.ToolSpec, source string, sourceID string, version string) string {
	raw, _ := json.Marshal(map[string]any{
		"name": spec.Name, "source": source, "sourceID": sourceID, "version": version,
		"capability": spec.Capability, "schema": spec.InputSchema, "selectionGroup": spec.SelectionGroup, "implementationHash": spec.ImplementationHash,
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:12])
}

func validateToolSelectionGroup(group *domain.ToolSelectionGroup) error {
	if group == nil {
		return nil
	}
	if err := validateCanonicalToolName(strings.TrimSpace(group.ID)); err != nil {
		return fmt.Errorf("selection group id is invalid: %w", err)
	}
	if strings.TrimSpace(group.ID) != group.ID {
		return errors.New("selection group id must be canonical")
	}
	name := strings.TrimSpace(group.Name)
	description := strings.TrimSpace(group.Description)
	if name == "" || len([]rune(name)) > 100 {
		return errors.New("selection group name is required and must be at most 100 characters")
	}
	if len([]rune(description)) > 500 {
		return errors.New("selection group description must be at most 500 characters")
	}
	return nil
}

func cloneToolSelectionGroup(group *domain.ToolSelectionGroup) *domain.ToolSelectionGroup {
	if group == nil {
		return nil
	}
	copy := *group
	return &copy
}

func toolSchemaHash(spec domain.ToolSpec) string {
	raw, _ := json.Marshal(spec.InputSchema)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func identityForRegisteredTool(name string, reg registeredTool) domain.ToolRegistrationIdentity {
	return domain.ToolRegistrationIdentity{Name: name, RegistrationID: reg.registrationID, SchemaHash: reg.schemaHash, Source: reg.source, SourceID: reg.sourceID, Version: reg.version, ImplementationHash: reg.implementationHash}
}

func NewReadOnlyToolRegistry(workspaceRoot string) (*Registry, error) {
	registry := NewRegistry()
	if err := registry.Register(NewReadTool(workspaceRoot)); err != nil {
		return nil, err
	}
	return registry, nil
}

func NewCodingToolRegistry(workspaceRoot string) (*Registry, error) {
	return NewCodingToolRegistryWithShellOutputSink(workspaceRoot, nil)
}

func NewCodingToolRegistryWithShellOutputSink(workspaceRoot string, outputSink ShellOutputSink) (*Registry, error) {
	return NewCodingToolRegistryWithExecutionEnvironment(workspaceRoot, outputSink, nil)
}

func NewCodingToolRegistryWithExecutionEnvironment(workspaceRoot string, outputSink ShellOutputSink, environment ExecutionEnvironment) (*Registry, error) {
	registry := NewRegistry()
	runner := NewLocalSandboxRunner()
	read := NewReadTool(workspaceRoot)
	read.environment = environment
	bash := NewBashTool(workspaceRoot, runner, outputSink)
	bash.environment = environment
	edit := NewEditTool(workspaceRoot)
	edit.environment = environment
	write := NewWriteTool(workspaceRoot)
	write.environment = environment
	grep := NewSearchFilesTool(workspaceRoot)
	grep.environment = environment
	find := NewGlobTool(workspaceRoot)
	find.environment = environment
	ls := NewListFilesTool(workspaceRoot)
	ls.environment = environment
	for _, tool := range []domain.Tool{
		read, bash, edit, write, grep, find, ls,
	} {
		if err := registry.Register(tool); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func workspaceHasGit(workspaceRoot string) bool {
	if strings.TrimSpace(workspaceRoot) == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workspaceRoot
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		return killProcessGroup(cmd.Process)
	}
	cmd.WaitDelay = 100 * time.Millisecond
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return err == nil && strings.TrimSpace(out.String()) == "true"
}
