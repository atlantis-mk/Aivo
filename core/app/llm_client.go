package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"aivo/core/domain"
)

const chatGPTCodexResponsesURL = "https://chatgpt.com/backend-api/codex/responses"

type llmCredential struct {
	Method      string
	APIKey      string
	AccessToken string
	Refresh     string
	ExpiresAt   string
	AccountID   string
	AuthRecord  *domain.ProviderAuthRecord
}

type llmChatMessage struct {
	Role        string                     `json:"role"`
	Text        string                     `json:"text"`
	Attachments []domain.MessageAttachment `json:"attachments,omitempty"`
	ToolCalls   []domain.ChatToolCall      `json:"toolCalls,omitempty"`
	ToolCallID  string                     `json:"toolCallId,omitempty"`
	Name        string                     `json:"name,omitempty"`
}

func (s *Service) GenerateChatReply(ctx context.Context, messages []domain.ChatMessage, requestedModel *domain.ModelRef, reasoningEffort string, serviceTier string) (string, *domain.ModelRef, error) {
	return s.GenerateChatReplyStream(ctx, messages, requestedModel, reasoningEffort, serviceTier, nil)
}

func (s *Service) GenerateChatReplyStream(ctx context.Context, messages []domain.ChatMessage, requestedModel *domain.ModelRef, reasoningEffort string, serviceTier string, onDelta func(string)) (string, *domain.ModelRef, error) {
	resp, model, err := s.GenerateChatResponseStream(ctx, domain.ChatRequest{Messages: messages}, requestedModel, reasoningEffort, serviceTier, onDelta)
	if err != nil {
		return "", nil, err
	}
	return resp.Text, model, nil
}

func (s *Service) GenerateChatResponseStream(ctx context.Context, chatRequest domain.ChatRequest, requestedModel *domain.ModelRef, reasoningEffort string, serviceTier string, onDelta func(string)) (domain.ChatResponse, *domain.ModelRef, error) {
	return s.GenerateChatResponseStreamWithToolDelta(ctx, chatRequest, requestedModel, reasoningEffort, serviceTier, onDelta, nil)
}

