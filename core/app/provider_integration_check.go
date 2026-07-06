package app

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"aivo/core/domain"
)

func (s *Service) CheckProviderIntegration(ctx context.Context, input domain.ProviderIntegrationCheckInput) (domain.ProviderIntegrationCheckResult, error) {
	connectInput := s.providerConnectInputForIntegrationCheck(ctx, input)
	provider, def, err := s.providerConfigFromInput(connectInput)
	if err != nil {
		return domain.ProviderIntegrationCheckResult{}, err
	}
	now := domain.NowString(s.now())
	appCfg := mustAppConfig(ctx, s)
	result := domain.ProviderIntegrationCheckResult{
		ProviderID: provider.ID,
		ModelID:    provider.Model,
		Status:     "checking",
		CheckedAt:  now,
		Transport:  string(inferTransport(provider.ID, provider.Type, provider.BaseURL)),
		BaseURL:    provider.BaseURL,
		Policy:     normalizeProviderRuntimePolicy(appCfg.ProviderPolicy),
	}
	result.Steps = append(result.Steps, okCheckStep("config", "provider config is valid"))

	credential, err := s.resolveCredentialWithDefinition(ctx, provider, def)
	if err != nil {
		result.Status = "failed"
		result.Steps = append(result.Steps, failedCheckStep("auth", "provider credentials are not ready", err))
		result.Recommended = append(result.Recommended, "Configure a supported credential method before selecting this provider.")
		return s.decorateProviderIntegrationCheck(ctx, result), nil
	}
	result.AuthMode = credential.Method
	result.Steps = append(result.Steps, okCheckStep("auth", "provider credentials resolved"))

	validationInput := domain.ProviderConnectInput{
		ProviderID: provider.ID,
		Type:       provider.Type,
		BaseURL:    provider.BaseURL,
		APIKeyEnv:  provider.APIKeyEnv,
		ModelID:    provider.Model,
		Method:     credential.Method,
		Headers:    provider.Headers,
	}
	validation, err := s.ValidateProvider(ctx, validationInput)
	if err != nil {
		result.Status = "failed"
		result.Steps = append(result.Steps, failedCheckStep("models", "model validation failed", err))
		return s.decorateProviderIntegrationCheck(ctx, result), nil
	}
	if !input.IncludeModelList {
		validation.Models = nil
	}
	result.Validation = &validation
	result.ModelCount = validation.ModelCount
	if validation.Ready {
		result.Steps = append(result.Steps, okCheckStep("models", "provider model list is available"))
	} else {
		result.Steps = append(result.Steps, domain.ProviderIntegrationCheckStep{ID: "models", Status: "warning", Message: "provider model list is not ready", Error: validation.Error})
		result.Recommended = append(result.Recommended, "Refresh models after credentials and network access are available.")
	}

	routeCfg := appCfg
	routeCfg.Provider = &provider
	routeCfg.DefaultModel = &domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model}
	route, err := s.ResolveModelRoute(ctx, routeCfg, &domain.ModelRef{ProviderID: provider.ID, ModelID: provider.Model})
	if err != nil {
		result.Status = "failed"
		result.Steps = append(result.Steps, failedCheckStep("runtime-route", "runtime route could not be resolved", err))
		result.Recommended = append(result.Recommended, "Check provider credentials, base URL, model id, and transport type.")
		return s.decorateProviderIntegrationCheck(ctx, result), nil
	}
	result.Steps = append(result.Steps, okCheckStep("runtime-route", "runtime route resolved"))
	if err := providerRuntimePreflight(route); err != nil {
		result.Status = "failed"
		result.Steps = append(result.Steps, failedCheckStep("runtime-preflight", "provider runtime prerequisites are not ready", err))
		result.Recommended = append(result.Recommended, providerRuntimeRecommendation(route.Transport))
		return s.decorateProviderIntegrationCheck(ctx, result), nil
	}
	result.Steps = append(result.Steps, okCheckStep("runtime-preflight", "provider runtime prerequisites are ready"))
	if model, ok := s.modelInfoForRoute(ctx, route); ok {
		result.Capabilities = append([]string(nil), model.Capabilities...)
		if model.ToolSupport && !containsString(result.Capabilities, "tools") {
			result.Capabilities = append(result.Capabilities, "tools")
		}
		if model.Streaming && !containsString(result.Capabilities, "streaming") {
			result.Capabilities = append(result.Capabilities, "streaming")
		}
		if len(model.ReasoningControls) > 0 && !containsString(result.Capabilities, "reasoning") {
			result.Capabilities = append(result.Capabilities, "reasoning")
		}
		if err := s.validateModelCapabilities(ctx, route, modelRequirement{Streaming: true}); err != nil {
			result.Steps = append(result.Steps, domain.ProviderIntegrationCheckStep{ID: "capabilities", Status: "warning", Message: "selected model has limited streaming metadata", Error: err.Error()})
		} else {
			result.Steps = append(result.Steps, okCheckStep("capabilities", "selected model capabilities are compatible with runtime checks"))
		}
	} else {
		result.Steps = append(result.Steps, domain.ProviderIntegrationCheckStep{ID: "capabilities", Status: "warning", Message: "selected model metadata is not available"})
		result.Recommended = append(result.Recommended, "Refresh model metadata to enable capability checks.")
	}

	result.Ready = true
	result.Status = "ready"
	return s.decorateProviderIntegrationCheck(ctx, result), nil
}

