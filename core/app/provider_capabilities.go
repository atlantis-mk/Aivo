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
	return effort == "minimal" || effort == "low" || effort == "high" || effort == "xhigh" || effort == "max" || effort == "ultra"
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
	if req.Tools && modelCapabilityMetadataKnownFor(model, "tools") && !modelSupportsCapability(model, "tools") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support tools", route.Model.ProviderID, route.Model.ModelID)
	}
	if req.Streaming && modelCapabilityMetadataKnownFor(model, "streaming") && !modelSupportsCapability(model, "streaming") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support streaming", route.Model.ProviderID, route.Model.ModelID)
	}
	if req.Reasoning && modelCapabilityMetadataKnownFor(model, "reasoning") && !modelSupportsCapability(model, "reasoning") {
		return fmt.Errorf("model capability unsupported: %s/%s does not support reasoning controls", route.Model.ProviderID, route.Model.ModelID)
	}
	return nil
}

func modelCapabilityMetadataKnown(model domain.ModelInfo) bool {
	return len(model.DeclaredCapabilities) > 0 || len(model.Capabilities) > 0 || model.Streaming || model.ToolSupport || len(model.ReasoningControls) > 0
}

func modelCapabilityMetadataKnownFor(model domain.ModelInfo, capability string) bool {
	if len(model.DeclaredCapabilities) > 0 {
		return containsString(model.DeclaredCapabilities, capability)
	}
	return modelCapabilityMetadataKnown(model)
}

func (s *Service) modelInfoForRoute(ctx context.Context, route ResolvedModelRoute) (domain.ModelInfo, bool) {
	staticModel, hasStatic := findModelInfo(route.Definition.Models, route.Model.ModelID)
	providerID := firstNonEmpty(route.Provider.ID, route.Definition.ID, route.Model.ProviderID)
	if s != nil && s.store != nil {
		if cache, err := s.store.LoadProviderModelCache(ctx, providerID); err == nil && cache != nil {
			if cachedModel, ok := findModelInfo(cache.Models, route.Model.ModelID); ok {
				if isChatGPTCodexRoute(route) {
					return cachedModel, true
				}
				if hasStatic {
					return mergeModelInfo(staticModel, cachedModel), true
				}
				return cachedModel, true
			}
		}
	}
	if hasStatic {
		return staticModel, true
	}
	return domain.ModelInfo{}, false
}

func mergeModelInfo(fallback, discovered domain.ModelInfo) domain.ModelInfo {
	merged := fallback
	merged.ID = firstNonEmpty(discovered.ID, fallback.ID)
	merged.ProviderID = firstNonEmpty(discovered.ProviderID, fallback.ProviderID)
	merged.Name = firstNonEmpty(discovered.Name, fallback.Name)
	merged.Recommended = discovered.Recommended || fallback.Recommended
	merged.Deprecated = discovered.Deprecated || fallback.Deprecated
	if discovered.ContextLength > 0 {
		merged.ContextLength = discovered.ContextLength
	}
	if discovered.MaxContextLength > 0 {
		merged.MaxContextLength = discovered.MaxContextLength
	}
	if discovered.AutoCompactTokenLimit > 0 {
		merged.AutoCompactTokenLimit = discovered.AutoCompactTokenLimit
	}
	if discovered.OutputLimit > 0 {
		merged.OutputLimit = discovered.OutputLimit
	}
	merged.Capabilities = appendUniqueStrings(append([]string(nil), fallback.Capabilities...), discovered.Capabilities...)
	merged.DeclaredCapabilities = appendUniqueStrings(append([]string(nil), fallback.DeclaredCapabilities...), discovered.DeclaredCapabilities...)
	merged.Modalities = appendUniqueStrings(append([]string(nil), fallback.Modalities...), discovered.Modalities...)
	merged.Streaming = discovered.Streaming || fallback.Streaming
	merged.ToolSupport = discovered.ToolSupport || fallback.ToolSupport
	if len(discovered.ReasoningControls) > 0 {
		merged.ReasoningControls = append([]string(nil), discovered.ReasoningControls...)
	}
	if len(discovered.SupportedReasoningEfforts) > 0 {
		merged.SupportedReasoningEfforts = append([]string(nil), discovered.SupportedReasoningEfforts...)
	}
	merged.DefaultReasoningEffort = firstNonEmpty(discovered.DefaultReasoningEffort, fallback.DefaultReasoningEffort)
	if discovered.SupportsVerbosity != nil {
		value := *discovered.SupportsVerbosity
		merged.SupportsVerbosity = &value
	}
	merged.DefaultVerbosity = firstNonEmpty(discovered.DefaultVerbosity, fallback.DefaultVerbosity)
	if len(discovered.ServiceTiers) > 0 {
		merged.ServiceTiers = append([]string(nil), discovered.ServiceTiers...)
	}
	merged.DefaultServiceTier = firstNonEmpty(discovered.DefaultServiceTier, fallback.DefaultServiceTier)
	if discovered.SupportsParallelToolCalls != nil {
		value := *discovered.SupportsParallelToolCalls
		merged.SupportsParallelToolCalls = &value
	}
	if discovered.WebSearchToolTypeKnown {
		merged.WebSearchToolTypeKnown = true
		merged.WebSearchToolType = discovered.WebSearchToolType
	}
	if discovered.UseResponsesLite != nil {
		value := *discovered.UseResponsesLite
		merged.UseResponsesLite = &value
	}
	if discovered.SupportsImageDetailOriginal != nil {
		value := *discovered.SupportsImageDetailOriginal
		merged.SupportsImageDetailOriginal = &value
	}
	if discovered.NativeToolsKnown {
		merged.NativeToolsKnown = true
		merged.NativeTools = append([]string(nil), discovered.NativeTools...)
	}
	for _, capability := range discovered.DeclaredCapabilities {
		supported := modelSupportsCapability(discovered, capability)
		merged.Capabilities = withoutString(merged.Capabilities, capability)
		if supported {
			merged.Capabilities = appendUniqueStrings(merged.Capabilities, capability)
		}
		switch capability {
		case "tools":
			merged.ToolSupport = supported
		case "streaming":
			merged.Streaming = supported
		case "reasoning":
			if supported {
				merged.ReasoningControls = append([]string(nil), discovered.ReasoningControls...)
			} else {
				merged.ReasoningControls = nil
			}
		case "vision":
			if !supported {
				merged.Modalities = withoutString(merged.Modalities, "image")
			}
		}
	}
	if len(discovered.Pricing) > 0 {
		merged.Pricing = cloneFloatMap(discovered.Pricing)
	}
	merged.Status = firstNonEmpty(discovered.Status, fallback.Status)
	merged.LastRefreshed = firstNonEmpty(discovered.LastRefreshed, fallback.LastRefreshed)
	return merged
}

func withoutString(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
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