func (s *Service) GenerateChatResponseStreamWithToolDelta(ctx context.Context, chatRequest domain.ChatRequest, requestedModel *domain.ModelRef, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, *domain.ModelRef, error) {
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return domain.ChatResponse{}, nil, err
	}
	reasoningEffort = normalizeReasoningEffort(reasoningEffort)
	serviceTier = normalizeServiceTier(serviceTier)
	routes, err := s.ResolveModelRoutes(ctx, cfg, requestedModel)
	if err != nil {
		return domain.ChatResponse{}, nil, err
	}
	if len(routes) == 0 {
		return domain.ChatResponse{}, nil, errors.New("provider is not configured")
	}
	policy := normalizeProviderRuntimePolicy(cfg.ProviderPolicy)
	if !providerPolicyBool(policy.EnableFallback) && len(routes) > 1 {
		routes = routes[:1]
	}
	chatMessages := normalizeChatMessages(chatRequest.Messages)
	var lastErr error
	for fallbackIndex, route := range routes {
		routeChatRequest := chatRequest
		routeChatRequest.Tools = s.toolsForModelRoute(ctx, cfg, route, chatRequest.Tools)
		requirements := requestRequirements(routeChatRequest, reasoningEffort, onDelta)
		if err := s.validateModelCapabilities(ctx, route, requirements); err != nil {
			lastErr = err
			if fallbackAllowed(err, false) {
				continue
			}
			return domain.ChatResponse{}, nil, err
		}
		shouldBuffer := providerPolicyBool(policy.BufferStreamingFallback) && requirements.Streaming && len(routes) > 1 && fallbackIndex < len(routes)-1
		routeOnDelta, flushDeltas, emittedOutput := bufferedDeltaForRoute(onDelta, shouldBuffer)
		response, err := s.callProviderWithRuntime(ctx, route, requirements, policy, fallbackIndex, func() (domain.ChatResponse, error) {
			switch route.Transport {
			case TransportOpenAIResponses:
				if isOAuthCredential(route.Credential) {
					return s.callChatGPTCodex(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
				}
				return callOpenAICompatible(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			case TransportAzureOpenAI:
				return callAzureOpenAI(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			case TransportAnthropicMessages:
				return callAnthropic(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGoogleGemini:
				return callGoogle(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGoogleVertex:
				return callGoogleVertex(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportBedrockConverse:
				return callBedrockConverse(ctx, route.Provider, route.Model, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGitHubCopilot:
				return callGitHubCopilot(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportOpenAIChat, TransportOpenAICompatible:
				return callOpenAICompatible(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			default:
				return domain.ChatResponse{}, errors.New("unsupported provider transport: " + string(route.Transport))
			}
		})
		if err == nil {
			flushDeltas()
			return response, &route.Model, nil
		}
		lastErr = err
		if !fallbackAllowed(err, emittedOutput()) {
			return domain.ChatResponse{}, nil, err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("provider request failed")
	}
	return domain.ChatResponse{}, nil, lastErr
}

func (s *Service) toolsForModelRoute(ctx context.Context, cfg domain.AppConfig, route ResolvedModelRoute, specs []domain.ToolSpec) []domain.ToolSpec {
	webSearch := normalizeWebSearchRuntimeConfig(cfg.WebSearch)
	nativeTools := normalizeNativeToolsRuntimeConfig(cfg.NativeTools)
	hostedSearch := webSearch.Mode != domain.WebSearchModeDisabled && routeSupportsHostedWebSearch(ctx, s, route, specs)
	hostedFetch := routeSupportsHostedWebFetch(ctx, s, route, specs)
	out := make([]domain.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		switch spec.Name {
		case "web_search":
			if webSearch.Mode == domain.WebSearchModeDisabled {
				continue
			}
			switch webSearch.Route {
			case domain.WebSearchRouteProvider:
				if hostedSearch {
					out = append(out, hostedWebSearchToolSpec(webSearch, route))
				}
			case domain.WebSearchRouteLocal:
				out = append(out, localToolSpec(spec))
			default:
				if hostedSearch {
					out = append(out, hostedWebSearchToolSpec(webSearch, route))
				} else {
					out = append(out, localToolSpec(spec))
				}
			}
		case "web_fetch":
			if hostedFetch {
				out = append(out, hostedWebFetchToolSpec(route))
			} else {
				out = append(out, localToolSpec(spec))
			}
		default:
			if spec.Hosted != nil && route.Transport != TransportOpenAIResponses {
				continue
			}
			out = append(out, spec)
		}
	}
	out = appendNativeHostedToolSpecs(ctx, s, route, nativeTools, out)
	return out
}

func routeSupportsHostedWebSearch(ctx context.Context, service *Service, route ResolvedModelRoute, specs []domain.ToolSpec) bool {
	if service == nil {
		return false
	}
	model, ok := service.modelInfoForRoute(ctx, route)
	if !ok {
		return false
	}
	providerID := normalizedRouteProviderID(route)
	switch route.Transport {
	case TransportOpenAIResponses, TransportAzureOpenAI:
		return modelSupportsCapability(model, "web_search") && (providerID == "openai" || providerID == "azure-openai")
	case TransportOpenAICompatible:
		if providerID == "xai" {
			return modelSupportsCapability(model, "web_search")
		}
		if providerID == "perplexity" {
			return modelSupportsCapability(model, "web_search") || modelSupportsCapability(model, "search")
		}
	case TransportAnthropicMessages:
		return modelSupportsCapability(model, "web_search") && (providerID == "anthropic" || providerID == "claude-code")
	case TransportGoogleGemini, TransportGoogleVertex:
		if !modelSupportsCapability(model, "web_search") {
			return false
		}
		return googleSearchCanCombineWithTools(route.Model.ModelID) || onlyWebSearchTool(specs)
	}
	return false
}

func routeSupportsHostedWebFetch(ctx context.Context, service *Service, route ResolvedModelRoute, specs []domain.ToolSpec) bool {
	if service == nil {
		return false
	}
	model, ok := service.modelInfoForRoute(ctx, route)
	if !ok || !modelSupportsCapability(model, "web_fetch") {
		return false
	}
	providerID := normalizedRouteProviderID(route)
	switch route.Transport {
	case TransportAnthropicMessages:
		return providerID == "anthropic" || providerID == "claude-code"
	case TransportGoogleGemini, TransportGoogleVertex:
		return googleSearchCanCombineWithTools(route.Model.ModelID) || onlyWebFetchTool(specs)
	default:
		return false
	}
}

func appendNativeHostedToolSpecs(ctx context.Context, service *Service, route ResolvedModelRoute, config domain.NativeToolsConfig, specs []domain.ToolSpec) []domain.ToolSpec {
	if service == nil {
		return specs
	}
	model, ok := service.modelInfoForRoute(ctx, route)
	if !ok {
		return specs
	}
	if config.XSearch.Enabled && routeSupportsNativeTool(route, model, "x_search") && !toolSpecNamed(specs, "x_search") {
		specs = append(specs, domain.ToolSpec{Name: "x_search", Description: "Search X using the active model provider's hosted X Search capability.", Hosted: &domain.HostedToolSpec{Type: "x_search"}, Capability: "web.x_search", Category: "web", RequiresNetwork: true})
	}
	if config.CodeExecution.Enabled {
		codeCapability := nativeCodeExecutionCapability(route)
		if routeSupportsNativeTool(route, model, codeCapability) && !toolSpecNamed(specs, codeCapability) {
			specs = append(specs, domain.ToolSpec{Name: codeCapability, Description: "Run code in the active model provider's hosted sandbox.", Hosted: &domain.HostedToolSpec{Type: codeCapability, ContainerID: config.CodeExecution.ContainerID, FileIDs: append([]string(nil), config.CodeExecution.FileIDs...)}, Capability: "hosted.code", Category: "hosted", RequiresNetwork: true})
		}
	}
	if config.FileSearch.Enabled && len(config.FileSearch.VectorStoreIDs) > 0 && routeSupportsNativeTool(route, model, "file_search") && !toolSpecNamed(specs, "file_search") {
		specs = append(specs, domain.ToolSpec{Name: "file_search", Description: "Search configured provider vector stores using the active model provider's hosted file search.", Hosted: &domain.HostedToolSpec{Type: "file_search", VectorStoreIDs: append([]string(nil), config.FileSearch.VectorStoreIDs...)}, Capability: "hosted.file_search", Category: "hosted", RequiresNetwork: true})
	}
	if len(config.RemoteMCP) > 0 && routeSupportsNativeTool(route, model, "remote_mcp") {
		for _, server := range config.RemoteMCP {
			if !server.Enabled || server.ServerURL == "" || server.ServerLabel == "" {
				continue
			}
			name := "remote_mcp_" + sanitizeHostedToolName(server.ServerLabel)
			if toolSpecNamed(specs, name) {
				continue
			}
			specs = append(specs, domain.ToolSpec{Name: name, Description: "Expose a configured remote MCP server to the active model provider.", Hosted: &domain.HostedToolSpec{Type: "mcp", ServerURL: server.ServerURL, ServerLabel: server.ServerLabel, AllowedTools: append([]string(nil), server.AllowedTools...)}, Capability: "hosted.mcp", Category: "hosted", RequiresNetwork: true})
		}
	}
	return specs
}

func nativeCodeExecutionCapability(route ResolvedModelRoute) string {
	switch route.Transport {
	case TransportGoogleGemini, TransportGoogleVertex:
		return "code_execution"
	default:
		return "code_interpreter"
	}
}

func routeSupportsNativeTool(route ResolvedModelRoute, model domain.ModelInfo, capability string) bool {
	if !modelSupportsCapability(model, capability) {
		return false
	}
	providerID := normalizedRouteProviderID(route)
	switch capability {
	case "x_search":
		return providerID == "xai" && route.Transport == TransportOpenAICompatible
	case "code_interpreter":
		return providerID == "openai" || providerID == "azure-openai" || providerID == "xai"
	case "code_execution":
		return (providerID == "gemini" || providerID == "google" || providerID == "google-vertex") && (route.Transport == TransportGoogleGemini || route.Transport == TransportGoogleVertex)
	case "file_search":
		if providerID == "openai" || providerID == "azure-openai" || providerID == "xai" {
			return true
		}
		return (providerID == "gemini" || providerID == "google" || providerID == "google-vertex") && (route.Transport == TransportGoogleGemini || route.Transport == TransportGoogleVertex)
	case "remote_mcp":
		return providerID == "openai" || providerID == "xai"
	default:
		return false
	}
}

func toolSpecNamed(specs []domain.ToolSpec, name string) bool {
	for _, spec := range specs {
		if spec.Name == name {
			return true
		}
	}
	return false
}

func sanitizeHostedToolName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('_')
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "server"
	}
	return out
}

func hostedWebSearchToolSpec(config domain.WebSearchConfig, route ResolvedModelRoute) domain.ToolSpec {
	externalWebAccess := true
	hostedType := "web_search"
	providerID := normalizedRouteProviderID(route)
	allowedDomains := append([]string(nil), config.AllowedDomains...)
	switch providerID {
	case "anthropic", "claude-code":
		hostedType = "web_search_20250305"
	case "gemini", "google", "google-vertex":
		hostedType = "google_search"
	case "perplexity":
		hostedType = "perplexity_search"
	case "xai":
		if len(allowedDomains) > 5 {
			allowedDomains = allowedDomains[:5]
		}
	}
	return domain.ToolSpec{
		Name:        "web_search",
		Description: "Search the public web using the active model provider's hosted web search capability.",
		Hosted: &domain.HostedToolSpec{
			Type:              hostedType,
			ExternalWebAccess: &externalWebAccess,
			SearchContextSize: config.SearchContextSize,
			AllowedDomains:    allowedDomains,
			UserLocation:      cloneWebSearchUserLocation(config.UserLocation),
		},
		Capability:      "web.search",
		Category:        "web",
		RequiresNetwork: true,
	}
}

func hostedWebFetchToolSpec(route ResolvedModelRoute) domain.ToolSpec {
	hostedType := "web_fetch"
	providerID := normalizedRouteProviderID(route)
	switch providerID {
	case "anthropic", "claude-code":
		hostedType = "web_fetch_20250910"
	case "gemini", "google", "google-vertex":
		hostedType = "url_context"
	}
	return domain.ToolSpec{
		Name:        "web_fetch",
		Description: "Fetch and inspect URL content using the active model provider's hosted URL context capability.",
		Hosted:      &domain.HostedToolSpec{Type: hostedType},
		Capability:  "web.fetch",
		Category:    "web",
	}
}

func normalizedRouteProviderID(route ResolvedModelRoute) string {
	if id := strings.TrimSpace(route.Provider.ID); id != "" {
		return normalizeProviderID(id)
	}
	return normalizeProviderID(route.Definition.ID)
}

func onlyWebSearchTool(specs []domain.ToolSpec) bool {
	for _, spec := range specs {
		if spec.Name != "web_search" {
			return false
		}
	}
	return len(specs) > 0
}

func onlyWebFetchTool(specs []domain.ToolSpec) bool {
	for _, spec := range specs {
		if spec.Name != "web_fetch" {
			return false
		}
	}
	return len(specs) > 0
}

func googleSearchCanCombineWithTools(modelID string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(modelID)), "gemini-3")
}

func localToolSpec(spec domain.ToolSpec) domain.ToolSpec {
	spec.Hosted = nil
	return spec
}

func cloneWebSearchUserLocation(location *domain.WebSearchUserLocation) *domain.WebSearchUserLocation {
	if location == nil {
		return nil
	}
	clone := *location
	return &clone
}

func normalizeWebSearchRuntimeConfig(config domain.WebSearchConfig) domain.WebSearchConfig {
	if strings.TrimSpace(config.Mode) == "" {
		config.Mode = domain.WebSearchModeLive
	}
	if config.Mode != domain.WebSearchModeDisabled && config.Mode != domain.WebSearchModeLive {
		config.Mode = domain.WebSearchModeLive
	}
	if strings.TrimSpace(config.Route) == "" {
		config.Route = domain.WebSearchRouteAuto
	}
	if config.Route != domain.WebSearchRouteAuto && config.Route != domain.WebSearchRouteLocal && config.Route != domain.WebSearchRouteProvider {
		config.Route = domain.WebSearchRouteAuto
	}
	switch strings.TrimSpace(config.SearchContextSize) {
	case "", "low", "medium", "high":
	default:
		config.SearchContextSize = ""
	}
	config.AllowedDomains = normalizeWebSearchDomains(config.AllowedDomains)
	if config.UserLocation != nil && strings.TrimSpace(config.UserLocation.Type) == "" {
		config.UserLocation.Type = "approximate"
	}
	return config
}

func normalizeWebSearchDomains(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.TrimPrefix(value, "http://")
		value = strings.TrimPrefix(value, "https://")
		value = strings.Trim(value, "/")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeNativeToolsRuntimeConfig(config domain.NativeToolsConfig) domain.NativeToolsConfig {
	config.CodeExecution.ContainerID = strings.TrimSpace(config.CodeExecution.ContainerID)
	config.CodeExecution.FileIDs = normalizeStringList(config.CodeExecution.FileIDs)
	config.FileSearch.VectorStoreIDs = normalizeStringList(config.FileSearch.VectorStoreIDs)
	if len(config.RemoteMCP) > 0 {
		out := make([]domain.NativeMCPToolConfig, 0, len(config.RemoteMCP))
		seen := map[string]bool{}
		for _, server := range config.RemoteMCP {
			server.ServerURL = strings.TrimSpace(server.ServerURL)
			server.ServerLabel = strings.TrimSpace(server.ServerLabel)
			server.AllowedTools = normalizeStringList(server.AllowedTools)
			if !server.Enabled || server.ServerURL == "" || server.ServerLabel == "" {
				continue
			}
			key := server.ServerURL + "\x00" + server.ServerLabel
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, server)
		}
		config.RemoteMCP = out
	}
	return config
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func bufferedDeltaForRoute(onDelta func(string), shouldBuffer bool) (func(string), func(), func() bool) {
	if onDelta == nil {
		return nil, func() {}, func() bool { return false }
	}
	if !shouldBuffer {
		emitted := false
		return func(delta string) {
			emitted = true
			onDelta(delta)
		}, func() {}, func() bool { return emitted }
	}
	var buffered []string
	return func(delta string) {
			buffered = append(buffered, delta)
		}, func() {
			for _, delta := range buffered {
				onDelta(delta)
			}
		}, func() bool {
			return false
		}
}

func providerConfigForModelRequest(cfg domain.AppConfig, providerID string, modelID string) domain.ProviderConfig {
	if cfg.Provider != nil && cfg.Provider.ID == providerID {
		provider := *cfg.Provider
		if provider.Type == "" {
			provider.Type = provider.ID
		}
		provider.Model = modelID
		return provider
	}
	for _, provider := range defaultProviders() {
		if provider.ID != providerID {
			continue
		}
		return domain.ProviderConfig{
			ID:        provider.ID,
			Type:      provider.Type,
			BaseURL:   provider.BaseURL,
			APIKeyEnv: provider.Environment,
			Model:     modelID,
		}
	}
	return domain.ProviderConfig{
		ID:      providerID,
		Type:    providerID,
		BaseURL: defaultBaseURLFor(providerID),
		Model:   modelID,
	}
}

func (s *Service) providerConfigForModelRequest(cfg domain.AppConfig, providerID string, modelID string) domain.ProviderConfig {
	if cfg.Provider != nil && s.normalizeProviderID(cfg.Provider.ID) == s.normalizeProviderID(providerID) {
		provider := *cfg.Provider
		provider.ID = s.normalizeProviderID(provider.ID)
		if provider.Type == "" {
			if def, ok := s.providerDefinition(provider.ID); ok {
				provider.Type = string(def.Transport)
			} else {
				provider.Type = string(inferTransport(provider.ID, "", provider.BaseURL))
			}
		}
		provider.Model = modelID
		return provider
	}
	if def, ok := s.providerDefinition(providerID); ok {
		apiKeyEnv := ""
		if len(def.APIKeyEnvVars) > 0 {
			apiKeyEnv = def.APIKeyEnvVars[0]
		}
		return domain.ProviderConfig{
			ID:        def.ID,
			Type:      string(def.Transport),
			BaseURL:   def.DefaultBaseURL,
			APIKeyEnv: apiKeyEnv,
			Model:     modelID,
		}
	}
	return domain.ProviderConfig{
		ID:      s.normalizeProviderID(providerID),
		Type:    string(inferTransport(providerID, "", "")),
		BaseURL: defaultBaseURLFor(providerID),
		Model:   modelID,
	}
}

func normalizeChatGPTCodexModel(model domain.ModelRef) domain.ModelRef {
	model.ModelID = normalizeModelIDForProvider(model.ProviderID, model.ModelID)
	return model
}

func normalizeModelIDForProvider(providerID string, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if providerID == "openai" && modelID == "gpt-5-codex" {
		return "gpt-5.5"
	}
	return modelID
}

func resolveActiveProvider(cfg domain.AppConfig) (domain.ProviderConfig, domain.ModelRef) {
	if cfg.Provider != nil {
		provider := *cfg.Provider
		if provider.Type == "" {
			provider.Type = provider.ID
		}
		if provider.Model == "" && cfg.DefaultModel != nil && cfg.DefaultModel.ProviderID == provider.ID {
			provider.Model = cfg.DefaultModel.ModelID
		}
		if provider.Model == "" {
			provider.Model = defaultModelFor(provider.ID)
		}
		return provider, domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
	}
	if cfg.DefaultModel != nil {
		return domain.ProviderConfig{
			ID:      cfg.DefaultModel.ProviderID,
			Type:    cfg.DefaultModel.ProviderID,
			BaseURL: defaultBaseURLFor(cfg.DefaultModel.ProviderID),
			Model:   cfg.DefaultModel.ModelID,
		}, *cfg.DefaultModel
	}
	return domain.ProviderConfig{}, domain.ModelRef{}
}

func (s *Service) resolveCredential(ctx context.Context, provider domain.ProviderConfig) (llmCredential, error) {
	return s.resolveCredentialWithDefinition(ctx, provider, providerDefinitionForConfig(provider))
}

func normalizeChatMessages(messages []domain.ChatMessage) []llmChatMessage {
	out := make([]llmChatMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role != "user" && role != "assistant" && role != "system" && role != "tool" {
			continue
		}
		text := strings.TrimSpace(message.Text)
		if text == "" && len(message.Attachments) == 0 && len(message.ToolCalls) == 0 {
			continue
		}
		out = append(out, llmChatMessage{Role: role, Text: text, Attachments: sanitizeLLMAttachments(message.Attachments), ToolCalls: message.ToolCalls, ToolCallID: message.ToolCallID, Name: message.Name})
	}
	return out
}

func sanitizeLLMAttachments(attachments []domain.MessageAttachment) []domain.MessageAttachment {
	out := make([]domain.MessageAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		data := strings.TrimSpace(attachment.Data)
		text := strings.TrimSpace(attachment.Text)
		if data == "" && text == "" {
			continue
		}
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = "attachment"
		}
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		kind := strings.TrimSpace(attachment.Kind)
		if kind == "" {
			if strings.HasPrefix(mimeType, "image/") {
				kind = "image"
			} else {
				kind = "file"
			}
		}
		out = append(out, domain.MessageAttachment{
			ID: attachment.ID, Name: name, MIMEType: mimeType, Kind: kind,
			Data: data, Text: text, Size: attachment.Size,
		})
	}
	return out
}

func (s *Service) callChatGPTCodex(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	access, accountID, err := s.validOpenAIAccessToken(ctx, credential)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	body := responsesRequestBody(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTCodexResponsesURL, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("User-Agent", openAIUserAgent)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func (s *Service) validOpenAIAccessToken(ctx context.Context, credential llmCredential) (string, string, error) {
	if credential.AccessToken != "" && !isExpired(credential.ExpiresAt, s.now()) {
		return credential.AccessToken, credential.AccountID, nil
	}
	if credential.Refresh == "" {
		return "", "", errors.New("OpenAI OAuth refresh token is missing")
	}
	tokens, err := refreshOpenAIToken(ctx, credential.Refresh)
	if err != nil {
		return "", "", err
	}
	accountID := extractOpenAIAccountID(tokens)
	if accountID == "" {
		accountID = credential.AccountID
	}
	expiresIn := tokens.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	if credential.AuthRecord != nil {
		next := *credential.AuthRecord
		next.AccessToken = tokens.AccessToken
		next.RefreshToken = firstNonEmpty(tokens.RefreshToken, credential.Refresh)
		next.ExpiresAt = domain.NowString(s.now().Add(time.Duration(expiresIn) * time.Second))
		next.AccountID = accountID
		next.UpdatedAt = domain.NowString(s.now())
		if err := s.saveProviderAuth(ctx, next); err != nil {
			return "", "", err
		}
	}
	return tokens.AccessToken, accountID, nil
}

func refreshOpenAIToken(ctx context.Context, refreshToken string) (openAITokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", openAIClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIIssuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return openAITokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doOpenAITokenRequest(req)
}

func isExpired(raw string, now time.Time) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return true
	}
	return !expiresAt.After(now.Add(30 * time.Second))
}

func callOpenAICompatible(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	var endpoint string
	var body map[string]any
	if providerUsesResponsesAPI(provider, tools) {
		endpoint = baseURL + "/responses"
		body = responsesRequestBody(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	} else {
		endpoint = baseURL + "/chat/completions"
		body = chatCompletionsRequestBody(model.ModelID, messages, tools)
	}
	applyProviderNativeWebSearchOptions(body, provider, tools)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	if credential.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+credential.APIKey)
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func providerUsesResponsesAPI(provider domain.ProviderConfig, tools []domain.ToolSpec) bool {
	providerID := normalizeProviderID(firstNonEmpty(provider.ID, provider.Type))
	if providerID == "openai" {
		return true
	}
	return providerID == "xai" && hasResponsesHostedTool(tools)
}

func hasResponsesHostedTool(tools []domain.ToolSpec) bool {
	for _, tool := range tools {
		if tool.Hosted == nil {
			continue
		}
		switch tool.Hosted.Type {
		case "web_search", "x_search", "code_interpreter", "file_search", "mcp":
			return true
		}
	}
	return false
}

func applyProviderNativeWebSearchOptions(body map[string]any, provider domain.ProviderConfig, tools []domain.ToolSpec) {
	if normalizeProviderID(provider.ID) != "perplexity" {
		return
	}
	for _, tool := range tools {
		if tool.Hosted == nil || tool.Hosted.Type != "perplexity_search" {
			continue
		}
		if len(tool.Hosted.AllowedDomains) > 0 {
			body["search_domain_filter"] = append([]string(nil), tool.Hosted.AllowedDomains...)
		}
		if size := strings.TrimSpace(tool.Hosted.SearchContextSize); size != "" {
			body["web_search_options"] = map[string]any{"search_context_size": size}
		}
		return
	}
}

func callAzureOpenAI(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := responsesRequestBody(model.ModelID, messages, tools, reasoningEffort, serviceTier)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("api-key", credential.APIKey)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callAnthropic(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	body := anthropicRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/messages", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	req.Header.Set("x-api-key", credential.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGoogle(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	if credential.APIKey == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, url.PathEscape(model.ModelID), url.QueryEscape(credential.APIKey))
	body := googleRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGoogleVertex(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	token := firstNonEmpty(credential.AccessToken, credential.APIKey)
	if token == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", baseURL, url.PathEscape(model.ModelID))
	body := googleRequestBody(model.ModelID, messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callBedrockConverse(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := bedrockConverseRequestBody(messages, tools, reasoningEffort)
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	endpoint := fmt.Sprintf("%s/model/%s/converse", baseURL, url.PathEscape(model.ModelID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	if err := signBedrockRequest(req, raw); err != nil {
		return domain.ChatResponse{}, err
	}
	return doLLMRequest(req, onDelta, onToolDelta)
}

func callGitHubCopilot(ctx context.Context, provider domain.ProviderConfig, model domain.ModelRef, credential llmCredential, requestProfile domain.ProviderRequestProfile, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	token := firstNonEmpty(credential.AccessToken, credential.APIKey)
	if token == "" {
		return domain.ChatResponse{}, fmt.Errorf("credentials are not configured for provider %q", provider.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURLFor(provider.ID)
	}
	if baseURL == "" {
		return domain.ChatResponse{}, fmt.Errorf("base URL is not configured for provider %q", provider.ID)
	}
	body := chatCompletionsRequestBody(model.ModelID, messages, tools)
	if effort := chatCompletionsReasoningEffort(reasoningEffort); effort != "" {
		body["reasoning_effort"] = effort
	}
	applyRequestProfile(body, requestProfile, provider, model.ModelID)
	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	applyRequestProfileHeaders(req, githubCopilotRequestProfile(), provider, model.ModelID)
	applyRequestProfileHeaders(req, requestProfile, provider, model.ModelID)
	return doLLMRequest(req, onDelta, onToolDelta)
}

func applyProviderHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		req.Header.Set(key, value)
	}
}

func signBedrockRequest(req *http.Request, payload []byte) error {
	cfg, err := bedrockSigningConfigForHost(req.URL.Hostname())
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(payload)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if cfg.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", cfg.SessionToken)
	}
	signedHeaders := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	canonicalHeaders := "host:" + req.URL.Host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	if token := req.Header.Get("X-Amz-Security-Token"); token != "" {
		signedHeaders = []string{"host", "x-amz-content-sha256", "x-amz-date", "x-amz-security-token"}
		canonicalHeaders += "x-amz-security-token:" + strings.TrimSpace(token) + "\n"
	}
	signedHeadersValue := strings.Join(signedHeaders, ";")
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeadersValue,
		payloadHash,
	}, "\n")
	scope := strings.Join([]string{dateStamp, cfg.Region, "bedrock", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := fmt.Sprintf("%x", hmacSHA256(awsSigningKey(cfg.SecretKey, dateStamp, cfg.Region, "bedrock"), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", cfg.AccessKey, scope, signedHeadersValue, signature))
	return nil
}

type bedrockSigningConfig struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Region       string
}

func bedrockSigningConfigForHost(host string) (bedrockSigningConfig, error) {
	cfg := bedrockSigningConfig{
		AccessKey:    lookupEnv("AWS_ACCESS_KEY_ID"),
		SecretKey:    lookupEnv("AWS_SECRET_ACCESS_KEY"),
		SessionToken: lookupEnv("AWS_SESSION_TOKEN"),
		Region:       firstNonEmpty(lookupEnv("AWS_REGION"), lookupEnv("AWS_DEFAULT_REGION"), bedrockRegionFromHost(host)),
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return bedrockSigningConfig{}, errors.New("AWS credentials are not configured: set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	}
	if cfg.Region == "" {
		return bedrockSigningConfig{}, errors.New("AWS region is not configured: set AWS_REGION or use a regional Bedrock runtime URL")
	}
	return cfg, nil
}

func bedrockRegionFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if !strings.HasPrefix(host, "bedrock-runtime.") || !strings.HasSuffix(host, ".amazonaws.com") {
		return ""
	}
	region := strings.TrimSuffix(strings.TrimPrefix(host, "bedrock-runtime."), ".amazonaws.com")
	if strings.Contains(region, ".") {
		return ""
	}
	return region
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func awsSigningKey(secret string, date string, region string, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key []byte, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func responsesRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string, serviceTier string) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		if message.Role == "tool" {
			outputType := "function_call_output"
			if strings.HasPrefix(strings.TrimSpace(message.Text), "{") && strings.Contains(message.Text, `"name":"apply_patch"`) {
				outputType = "custom_tool_call_output"
			}
			input = append(input, map[string]any{
				"type":    outputType,
				"call_id": message.ToolCallID,
				"output":  message.Text,
			})
			continue
		}
		if len(message.ToolCalls) > 0 {
			if strings.TrimSpace(message.Text) != "" {
				input = append(input, responsesMessageItem(message.Role, message.Text, nil))
			}
			for _, call := range message.ToolCalls {
				item := map[string]any{
					"type":      "function_call",
					"call_id":   call.ID,
					"name":      call.Name,
					"arguments": string(call.Arguments),
				}
				if strings.HasPrefix(strings.TrimSpace(string(call.Arguments)), "*** Begin Patch") {
					item = map[string]any{
						"type":    "custom_tool_call",
						"call_id": call.ID,
						"name":    call.Name,
						"input":   string(call.Arguments),
					}
				}
				input = append(input, item)
			}
			continue
		}
		if message.Role == "system" {
			input = append(input, map[string]any{
				"role":    "system",
				"content": message.Text,
			})
			continue
		}
		input = append(input, responsesMessageItem(message.Role, message.Text, message.Attachments))
	}
	body := map[string]any{
		"model":               model,
		"input":               input,
		"tool_choice":         "auto",
		"parallel_tool_calls": len(tools) > 0,
		"stream":              true,
		"store":               false,
	}
	if len(tools) > 0 {
		body["tools"] = responsesTools(tools)
	}
	if effort := responsesReasoningEffort(reasoningEffort); effort != "" {
		body["reasoning"] = map[string]string{"effort": effort}
	}
	if tier := responsesServiceTier(serviceTier); tier != "" {
		body["service_tier"] = tier
	}
	return body
}

func responsesMessageItem(role string, text string, attachments []domain.MessageAttachment) map[string]any {
	contentType := "input_text"
	if role == "assistant" {
		contentType = "output_text"
	}
	content := []map[string]string{}
	if strings.TrimSpace(text) != "" {
		content = append(content, map[string]string{"type": contentType, "text": text})
	}
	if role == "user" {
		for _, attachment := range attachments {
			if part := responsesAttachmentPart(attachment); len(part) > 0 {
				content = append(content, part)
			}
		}
	}
	return map[string]any{
		"role":    role,
		"content": content,
	}
}

func responsesAttachmentPart(attachment domain.MessageAttachment) map[string]string {
	data := strings.TrimSpace(attachment.Data)
	if data == "" {
		text := strings.TrimSpace(attachment.Text)
		if text == "" {
			return nil
		}
		return map[string]string{"type": "input_text", "text": attachment.Name + "\n" + text}
	}
	mimeType := strings.TrimSpace(attachment.MIMEType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	dataURL := dataURLForAttachment(mimeType, data)
	if strings.HasPrefix(mimeType, "image/") || attachment.Kind == "image" {
		return map[string]string{"type": "input_image", "image_url": dataURL}
	}
	return map[string]string{"type": "input_file", "filename": attachment.Name, "file_data": dataURL}
}

func dataURLForAttachment(mimeType string, data string) string {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "data:") {
		return data
	}
	return "data:" + mimeType + ";base64," + data
}

func responsesServiceTier(serviceTier string) string {
	serviceTier = normalizeServiceTier(serviceTier)
	if serviceTier == "default" {
		return ""
	}
	return serviceTier
}

func responsesReasoningEffort(effort string) string {
	switch normalizeReasoningEffort(effort) {
	case "low":
		return "low"
	case "high":
		return "high"
	case "ultra":
		return "high"
	case "medium":
		return "medium"
	default:
		return ""
	}
}

func chatCompletionsRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{
			"role":    message.Role,
			"content": message.Text,
		}
		if message.Role == "user" && len(message.Attachments) > 0 {
			item["content"] = chatCompletionContentParts(message.Text, message.Attachments)
		}
		if message.Role == "tool" {
			item["tool_call_id"] = message.ToolCallID
			item["name"] = message.Name
		}
		if len(message.ToolCalls) > 0 {
			item["tool_calls"] = chatCompletionToolCalls(message.ToolCalls)
		}
		input = append(input, item)
	}
	body := map[string]any{
		"model":    model,
		"messages": input,
		"stream":   true,
	}
	if len(tools) > 0 {
		if serializedTools := chatCompletionTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
			body["tool_choice"] = "auto"
		}
	}
	return body
}

func chatCompletionContentParts(text string, attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	if strings.TrimSpace(text) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": text})
	}
	for _, attachment := range attachments {
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := strings.TrimSpace(attachment.Text); text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": attachment.Name + "\n" + text})
			}
			continue
		}
		if strings.HasPrefix(mimeType, "image/") || attachment.Kind == "image" {
			parts = append(parts, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": dataURLForAttachment(mimeType, data),
				},
			})
		}
	}
	return parts
}

func chatCompletionsReasoningEffort(effort string) string {
	switch normalizeReasoningEffort(effort) {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "ultra":
		return "high"
	default:
		return ""
	}
}

func anthropicRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	input := make([]map[string]any, 0, len(messages))
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			system = append(system, message.Text)
			continue
		}
		if message.Role == "tool" {
			input = append(input, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": message.ToolCallID,
					"content":     message.Text,
				}},
			})
			continue
		}
		role := message.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		item := map[string]any{"role": role, "content": message.Text}
		if role == "user" && len(message.Attachments) > 0 {
			item["content"] = anthropicContentParts(message.Text, message.Attachments)
		}
		if len(message.ToolCalls) > 0 {
			content := make([]map[string]any, 0, len(message.ToolCalls)+1)
			if strings.TrimSpace(message.Text) != "" {
				content = append(content, map[string]any{"type": "text", "text": message.Text})
			}
			content = append(content, anthropicAttachmentParts(message.Attachments)...)
			for _, call := range message.ToolCalls {
				var input any = map[string]any{}
				_ = json.Unmarshal(call.Arguments, &input)
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    call.ID,
					"name":  call.Name,
					"input": input,
				})
			}
			item["content"] = content
		}
		input = append(input, item)
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": anthropicDefaultMaxTokens(model),
		"messages":   input,
		"stream":     true,
	}
	if len(tools) > 0 {
		if serializedTools := anthropicTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
		}
	}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n")
	}
	applyAnthropicReasoning(body, model, reasoningEffort)
	return body
}