func providerRuntimePreflight(route ResolvedModelRoute) error {
	switch route.Transport {
	case TransportBedrockConverse:
		_, err := bedrockSigningConfigForHost(providerHost(route.Provider.BaseURL))
		return err
	case TransportGoogleVertex:
		if strings.Contains(route.Provider.BaseURL, "YOUR_PROJECT_ID") {
			return errors.New("Google Vertex base URL still contains YOUR_PROJECT_ID placeholder")
		}
	case TransportAzureOpenAI:
		if strings.Contains(route.Provider.BaseURL, "YOUR-RESOURCE-NAME") {
			return errors.New("Azure OpenAI base URL still contains YOUR-RESOURCE-NAME placeholder")
		}
	}
	return nil
}

func providerRuntimeRecommendation(transport TransportType) string {
	switch transport {
	case TransportBedrockConverse:
		return "Configure AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_REGION/AWS_DEFAULT_REGION or use a regional Bedrock runtime URL."
	case TransportGoogleVertex:
		return "Set Google Vertex base URL to a real project/location publisher endpoint and provide an access token."
	case TransportAzureOpenAI:
		return "Set Azure OpenAI base URL to the real resource endpoint and provide AZURE_OPENAI_API_KEY."
	default:
		return "Check provider credentials, base URL, model id, and transport-specific settings."
	}
}

func providerHost(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func (s *Service) providerConnectInputForIntegrationCheck(ctx context.Context, input domain.ProviderIntegrationCheckInput) domain.ProviderConnectInput {
	providerID := s.normalizeProviderID(input.ProviderID)
	out := domain.ProviderConnectInput{ProviderID: providerID, ModelID: input.ModelID}
	saved, _ := s.store.ListProviders(ctx)
	for _, provider := range saved {
		if s.normalizeProviderID(provider.ID) != providerID {
			continue
		}
		out.Type = provider.Type
		out.BaseURL = provider.BaseURL
		out.APIKeyEnv = provider.APIKeyEnv
		out.Headers = provider.Headers
		if out.ModelID == "" {
			out.ModelID = provider.Model
		}
		return out
	}
	return out
}

func (s *Service) decorateProviderIntegrationCheck(ctx context.Context, result domain.ProviderIntegrationCheckResult) domain.ProviderIntegrationCheckResult {
	if health, err := s.store.LoadProviderHealth(ctx, result.ProviderID); err == nil {
		result.Health = health
	}
	if usage, err := s.GetProviderUsage(ctx, domain.ProviderUsageInput{ProviderID: result.ProviderID, Limit: 200}); err == nil {
		result.Usage = &usage
	}
	if events, err := s.store.ListProviderCallEvents(ctx, result.ProviderID, 5); err == nil {
		result.RecentEvents = events
	}
	if result.Status == "" || strings.EqualFold(result.Status, "checking") {
		if result.Ready {
			result.Status = "ready"
		} else {
			result.Status = "failed"
		}
	}
	return result
}

func okCheckStep(id string, message string) domain.ProviderIntegrationCheckStep {
	return domain.ProviderIntegrationCheckStep{ID: id, Status: "ok", Message: message}
}

func failedCheckStep(id string, message string, err error) domain.ProviderIntegrationCheckStep {
	step := domain.ProviderIntegrationCheckStep{ID: id, Status: "failed", Message: message}
	if err != nil {
		step.Error = safeProviderError(err)
	}
	return step
}

func mustAppConfig(ctx context.Context, service *Service) domain.AppConfig {
	cfg, err := service.AppConfig(ctx)
	if err != nil {
		return domain.AppConfig{}
	}
	return cfg
}
