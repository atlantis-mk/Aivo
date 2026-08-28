package app

import (
	"context"
	"errors"

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
	chatMessages, err := normalizeChatMessages(chatRequest.Messages)
	if err != nil {
		return domain.ChatResponse{}, nil, err
	}
	var lastErr error
	for fallbackIndex, route := range routes {
		route = applyChatRequestGenerationSettings(route, chatRequest)
		s.ensureDynamicProviderCapabilities(ctx, route)
		routeChatRequest := chatRequest
		routeChatRequest.Tools = s.toolsForModelRoute(ctx, cfg, route, chatRequest.Tools)
		if isChatGPTCodexRoute(route) {
			if modelInfo, ok := s.modelInfoForRoute(ctx, route); ok {
				if err := validateCodexInputModalities(modelInfo, chatMessages); err != nil {
					lastErr = err
					continue
				}
			}
		}
		if err := validateProviderToolIdentities(routeChatRequest.Tools, chatMessages); err != nil {
			return domain.ChatResponse{}, nil, err
		}
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
			var response domain.ChatResponse
			var err error
			switch route.Transport {
			case TransportOpenAIResponses:
				if isOAuthCredential(route.Credential) {
					modelInfo, _ := s.modelInfoForRoute(ctx, route)
					response, err = s.callChatGPTCodex(ctx, route.Provider, route.Model, modelInfo, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
					break
				}
				response, err = callOpenAICompatible(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			case TransportAzureOpenAI:
				response, err = callAzureOpenAI(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			case TransportAnthropicMessages:
				response, err = callAnthropic(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGoogleGemini:
				response, err = callGoogle(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGoogleVertex:
				response, err = callGoogleVertex(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportBedrockConverse:
				response, err = callBedrockConverse(ctx, route.Provider, route.Model, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportGitHubCopilot:
				response, err = callGitHubCopilot(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, routeOnDelta, onToolDelta)
			case TransportExternalProcess:
				response, err = callExternalProcessProvider(ctx, route.Definition, route.Model, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta)
			case TransportOpenAIChat, TransportOpenAICompatible:
				response, err = callOpenAICompatible(ctx, route.Provider, route.Model, route.Credential, route.Definition.RequestProfile, chatMessages, routeChatRequest.Tools, reasoningEffort, serviceTier, routeOnDelta, onToolDelta)
			default:
				return domain.ChatResponse{}, errors.New("unsupported provider transport: " + string(route.Transport))
			}
			return response, err
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

func validateCodexInputModalities(model domain.ModelInfo, messages []llmChatMessage) error {
	if len(model.Modalities) == 0 || containsString(model.Modalities, "image") {
		return nil
	}
	for _, message := range messages {
		for _, attachment := range message.Attachments {
			if isImageAttachmentMIME(attachment.MIMEType) {
				return errors.New("model capability unsupported: selected Codex model does not accept image input")
			}
		}
	}
	return nil
}

func applyChatRequestGenerationSettings(route ResolvedModelRoute, request domain.ChatRequest) ResolvedModelRoute {
	if request.Temperature == nil && request.TopP == nil && len(request.Options) == 0 {
		return route
	}
	route.Provider.RequestParams = cloneAnyMap(route.Provider.RequestParams)
	if route.Provider.RequestParams == nil {
		route.Provider.RequestParams = map[string]any{}
	}
	mergeRequestParams(route.Provider.RequestParams, request.Options, true)
	var temperature *float64
	if request.Temperature != nil {
		value := min(2, max(0, *request.Temperature))
		temperature = &value
	}
	var topP *float64
	if request.TopP != nil {
		value := min(1, max(0, *request.TopP))
		topP = &value
	}
	switch route.Transport {
	case TransportGoogleGemini, TransportGoogleVertex:
		generation, _ := route.Provider.RequestParams["generationConfig"].(map[string]any)
		generation = cloneAnyMap(generation)
		if generation == nil {
			generation = map[string]any{}
		}
		if temperature != nil {
			generation["temperature"] = *temperature
		}
		if topP != nil {
			generation["topP"] = *topP
		}
		route.Provider.RequestParams["generationConfig"] = generation
	default:
		if temperature != nil {
			route.Provider.RequestParams["temperature"] = *temperature
		}
		if topP != nil {
			route.Provider.RequestParams["top_p"] = *topP
		}
	}
	return route
}

type bedrockSigningConfig struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Region       string
}