func anthropicContentParts(text string, attachments []domain.MessageAttachment) []map[string]any {
	content := []map[string]any{}
	if strings.TrimSpace(text) != "" {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	content = append(content, anthropicAttachmentParts(attachments)...)
	if len(content) == 0 {
		content = append(content, map[string]any{"type": "text", "text": ""})
	}
	return content
}

func anthropicAttachmentParts(attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	for _, attachment := range attachments {
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := strings.TrimSpace(attachment.Text); text != "" {
				parts = append(parts, map[string]any{"type": "text", "text": attachment.Name + "\n" + text})
			}
			continue
		}
		switch {
		case strings.HasPrefix(mimeType, "image/") || attachment.Kind == "image":
			parts = append(parts, map[string]any{
				"type": "image",
				"source": map[string]string{
					"type":       "base64",
					"media_type": mimeType,
					"data":       data,
				},
			})
		case mimeType == "application/pdf":
			parts = append(parts, map[string]any{
				"type":  "document",
				"title": attachment.Name,
				"source": map[string]string{
					"type":       "base64",
					"media_type": mimeType,
					"data":       data,
				},
			})
		}
	}
	return parts
}

func applyAnthropicReasoning(body map[string]any, model string, reasoningEffort string) {
	effort := normalizeReasoningEffort(reasoningEffort)
	model = strings.ToLower(strings.TrimSpace(model))
	if effort == "" || effort == "medium" || model == "" {
		return
	}
	if usesAnthropicAdaptiveThinking(model) {
		body["thinking"] = map[string]any{"type": "adaptive"}
		body["output_config"] = map[string]any{"effort": anthropicOutputEffort(effort)}
		return
	}
	if !supportsAnthropicBudgetThinking(model) {
		return
	}
	budget := anthropicThinkingBudget(effort)
	if budget <= 0 {
		return
	}
	maxTokens := budget + 4096
	if existing, ok := body["max_tokens"].(int); ok && existing > maxTokens {
		maxTokens = existing
	}
	body["max_tokens"] = maxTokens
	body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
}

