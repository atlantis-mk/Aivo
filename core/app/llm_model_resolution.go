package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

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

func (s *Service) providerConfigForModelRequest(ctx context.Context, cfg domain.AppConfig, providerID string, modelID string) domain.ProviderConfig {
	providerID = s.normalizeProviderID(providerID)
	if cfg.Provider != nil && s.normalizeProviderID(cfg.Provider.ID) == providerID {
		provider := cloneProviderConfig(*cfg.Provider)
		if saved, ok := s.savedProviderConfig(ctx, providerID); ok {
			provider = mergeMissingProviderConfig(provider, saved)
		}
		provider.ID = providerID
		provider.Model = modelID
		return s.providerConfigWithDefaults(provider)
	}
	if provider, ok := s.savedProviderConfig(ctx, providerID); ok {
		provider.ID = providerID
		provider.Model = modelID
		return s.providerConfigWithDefaults(provider)
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
		ID:      providerID,
		Type:    string(inferTransport(providerID, "", "")),
		BaseURL: defaultBaseURLFor(providerID),
		Model:   modelID,
	}
}

func (s *Service) savedProviderConfig(ctx context.Context, providerID string) (domain.ProviderConfig, bool) {
	providerID = s.normalizeProviderID(providerID)
	providers, err := s.store.ListProviders(ctx)
	if err != nil {
		return domain.ProviderConfig{}, false
	}
	for _, provider := range providers {
		if s.normalizeProviderID(provider.ID) != providerID {
			continue
		}
		provider = cloneProviderConfig(provider)
		provider.ID = providerID
		return provider, true
	}
	return domain.ProviderConfig{}, false
}

func (s *Service) providerConfigWithDefaults(provider domain.ProviderConfig) domain.ProviderConfig {
	if def, ok := s.providerDefinition(provider.ID); ok {
		if provider.Type == "" {
			provider.Type = string(def.Transport)
		}
		if provider.BaseURL == "" {
			provider.BaseURL = def.DefaultBaseURL
		}
		if provider.APIKeyEnv == "" && len(def.APIKeyEnvVars) > 0 {
			provider.APIKeyEnv = def.APIKeyEnvVars[0]
		}
	} else if provider.Type == "" {
		provider.Type = string(inferTransport(provider.ID, "", provider.BaseURL))
	}
	return provider
}

func mergeMissingProviderConfig(primary domain.ProviderConfig, fallback domain.ProviderConfig) domain.ProviderConfig {
	if primary.ID == "" {
		primary.ID = fallback.ID
	}
	if primary.Type == "" {
		primary.Type = fallback.Type
	}
	if primary.BaseURL == "" {
		primary.BaseURL = fallback.BaseURL
	}
	if primary.APIKeyEnv == "" {
		primary.APIKeyEnv = fallback.APIKeyEnv
	}
	if primary.Model == "" {
		primary.Model = fallback.Model
	}
	if len(primary.Headers) == 0 {
		primary.Headers = cloneStringMap(fallback.Headers)
	}
	if len(primary.RequestParams) == 0 {
		primary.RequestParams = cloneAnyMap(fallback.RequestParams)
	}
	return primary
}

func cloneProviderConfig(provider domain.ProviderConfig) domain.ProviderConfig {
	provider.Headers = cloneStringMap(provider.Headers)
	provider.RequestParams = cloneAnyMap(provider.RequestParams)
	return provider
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
