package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"aivo/core/domain"
)

// ExecutionEnvironment owns the coherent filesystem, process, and artifact
// namespace behind all four reserved primitives and optional built-in file
// tools. Implementations must never fall back to a different environment after
// selection.
type ExecutionEnvironment interface {
	Identity() string
	ExecutePrimitive(context.Context, string, json.RawMessage, domain.ToolExecutionContext) domain.ToolResult
}

type extensionExecutionEnvironment struct {
	supervisor  *ExtensionSupervisor
	extensionID string
	generation  string
	environment string
}

func (e *extensionExecutionEnvironment) Identity() string {
	return e.extensionID + "@" + e.generation + ":" + e.environment
}

func (e *extensionExecutionEnvironment) ExecutePrimitive(ctx context.Context, operation string, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	if e == nil || e.supervisor == nil {
		return primitiveError(operation, "environment_unavailable", errors.New("execution environment is unavailable"))
	}
	result := e.supervisor.executeGeneration(ctx, e.extensionID, e.generation, "environment."+operation, args, execCtx)
	result.Name = operation
	if result.ToolError != nil && strings.HasPrefix(result.ToolError.Code, "extension_") {
		result.ToolError.Code = "environment_unavailable"
	}
	return result
}

func (s *ExtensionSupervisor) ExecutionEnvironment(id string) (ExecutionEnvironment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.itemLocked(id)
	if err != nil {
		return nil, err
	}
	if item.loaded.Manifest.Contributes.Environment == nil || strings.TrimSpace(item.loaded.Manifest.Contributes.Environment.ID) == "" {
		return nil, errors.New("extension does not contribute an execution environment")
	}
	if item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive {
		return nil, errors.New("extension execution environment is not ready")
	}
	return &extensionExecutionEnvironment{supervisor: s, extensionID: id, generation: item.loaded.Integrity, environment: item.loaded.Manifest.Contributes.Environment.ID}, nil
}

func executionEnvironmentHash(environment ExecutionEnvironment) string {
	if environment == nil {
		return ""
	}
	return "environment:" + environment.Identity()
}