func anthropicDefaultMaxTokens(model string) int {
	model = strings.ToLower(strings.TrimSpace(model))
	limits := []struct {
		match string
		limit int
	}{
		{"claude-3-7-sonnet", 128000},
		{"claude-3-5-sonnet", 8192},
		{"claude-3-5-haiku", 8192},
		{"claude-3-opus", 4096},
		{"claude-3-sonnet", 4096},
		{"claude-3-haiku", 4096},
		{"claude-opus-4-8", 128000},
		{"claude-opus-4-7", 128000},
		{"claude-opus-4-6", 128000},
		{"claude-sonnet-4-6", 64000},
		{"claude-opus-4-5", 64000},
		{"claude-sonnet-4-5", 64000},
		{"claude-haiku-4-5", 64000},
		{"claude-sonnet-4", 64000},
		{"claude-opus-4", 32000},
		{"claude-fable", 128000},
		{"minimax", 131072},
		{"qwen3", 65536},
	}
	for _, item := range limits {
		if strings.Contains(model, item.match) {
			return item.limit
		}
	}
	return 128000
}

func anthropicOutputEffort(effort string) string {
	switch effort {
	case "low":
		return "low"
	case "high":
		return "high"
	case "ultra":
		return "xhigh"
	default:
		return "medium"
	}
}

