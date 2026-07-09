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
