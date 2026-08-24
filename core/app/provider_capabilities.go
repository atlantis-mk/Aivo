package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aivo/core/domain"
)

type modelRequirement struct {
	Tools         bool
	ToolCallCount int
	InputTokens   int
	Streaming     bool
	Reasoning     bool
}

func requestRequirements(chatRequest domain.ChatRequest, reasoningEffort string, onDelta func(string)) modelRequirement {
	return modelRequirement{
		Tools:         len(chatRequest.Tools) > 0,
		ToolCallCount: len(chatRequest.Tools),
		InputTokens:   estimateChatRequestInputTokens(chatRequest),
		Streaming:     onDelta != nil,
		Reasoning:     reasoningRequested(reasoningEffort),
	}
}

func estimateChatRequestInputTokens(chatRequest domain.ChatRequest) int {
	var chars int
	for _, message := range chatRequest.Messages {
		chars += len([]rune(message.Role)) + len([]rune(message.Text)) + len([]rune(message.Name))
		for _, call := range message.ToolCalls {
			chars += len([]rune(call.ID)) + len([]rune(call.Name)) + len([]rune(string(call.Arguments)))
		}
	}
	for _, tool := range chatRequest.Tools {
		chars += len([]rune(tool.Name)) + len([]rune(tool.Description))
		if len(tool.InputSchema) > 0 {
			raw, _ := json.Marshal(tool.InputSchema)
			chars += len([]rune(string(raw)))
		}
	}
	if chars == 0 {
		return 0
	}
	return estimateTokens(strings.Repeat("x", chars))
}

func reasoningRequested(reasoningEffort string) bool {
	effort := normalizeReasoningEffort(reasoningEffort)
	return effort == "low" || effort == "high" || effort == "ultra"
}

func (s *Service) validateModelCapabilities(ctx context.Context, route ResolvedModelRoute, req modelRequirement) error {
	model, ok := s.modelInfoForRoute(ctx, route)
	if !ok {
		return nil
	}
	if model.Deprecated || strings.EqualFold(model.Status, "deprecated") {
		return fmt.Errorf("model unavailable: %s/%s is deprecated", route.Model.ProviderID, route.Model.ModelID)
	}
	// Several OpenAI-compatible /models endpoints return only an id and name.
	// Treat that shape as unknown capability metadata instead of interpreting
	// every omitted capability as explicitly unsupported. The provider request
	// remains the source of truth in that case.
	if !modelCapabilityMetadataKnown(model) {
		return nil
	}
	if req.Tools && !modelSupportsCapability(model, "tools") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support tools", route.Model.ProviderID, route.Model.ModelID)
	}
	if req.Streaming && !modelSupportsCapability(model, "streaming") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support streaming", route.Model.ProviderID, route.Model.ModelID)
	}
	if req.Reasoning && !modelSupportsCapability(model, "reasoning") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support reasoning controls", route.Model.ProviderID, route.Model.ModelID)
	}
	return nil
}

func modelCapabilityMetadataKnown(model domain.ModelInfo) bool {
	return len(model.Capabilities) > 0 || model.Streaming || model.ToolSupport || len(model.ReasoningControls) > 0
}

func (s *Service) modelInfoForRoute(ctx context.Context, route ResolvedModelRoute) (domain.ModelInfo, bool) {
	if model, ok := findModelInfo(route.Definition.Models, route.Model.ModelID); ok {
		return model, true
	}
	if s != nil && s.store != nil {
		if cache, err := s.store.LoadProviderModelCache(ctx, route.Provider.ID); err == nil && cache != nil {
			if model, ok := findModelInfo(cache.Models, route.Model.ModelID); ok {
				return model, true
			}
		}
	}
	return domain.ModelInfo{}, false
}

func findModelInfo(models []domain.ModelInfo, modelID string) (domain.ModelInfo, bool) {
	modelID = strings.TrimSpace(modelID)
	for _, model := range models {
		if strings.TrimSpace(model.ID) == modelID {
			return model, true
		}
	}
	return domain.ModelInfo{}, false
}

func modelSupportsCapability(model domain.ModelInfo, capability string) bool {
	switch capability {
	case "tools":
		if model.ToolSupport {
			return true
		}
	case "streaming":
		if model.Streaming {
			return true
		}
	case "reasoning":
		if len(model.ReasoningControls) > 0 {
			return true
		}
	}
	return containsString(model.Capabilities, capability)
}