func usesAnthropicAdaptiveThinking(model string) bool {
	return strings.Contains(model, "4-6") ||
		strings.Contains(model, "4.6") ||
		strings.Contains(model, "4-7") ||
		strings.Contains(model, "4.7") ||
		strings.Contains(model, "4-8") ||
		strings.Contains(model, "4.8") ||
		strings.Contains(model, "claude-fable-5") ||
		strings.Contains(model, "claude-mythos-5")
}

func supportsAnthropicBudgetThinking(model string) bool {
	return strings.Contains(model, "claude-3-7") ||
		strings.Contains(model, "claude-4") ||
		strings.Contains(model, "sonnet-4") ||
		strings.Contains(model, "opus-4")
}

func anthropicThinkingBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "high":
		return 4096
	case "ultra":
		return 8192
	default:
		return 0
	}
}

func googleRequestBody(model string, messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	contents := make([]map[string]any, 0, len(messages))
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			system = append(system, message.Text)
			continue
		}
		if message.Role == "tool" {
			var response any = map[string]any{"content": message.Text}
			contents = append(contents, map[string]any{
				"role": "function",
				"parts": []map[string]any{{
					"functionResponse": map[string]any{
						"name":     message.Name,
						"response": response,
					},
				}},
			})
			continue
		}
		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}
		parts := []map[string]any{}
		if strings.TrimSpace(message.Text) != "" {
			parts = append(parts, map[string]any{"text": message.Text})
		}
		if role == "user" {
			parts = append(parts, googleAttachmentParts(message.Attachments)...)
		}
		for _, call := range message.ToolCalls {
			var args any = map[string]any{}
			_ = json.Unmarshal(call.Arguments, &args)
			parts = append(parts, map[string]any{
				"functionCall": map[string]any{
					"name": call.Name,
					"args": args,
				},
			})
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	body := map[string]any{"contents": contents}
	if len(tools) > 0 {
		if serializedTools := googleTools(tools); len(serializedTools) > 0 {
			body["tools"] = serializedTools
		}
	}
	if len(system) > 0 {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": strings.Join(system, "\n")}},
		}
	}
	if thinkingConfig := googleThinkingConfig(model, reasoningEffort); len(thinkingConfig) > 0 {
		body["generationConfig"] = map[string]any{"thinkingConfig": thinkingConfig}
	}
	return body
}

func googleAttachmentParts(attachments []domain.MessageAttachment) []map[string]any {
	parts := []map[string]any{}
	for _, attachment := range attachments {
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		data := strings.TrimSpace(attachment.Data)
		if data == "" {
			if text := strings.TrimSpace(attachment.Text); text != "" {
				parts = append(parts, map[string]any{"text": attachment.Name + "\n" + text})
			}
			continue
		}
		parts = append(parts, map[string]any{
			"inlineData": map[string]string{
				"mimeType": mimeType,
				"data":     data,
			},
		})
	}
	return parts
}

func googleThinkingConfig(model string, reasoningEffort string) map[string]any {
	effort := normalizeReasoningEffort(reasoningEffort)
	model = strings.ToLower(strings.TrimSpace(model))
	if effort == "" || effort == "medium" || model == "" {
		return nil
	}
	if strings.HasPrefix(model, "gemini-3") {
		return map[string]any{"thinkingLevel": googleThinkingLevel(effort)}
	}
	if strings.HasPrefix(model, "gemini-2.5") {
		return map[string]any{"thinkingBudget": googleThinkingBudget(effort)}
	}
	return nil
}

func googleThinkingLevel(effort string) string {
	switch effort {
	case "low":
		return "low"
	case "high", "ultra":
		return "high"
	default:
		return "medium"
	}
}

func googleThinkingBudget(effort string) int {
	switch effort {
	case "low":
		return 1024
	case "high":
		return 8192
	case "ultra":
		return 24576
	default:
		return -1
	}
}

func bedrockConverseRequestBody(messages []llmChatMessage, tools []domain.ToolSpec, reasoningEffort string) map[string]any {
	conversation := make([]map[string]any, 0, len(messages))
	var system []map[string]string
	for _, message := range messages {
		if message.Role == "system" {
			if text := strings.TrimSpace(message.Text); text != "" {
				system = append(system, map[string]string{"text": text})
			}
			continue
		}
		if message.Role == "tool" {
			content := []map[string]any{{
				"toolResult": map[string]any{
					"toolUseId": message.ToolCallID,
					"content":   bedrockToolResultContent(message.Text),
				},
			}}
			conversation = append(conversation, map[string]any{"role": "user", "content": content})
			continue
		}
		role := "user"
		if message.Role == "assistant" {
			role = "assistant"
		}
		content := make([]map[string]any, 0, len(message.ToolCalls)+1)
		if text := strings.TrimSpace(message.Text); text != "" {
			content = append(content, map[string]any{"text": text})
		}
		for _, call := range message.ToolCalls {
			var input any = map[string]any{}
			_ = json.Unmarshal(call.Arguments, &input)
			content = append(content, map[string]any{
				"toolUse": map[string]any{
					"toolUseId": call.ID,
					"name":      call.Name,
					"input":     input,
				},
			})
		}
		if len(content) == 0 {
			continue
		}
		conversation = append(conversation, map[string]any{"role": role, "content": content})
	}
	body := map[string]any{
		"messages":        conversation,
		"inferenceConfig": map[string]any{"maxTokens": 4096},
	}
	if len(system) > 0 {
		body["system"] = system
	}
	if len(tools) > 0 {
		body["toolConfig"] = map[string]any{
			"tools":      bedrockTools(tools),
			"toolChoice": map[string]any{"auto": map[string]any{}},
		}
	}
	if budget := bedrockReasoningBudget(reasoningEffort); budget > 0 {
		body["additionalModelRequestFields"] = map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": budget},
		}
	}
	return body
}

func bedrockToolResultContent(text string) []map[string]any {
	text = strings.TrimSpace(text)
	if text == "" {
		return []map[string]any{{"text": ""}}
	}
	var parsed any
	if json.Unmarshal([]byte(text), &parsed) == nil {
		return []map[string]any{{"json": parsed}}
	}
	return []map[string]any{{"text": text}}
}

func bedrockReasoningBudget(reasoningEffort string) int {
	switch normalizeReasoningEffort(reasoningEffort) {
	case "low":
		return 1024
	case "high":
		return 4096
	case "ultra":
		return 8192
	default:
		return 0
	}
}

func doLLMRequest(req *http.Request, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return domain.ChatResponse{}, providerHTTPError(resp.StatusCode, resp.Status, string(raw))
	}
	if shouldReadEventStream(req, resp) {
		response, err := readLLMEventStream(resp.Body, onDelta, onToolDelta)
		if err != nil {
			return domain.ChatResponse{}, err
		}
		if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
			return domain.ChatResponse{}, providerResponseError("provider response did not include text")
		}
		return response, nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	response := extractChatResponse(raw)
	if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 {
		return domain.ChatResponse{}, providerResponseError("provider response did not include text")
	}
	if onDelta != nil && strings.TrimSpace(response.Text) != "" && len(response.ToolCalls) == 0 {
		onDelta(response.Text)
	}
	return response, nil
}

func shouldReadEventStream(req *http.Request, resp *http.Response) bool {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return true
	}
	return strings.Contains(strings.ToLower(req.Header.Get("Accept")), "text/event-stream")
}

