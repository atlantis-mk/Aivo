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
	tool           domain.Tool
	registrationID string
	source         string
	sourceID       string
	version        string
	enabled        bool
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string][]registeredTool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string][]registeredTool{}}
}

func (r *Registry) Register(tool domain.Tool) error {
	if tool == nil {
		return errors.New("tool is required")
	}
	spec := tool.Spec()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return errors.New("tool name is required")
	}
	return r.RegisterScoped(tool, domain.ToolSourceBuiltin, "", "")
}

func (r *Registry) RegisterScoped(tool domain.Tool, source string, sourceID string, version string) error {
	if tool == nil {
		return errors.New("tool is required")
	}
	spec := tool.Spec()
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return errors.New("tool name is required")
	}
	if source == "" {
		source = domain.ToolSourceBuiltin
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if source == domain.ToolSourceBuiltin && len(r.tools[name]) > 0 {
		return fmt.Errorf("tool %q is already registered", name)
	}
	r.tools[name] = append(r.tools[name], registeredTool{
		tool: tool, registrationID: toolRegistrationID(spec, source, sourceID, version),
		source: source, sourceID: sourceID, version: version, enabled: true,
	})
	return nil
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
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
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
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]domain.ToolCatalogEntry, 0, len(names))
	for _, name := range names {
		reg, ok := r.effectiveLocked(name)
		if !ok {
			continue
		}
		spec := reg.tool.Spec()
		out = append(out, domain.ToolCatalogEntry{
			Name: spec.Name, Description: spec.Description, InputSchema: spec.InputSchema,
			Namespace: spec.Namespace, Capability: spec.Capability, RiskLevel: spec.RiskLevel,
			Category: spec.Category, Toolsets: spec.Toolsets, Source: reg.source, SourceID: reg.sourceID,
			RegistrationID: reg.registrationID, Enabled: reg.enabled,
		})
	}
	return out
}

func toolRegistrationID(spec domain.ToolSpec, source string, sourceID string, version string) string {
	raw, _ := json.Marshal(map[string]any{
		"name": spec.Name, "source": source, "sourceID": sourceID, "version": version,
		"capability": spec.Capability, "schema": spec.InputSchema,
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:12])
}

func identityForRegisteredTool(name string, reg registeredTool) domain.ToolRegistrationIdentity {
	return domain.ToolRegistrationIdentity{Name: name, RegistrationID: reg.registrationID, Source: reg.source, SourceID: reg.sourceID, Version: reg.version}
}

func NewReadOnlyToolRegistry(workspaceRoot string) (*Registry, error) {
	registry := NewRegistry()
	for _, tool := range []domain.Tool{
		NewReadFileTool(workspaceRoot),
		NewListFilesTool(workspaceRoot),
		NewGlobTool(workspaceRoot),
		NewSearchFilesTool(workspaceRoot),
		NewLSPDiagnosticsTool(workspaceRoot),
		NewLSPDefinitionTool(workspaceRoot),
		NewLSPReferencesTool(workspaceRoot),
		NewLSPSymbolSearchTool(workspaceRoot),
		NewWebFetchTool(),
		NewWebSearchTool(),
	} {
		if err := registry.Register(tool); err != nil {
			return nil, err
		}
	}
	if workspaceHasGit(workspaceRoot) {
		for _, tool := range []domain.Tool{
			NewGitStatusTool(workspaceRoot),
			NewGitDiffTool(workspaceRoot),
		} {
			if err := registry.Register(tool); err != nil {
				return nil, err
			}
		}
	}
	return registry, nil
}

func NewCodingToolRegistry(workspaceRoot string) (*Registry, error) {
	return NewCodingToolRegistryWithShellOutputSink(workspaceRoot, nil)
}

func NewCodingToolRegistryWithShellOutputSink(workspaceRoot string, outputSink ShellOutputSink) (*Registry, error) {
	registry, err := NewReadOnlyToolRegistry(workspaceRoot)
	if err != nil {
		return nil, err
	}
	for _, tool := range []domain.Tool{
		NewWriteFileTool(workspaceRoot),
		NewEditFileTool(workspaceRoot),
		NewFormatCodeTool(workspaceRoot, nil, outputSink),
	} {
		if err := registry.Register(tool); err != nil {
			return nil, err
		}
	}
	if err := registry.Register(NewApplyPatchTool(workspaceRoot)); err != nil {
		return nil, err
	}
	runner := NewLocalSandboxRunner()
	for _, tool := range []domain.Tool{
		NewReadDiagnosticsTool(workspaceRoot, runner, outputSink),
		NewRunTestsTool(workspaceRoot, runner, outputSink),
		NewBashTool(workspaceRoot, runner, outputSink),
		NewExecCommandTool(workspaceRoot, defaultAgentPTYRegistry, outputSink),
		NewWriteStdinTool(workspaceRoot, defaultAgentPTYRegistry),
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