type streamedToolCall struct {
	ID        string
	Name      string
	Arguments string
}

func readLLMEventStream(reader io.Reader, onDelta func(string), onToolDelta func(domain.ChatToolCall)) (domain.ChatResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var deltas []string
	var completed []string
	var rawLines []string
	var toolCalls []domain.ChatToolCall
	var usage *domain.TokenUsage
	responseTools := map[string]*streamedToolCall{}
	chatTools := map[int]*streamedToolCall{}
	anthropicTools := map[int]*streamedToolCall{}
	for scanner.Scan() {
		rawLine := scanner.Text()
		rawLines = append(rawLines, rawLine)
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if nextUsage := extractTokenUsage(event); nextUsage != nil {
			usage = mergeTokenUsage(usage, nextUsage)
		}
		toolCalls = appendUniqueToolCalls(toolCalls, updateResponsesStreamToolCalls(responseTools, event, onToolDelta)...)
		toolCalls = appendUniqueToolCalls(toolCalls, updateChatCompletionsStreamToolCalls(chatTools, event, onToolDelta)...)
		toolCalls = appendUniqueToolCalls(toolCalls, updateAnthropicStreamToolCalls(anthropicTools, event, onToolDelta)...)
		toolCalls = appendUniqueToolCalls(toolCalls, extractGoogleStreamToolCalls(event)...)
		if delta := extractResponseDeltaText(event); delta != "" {
			deltas = append(deltas, delta)
			if onDelta != nil {
				onDelta(delta)
			}
			continue
		}
		if text := extractResponsePayloadText(event); strings.TrimSpace(text) != "" {
			completed = append(completed, text)
			continue
		}
		if item, _ := event["item"].(map[string]any); item != nil {
			if text := extractResponsePayloadText(item); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
				continue
			}
		}
		if response, _ := event["response"].(map[string]any); response != nil {
			toolCalls = appendUniqueToolCalls(toolCalls, extractResponseToolCalls(response)...)
			if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return domain.ChatResponse{}, err
	}
	toolCalls = appendUniqueToolCalls(toolCalls, finishChatCompletionsStreamToolCalls(chatTools)...)
	toolCalls = appendUniqueToolCalls(toolCalls, finishIndexedStreamToolCalls(anthropicTools)...)
	if len(deltas) > 0 {
		return domain.ChatResponse{Text: strings.Join(deltas, ""), ToolCalls: toolCalls, Usage: usage}, nil
	}
	if len(completed) > 0 {
		text := strings.Join(completed, "\n")
		return domain.ChatResponse{Text: text, ToolCalls: toolCalls, Usage: usage}, nil
	}
	if len(rawLines) > 0 {
		response := extractChatResponse([]byte(strings.Join(rawLines, "\n")))
		response.ToolCalls = appendUniqueToolCalls(toolCalls, response.ToolCalls...)
		response.Usage = mergeTokenUsage(response.Usage, usage)
		if strings.TrimSpace(response.Text) != "" {
			return response, nil
		}
		if len(response.ToolCalls) > 0 {
			return response, nil
		}
	}
	return domain.ChatResponse{ToolCalls: toolCalls, Usage: usage}, nil
}

func previewLogText(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\r", "\\r")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func updateResponsesStreamToolCalls(tools map[string]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	eventType, _ := event["type"].(string)
	if eventType == "response.output_item.added" {
		item, _ := event["item"].(map[string]any)
		itemType, _ := item["type"].(string)
		if itemType != "function_call" && itemType != "custom_tool_call" {
			return nil
		}
		key := firstString(item, "id")
		if key == "" {
			return nil
		}
		tools[key] = &streamedToolCall{
			ID:        firstString(item, "call_id", "id"),
			Name:      firstString(item, "name"),
			Arguments: argumentStringFromAny(firstNonNil(item["arguments"], item["input"])),
		}
		emitStreamedToolDelta(tools[key], onToolDelta)
		return nil
	}
	if eventType == "response.function_call_arguments.delta" {
		key := firstString(event, "item_id")
		if key == "" {
			return nil
		}
		tool := tools[key]
		if tool == nil {
			tool = &streamedToolCall{ID: key}
			tools[key] = tool
		}
		if delta, _ := event["delta"].(string); delta != "" {
			tool.Arguments += delta
		}
		emitStreamedToolDelta(tool, onToolDelta)
		return nil
	}
	if eventType == "response.custom_tool_call_input.delta" {
		key := firstString(event, "item_id")
		if key == "" {
			return nil
		}
		tool := tools[key]
		if tool == nil {
			tool = &streamedToolCall{ID: key}
			tools[key] = tool
		}
		if delta, _ := event["delta"].(string); delta != "" {
			tool.Arguments += delta
		}
		emitStreamedToolDelta(tool, onToolDelta)
		return nil
	}
	if eventType == "response.custom_tool_call_input.done" {
		key := firstString(event, "item_id")
		tool := tools[key]
		if tool != nil {
			if args, _ := event["input"].(string); args != "" {
				tool.Arguments = args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		return nil
	}
	if eventType == "response.function_call_arguments.done" {
		key := firstString(event, "item_id")
		tool := tools[key]
		if tool != nil {
			if args, _ := event["arguments"].(string); args != "" {
				tool.Arguments = args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		return nil
	}
	if eventType != "response.output_item.done" {
		return nil
	}
	item, _ := event["item"].(map[string]any)
	itemType, _ := item["type"].(string)
	if itemType != "function_call" && itemType != "custom_tool_call" {
		return nil
	}
	key := firstString(item, "id")
	tool := tools[key]
	if tool == nil {
		tool = &streamedToolCall{}
	}
	if id := firstString(item, "call_id", "id"); id != "" {
		tool.ID = id
	}
	if name := firstString(item, "name"); name != "" {
		tool.Name = name
	}
	if args, _ := item["arguments"].(string); args != "" {
		tool.Arguments = args
	}
	if args, _ := item["input"].(string); args != "" {
		tool.Arguments = args
	}
	delete(tools, key)
	return []domain.ChatToolCall{tool.toChatToolCall()}
}

func updateChatCompletionsStreamToolCalls(tools map[int]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	choices, _ := event["choices"].([]any)
	var finished []domain.ChatToolCall
	for _, choiceRaw := range choices {
		choice, _ := choiceRaw.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		rawCalls, _ := delta["tool_calls"].([]any)
		for _, rawCall := range rawCalls {
			call, _ := rawCall.(map[string]any)
			index, ok := numberAsInt(call["index"])
			if !ok {
				index = len(tools)
			}
			tool := tools[index]
			if tool == nil {
				tool = &streamedToolCall{}
				tools[index] = tool
			}
			if id, _ := call["id"].(string); id != "" {
				tool.ID = id
			}
			fn, _ := call["function"].(map[string]any)
			if name, _ := fn["name"].(string); name != "" {
				tool.Name = name
			}
			if args, _ := fn["arguments"].(string); args != "" {
				tool.Arguments += args
			}
			emitStreamedToolDelta(tool, onToolDelta)
		}
		if reason, _ := choice["finish_reason"].(string); reason == "tool_calls" || reason == "function_call" {
			finished = append(finished, finishChatCompletionsStreamToolCalls(tools)...)
		}
	}
	return finished
}

func updateAnthropicStreamToolCalls(tools map[int]*streamedToolCall, event map[string]any, onToolDelta func(domain.ChatToolCall)) []domain.ChatToolCall {
	eventType, _ := event["type"].(string)
	index, _ := numberAsInt(event["index"])
	switch eventType {
	case "content_block_start":
		block, _ := event["content_block"].(map[string]any)
		if blockType, _ := block["type"].(string); blockType != "tool_use" {
			return nil
		}
		tools[index] = &streamedToolCall{
			ID:        firstString(block, "id"),
			Name:      firstString(block, "name"),
			Arguments: initialAnthropicToolInput(block["input"]),
		}
		emitStreamedToolDelta(tools[index], onToolDelta)
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]any)
		if deltaType, _ := delta["type"].(string); deltaType != "input_json_delta" {
			return nil
		}
		tool := tools[index]
		if tool == nil {
			tool = &streamedToolCall{}
			tools[index] = tool
		}
		if partial, _ := delta["partial_json"].(string); partial != "" {
			tool.Arguments += partial
		}
		emitStreamedToolDelta(tool, onToolDelta)
	case "content_block_stop":
		tool := tools[index]
		if tool == nil {
			return nil
		}
		delete(tools, index)
		return []domain.ChatToolCall{tool.toChatToolCall()}
	}
	return nil
}

func extractGoogleStreamToolCalls(event map[string]any) []domain.ChatToolCall {
	if _, ok := event["candidates"]; !ok {
		return nil
	}
	return extractResponseToolCalls(event)
}

func initialAnthropicToolInput(value any) string {
	if object, ok := value.(map[string]any); ok && len(object) == 0 {
		return ""
	}
	return argumentStringFromAny(value)
}

func finishChatCompletionsStreamToolCalls(tools map[int]*streamedToolCall) []domain.ChatToolCall {
	return finishIndexedStreamToolCalls(tools)
}

func finishIndexedStreamToolCalls(tools map[int]*streamedToolCall) []domain.ChatToolCall {
	if len(tools) == 0 {
		return nil
	}
	keys := make([]int, 0, len(tools))
	for key := range tools {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	calls := make([]domain.ChatToolCall, 0, len(keys))
	for _, key := range keys {
		calls = append(calls, tools[key].toChatToolCall())
		delete(tools, key)
	}
	return calls
}

func emitStreamedToolDelta(call *streamedToolCall, onToolDelta func(domain.ChatToolCall)) {
	if onToolDelta == nil || call == nil {
		return
	}
	if strings.TrimSpace(call.Name) == "" && strings.TrimSpace(call.ID) == "" && strings.TrimSpace(call.Arguments) == "" {
		return
	}
	onToolDelta(call.toChatToolCall())
}

func (call streamedToolCall) toChatToolCall() domain.ChatToolCall {
	id := call.ID
	if id == "" {
		id = call.Name
	}
	args := strings.TrimSpace(call.Arguments)
	if args == "" {
		args = "{}"
	}
	return domain.ChatToolCall{ID: id, Name: call.Name, Arguments: json.RawMessage(args)}
}

func appendUniqueToolCalls(existing []domain.ChatToolCall, next ...domain.ChatToolCall) []domain.ChatToolCall {
	seen := make(map[string]bool, len(existing)+len(next))
	for _, call := range existing {
		seen[toolCallKey(call)] = true
	}
	for _, call := range next {
		if call.Name == "" && call.ID == "" {
			continue
		}
		if len(call.Arguments) == 0 {
			call.Arguments = json.RawMessage(`{}`)
		}
		key := toolCallKey(call)
		if seen[key] {
			continue
		}
		seen[key] = true
		existing = append(existing, call)
	}
	return existing
}

func toolCallKey(call domain.ChatToolCall) string {
	return firstNonEmpty(call.ID, call.Name) + "\x00" + call.Name
}

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func extractChatResponse(raw []byte) domain.ChatResponse {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.ChatResponse{Text: extractResponseStreamText(raw)}
	}
	return domain.ChatResponse{Text: extractResponsePayloadText(payload), ToolCalls: extractResponseToolCalls(payload), Usage: extractTokenUsage(payload)}
}

func extractResponseText(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return extractResponseStreamText(raw)
	}
	return extractResponsePayloadText(payload)
}

func extractResponsePayloadText(payload map[string]any) string {
	if text, _ := payload["output_text"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if text := textFromContentValue(payload["message"]); strings.TrimSpace(text) != "" {
		return text
	}
	if text := textFromContentValue(payload["content"]); strings.TrimSpace(text) != "" {
		return text
	}
	if text, _ := payload["text"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if text, _ := payload["response"].(string); strings.TrimSpace(text) != "" {
		return text
	}
	if response, _ := payload["response"].(map[string]any); response != nil {
		if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if choices, ok := payload["choices"].([]any); ok {
		var parts []string
		for _, choice := range choices {
			item, _ := choice.(map[string]any)
			if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
			message, _ := item["message"].(map[string]any)
			if text := textFromContentValue(message); strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	if output, ok := payload["output"].([]any); ok {
		var parts []string
		for _, item := range output {
			outputItem, _ := item.(map[string]any)
			content, _ := outputItem["content"].([]any)
			for _, contentItem := range content {
				part, _ := contentItem.(map[string]any)
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	if output, _ := payload["output"].(map[string]any); output != nil {
		if message, _ := output["message"].(map[string]any); message != nil {
			if text := textFromContentValue(message); strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	if candidates, ok := payload["candidates"].([]any); ok {
		var parts []string
		for _, candidate := range candidates {
			item, _ := candidate.(map[string]any)
			content, _ := item["content"].(map[string]any)
			partsRaw, _ := content["parts"].([]any)
			for _, partRaw := range partsRaw {
				part, _ := partRaw.(map[string]any)
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func textFromContentValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case map[string]any:
		if text, _ := typed["content"].(string); strings.TrimSpace(text) != "" {
			return text
		}
		if text, _ := typed["text"].(string); strings.TrimSpace(text) != "" {
			return text
		}
		if content, ok := typed["content"].([]any); ok {
			return textFromContentParts(content)
		}
	case []any:
		return textFromContentParts(typed)
	}
	return ""
}

func textFromContentParts(content []any) string {
	var parts []string
	for _, itemRaw := range content {
		item, _ := itemRaw.(map[string]any)
		if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractTokenUsage(payload map[string]any) *domain.TokenUsage {
	if payload == nil {
		return nil
	}
	if usage := tokenUsageFromMap(mapValue(payload, "usage")); usage != nil {
		return usage
	}
	if usage := tokenUsageFromMap(mapValue(payload, "usageMetadata", "usage_metadata")); usage != nil {
		return usage
	}
	if response := mapValue(payload, "response"); response != nil {
		if usage := extractTokenUsage(response); usage != nil {
			return usage
		}
	}
	return nil
}

func tokenUsageFromMap(usage map[string]any) *domain.TokenUsage {
	if usage == nil {
		return nil
	}
	input := firstUsageInt(usage, "input_tokens", "prompt_tokens", "promptTokenCount", "inputTokenCount", "inputTokens", "cache_read_input_tokens")
	output := firstUsageInt(usage, "output_tokens", "completion_tokens", "candidatesTokenCount", "outputTokenCount", "outputTokens")
	total := firstUsageInt(usage, "total_tokens", "totalTokenCount", "totalTokens")
	if total == 0 && (input > 0 || output > 0) {
		total = input + output
	}
	if input == 0 && output == 0 && total == 0 {
		return nil
	}
	return &domain.TokenUsage{InputTokens: input, OutputTokens: output, TotalTokens: total}
}

func mapValue(payload map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, _ := payload[key].(map[string]any); value != nil {
			return value
		}
	}
	return nil
}

func firstUsageInt(payload map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := numberAsInt(payload[key]); ok && value > 0 {
			return value
		}
	}
	return 0
}

func mergeTokenUsage(primary *domain.TokenUsage, next *domain.TokenUsage) *domain.TokenUsage {
	if primary == nil {
		return next
	}
	if next == nil {
		return primary
	}
	if next.InputTokens > 0 {
		primary.InputTokens = next.InputTokens
	}
	if next.OutputTokens > 0 {
		primary.OutputTokens = next.OutputTokens
	}
	if next.TotalTokens > 0 {
		primary.TotalTokens = next.TotalTokens
	}
	primary.Estimated = primary.Estimated && next.Estimated
	return primary
}

func extractResponseToolCalls(payload map[string]any) []domain.ChatToolCall {
	var calls []domain.ChatToolCall
	if choices, ok := payload["choices"].([]any); ok {
		for _, choiceRaw := range choices {
			choice, _ := choiceRaw.(map[string]any)
			message, _ := choice["message"].(map[string]any)
			calls = append(calls, extractOpenAIChatToolCalls(message)...)
		}
	}
	if output, ok := payload["output"].([]any); ok {
		for _, itemRaw := range output {
			item, _ := itemRaw.(map[string]any)
			itemType, _ := item["type"].(string)
			if itemType != "function_call" {
				continue
			}
			id := firstString(item, "call_id", "id")
			name, _ := item["name"].(string)
			args := rawJSONFromAny(firstNonNil(item["arguments"], item["input"]))
			calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: args})
		}
	}
	if content, ok := payload["content"].([]any); ok {
		for _, itemRaw := range content {
			item, _ := itemRaw.(map[string]any)
			itemType, _ := item["type"].(string)
			if itemType != "tool_use" {
				continue
			}
			id, _ := item["id"].(string)
			name, _ := item["name"].(string)
			calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: rawJSONFromAny(item["input"])})
		}
	}
	if output, _ := payload["output"].(map[string]any); output != nil {
		if message, _ := output["message"].(map[string]any); message != nil {
			if content, _ := message["content"].([]any); content != nil {
				for _, itemRaw := range content {
					item, _ := itemRaw.(map[string]any)
					toolUse, _ := item["toolUse"].(map[string]any)
					if toolUse == nil {
						continue
					}
					id, _ := toolUse["toolUseId"].(string)
					name, _ := toolUse["name"].(string)
					calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: rawJSONFromAny(toolUse["input"])})
				}
			}
		}
	}
	if candidates, ok := payload["candidates"].([]any); ok {
		for _, candidateRaw := range candidates {
			candidate, _ := candidateRaw.(map[string]any)
			content, _ := candidate["content"].(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, partRaw := range parts {
				part, _ := partRaw.(map[string]any)
				fc, _ := part["functionCall"].(map[string]any)
				if fc == nil {
					continue
				}
				name, _ := fc["name"].(string)
				calls = append(calls, domain.ChatToolCall{ID: name, Name: name, Arguments: rawJSONFromAny(fc["args"])})
			}
		}
	}
	for i := range calls {
		if calls[i].ID == "" {
			calls[i].ID = fmt.Sprintf("call_%d", i+1)
		}
		if len(calls[i].Arguments) == 0 {
			calls[i].Arguments = json.RawMessage(`{}`)
		}
	}
	return calls
}

func extractOpenAIChatToolCalls(message map[string]any) []domain.ChatToolCall {
	if message == nil {
		return nil
	}
	rawCalls, _ := message["tool_calls"].([]any)
	calls := make([]domain.ChatToolCall, 0, len(rawCalls))
	for _, rawCall := range rawCalls {
		call, _ := rawCall.(map[string]any)
		id, _ := call["id"].(string)
		fn, _ := call["function"].(map[string]any)
		if fn == nil {
			continue
		}
		name, _ := fn["name"].(string)
		args, _ := fn["arguments"].(string)
		calls = append(calls, domain.ChatToolCall{ID: id, Name: name, Arguments: json.RawMessage(firstNonEmpty(args, "{}"))})
	}
	return calls
}

func responsesTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	namespaceIndices := map[string]int{}
	for _, spec := range specs {
		if spec.Hosted != nil {
			if hosted := responsesHostedTool(spec.Hosted); hosted != nil {
				tools = append(tools, hosted)
			}
			continue
		}
		if spec.Kind == domain.ToolKindFreeform {
			tool := map[string]any{
				"type":        "custom",
				"name":        spec.Name,
				"description": spec.Description,
			}
			if spec.Format != nil {
				tool["format"] = spec.Format
			}
			tools = append(tools, tool)
			continue
		}
		tool := map[string]any{
			"type":        "function",
			"name":        spec.Name,
			"description": spec.Description,
			"parameters":  spec.InputSchema,
		}
		namespace := responsesToolNamespace(spec)
		if namespace == "" {
			tools = append(tools, tool)
			continue
		}
		if index, ok := namespaceIndices[namespace]; ok {
			namespaceTools, _ := tools[index]["tools"].([]map[string]any)
			tools[index]["tools"] = append(namespaceTools, tool)
			continue
		}
		description := strings.TrimSpace(spec.NamespaceDescription)
		if description == "" {
			description = fmt.Sprintf("Tools in the %s namespace.", namespace)
		}
		namespaceIndices[namespace] = len(tools)
		tools = append(tools, map[string]any{
			"type":        "namespace",
			"name":        namespace,
			"description": description,
			"tools":       []map[string]any{tool},
		})
	}
	return tools
}

func responsesToolNamespace(spec domain.ToolSpec) string {
	namespace := strings.TrimSpace(spec.Namespace)
	switch namespace {
	case "browser":
		return ""
	default:
		return namespace
	}
}

func responsesHostedTool(spec *domain.HostedToolSpec) map[string]any {
	if spec == nil {
		return nil
	}
	switch strings.TrimSpace(spec.Type) {
	case "web_search", "x_search":
		tool := map[string]any{"type": strings.TrimSpace(spec.Type)}
		if spec.ExternalWebAccess != nil {
			tool["external_web_access"] = *spec.ExternalWebAccess
		}
		if size := strings.TrimSpace(spec.SearchContextSize); size != "" {
			tool["search_context_size"] = size
		}
		if len(spec.AllowedDomains) > 0 {
			tool["filters"] = map[string]any{"allowed_domains": append([]string(nil), spec.AllowedDomains...)}
		}
		if spec.UserLocation != nil {
			location := hostedUserLocationMap(spec.UserLocation)
			if len(location) > 0 {
				tool["user_location"] = location
			}
		}
		return tool
	case "code_interpreter":
		tool := map[string]any{"type": "code_interpreter"}
		container := map[string]any{"type": "auto"}
		if id := strings.TrimSpace(spec.ContainerID); id != "" {
			container = map[string]any{"type": "container_id", "container_id": id}
		}
		if len(spec.FileIDs) > 0 {
			container["files"] = append([]string(nil), spec.FileIDs...)
		}
		tool["container"] = container
		return tool
	case "file_search":
		if len(spec.VectorStoreIDs) == 0 {
			return nil
		}
		return map[string]any{
			"type":             "file_search",
			"vector_store_ids": append([]string(nil), spec.VectorStoreIDs...),
		}
	case "mcp":
		serverURL := strings.TrimSpace(spec.ServerURL)
		serverLabel := strings.TrimSpace(spec.ServerLabel)
		if serverURL == "" || serverLabel == "" {
			return nil
		}
		tool := map[string]any{"type": "mcp", "server_url": serverURL, "server_label": serverLabel}
		if len(spec.AllowedTools) > 0 {
			tool["allowed_tools"] = append([]string(nil), spec.AllowedTools...)
		}
		return tool
	default:
		return nil
	}
}

func hostedUserLocationMap(location *domain.WebSearchUserLocation) map[string]any {
	if location == nil {
		return nil
	}
	out := map[string]any{}
	if value := strings.TrimSpace(location.Type); value != "" {
		out["type"] = value
	}
	if value := strings.TrimSpace(location.Country); value != "" {
		out["country"] = value
	}
	if value := strings.TrimSpace(location.Region); value != "" {
		out["region"] = value
	}
	if value := strings.TrimSpace(location.City); value != "" {
		out["city"] = value
	}
	if value := strings.TrimSpace(location.Timezone); value != "" {
		out["timezone"] = value
	}
	return out
}

func chatCompletionTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			continue
		}
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"parameters":  spec.InputSchema,
			},
		})
	}
	return tools
}

func anthropicTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			if tool := anthropicHostedTool(spec.Hosted); tool != nil {
				tools = append(tools, tool)
			}
			continue
		}
		tools = append(tools, map[string]any{
			"name":         spec.Name,
			"description":  spec.Description,
			"input_schema": spec.InputSchema,
		})
	}
	return tools
}

func anthropicHostedTool(spec *domain.HostedToolSpec) map[string]any {
	if spec == nil {
		return nil
	}
	if !strings.HasPrefix(spec.Type, "web_search_") && !strings.HasPrefix(spec.Type, "web_fetch_") {
		return nil
	}
	name := "web_search"
	if strings.HasPrefix(spec.Type, "web_fetch_") {
		name = "web_fetch"
	}
	tool := map[string]any{
		"type": spec.Type,
		"name": name,
	}
	if spec.MaxUses > 0 {
		tool["max_uses"] = spec.MaxUses
	}
	if len(spec.AllowedDomains) > 0 {
		tool["allowed_domains"] = append([]string(nil), spec.AllowedDomains...)
	}
	if spec.UserLocation != nil {
		location := hostedUserLocationMap(spec.UserLocation)
		if len(location) > 0 {
			tool["user_location"] = location
		}
	}
	return tool
}

func googleTools(specs []domain.ToolSpec) []map[string]any {
	declarations := make([]map[string]any, 0, len(specs))
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			switch spec.Hosted.Type {
			case "google_search":
				tools = append(tools, map[string]any{"google_search": map[string]any{}})
			case "url_context":
				tools = append(tools, map[string]any{"url_context": map[string]any{}})
			case "code_execution":
				tools = append(tools, map[string]any{"code_execution": map[string]any{}})
			case "file_search":
				if len(spec.Hosted.VectorStoreIDs) > 0 {
					tools = append(tools, map[string]any{"file_search": map[string]any{"file_search_store_names": append([]string(nil), spec.Hosted.VectorStoreIDs...)}})
				}
			}
			continue
		}
		declarations = append(declarations, map[string]any{
			"name":        spec.Name,
			"description": spec.Description,
			"parameters":  spec.InputSchema,
		})
	}
	if len(declarations) > 0 {
		tools = append(tools, map[string]any{"functionDeclarations": declarations})
	}
	return tools
}

func bedrockTools(specs []domain.ToolSpec) []map[string]any {
	tools := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if spec.Hosted != nil {
			continue
		}
		schema := spec.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		tools = append(tools, map[string]any{
			"toolSpec": map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"inputSchema": map[string]any{"json": schema},
			},
		})
	}
	return tools
}

func responsesToolCalls(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{
			"type":      "function_call",
			"call_id":   call.ID,
			"name":      call.Name,
			"arguments": string(call.Arguments),
		})
	}
	return out
}

func chatCompletionToolCalls(calls []domain.ChatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{
			"id":   call.ID,
			"type": "function",
			"function": map[string]any{
				"name":      call.Name,
				"arguments": string(call.Arguments),
			},
		})
	}
	return out
}

func rawJSONFromAny(value any) json.RawMessage {
	switch typed := value.(type) {
	case nil:
		return json.RawMessage(`{}`)
	case string:
		if strings.TrimSpace(typed) == "" {
			return json.RawMessage(`{}`)
		}
		return json.RawMessage(typed)
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return json.RawMessage(`{}`)
		}
		return raw
	}
}

func argumentStringFromAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func firstString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, _ := item[key].(string); value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func extractResponseStreamText(raw []byte) string {
	var deltas []string
	var completed []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if delta := extractResponseDeltaText(event); delta != "" {
			deltas = append(deltas, delta)
			continue
		}
		if text := extractResponsePayloadText(event); strings.TrimSpace(text) != "" {
			completed = append(completed, text)
			continue
		}
		if item, _ := event["item"].(map[string]any); item != nil {
			if text := extractResponsePayloadText(item); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
				continue
			}
		}
		if response, _ := event["response"].(map[string]any); response != nil {
			if text := extractResponsePayloadText(response); strings.TrimSpace(text) != "" {
				completed = append(completed, text)
			}
		}
	}
	if len(deltas) > 0 {
		return strings.Join(deltas, "")
	}
	if len(completed) > 0 {
		return strings.Join(completed, "\n")
	}
	return ""
}

func extractResponseDeltaText(event map[string]any) string {
	if eventType, _ := event["type"].(string); eventType == "response.output_text.delta" || eventType == "response.refusal.delta" {
		if delta, _ := event["delta"].(string); delta != "" {
			return delta
		}
		if text, _ := event["text"].(string); text != "" {
			return text
		}
	}
	if text := extractChatCompletionDeltaText(event); text != "" {
		return text
	}
	if text := extractAnthropicDeltaText(event); text != "" {
		return text
	}
	if _, ok := event["candidates"]; ok {
		return extractResponsePayloadText(event)
	}
	return ""
}

func extractChatCompletionDeltaText(event map[string]any) string {
	choices, _ := event["choices"].([]any)
	var parts []string
	for _, choiceRaw := range choices {
		choice, _ := choiceRaw.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if text, _ := delta["content"].(string); text != "" {
			parts = append(parts, text)
		}
		if content, ok := delta["content"].([]any); ok {
			for _, itemRaw := range content {
				item, _ := itemRaw.(map[string]any)
				if text, _ := item["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		if text, _ := delta["text"].(string); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}

func extractAnthropicDeltaText(event map[string]any) string {
	eventType, _ := event["type"].(string)
	switch eventType {
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]any)
		if deltaType, _ := delta["type"].(string); deltaType != "" && deltaType != "text_delta" {
			return ""
		}
		text, _ := delta["text"].(string)
		return text
	case "content_block_start":
		block, _ := event["content_block"].(map[string]any)
		if blockType, _ := block["type"].(string); blockType != "" && blockType != "text" {
			return ""
		}
		text, _ := block["text"].(string)
		return text
	default:
		return ""
	}
}
