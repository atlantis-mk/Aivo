package app

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"aivo/core/domain"
)

func parseCodexModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body codexModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	items := make([]codexModelInfo, 0, len(body.Models))
	for _, model := range body.Models {
		if strings.TrimSpace(model.Visibility) != "" && model.Visibility != "list" {
			continue
		}
		id := firstNonEmpty(model.Slug, model.ID)
		if strings.TrimSpace(id) == "" {
			continue
		}
		items = append(items, model)
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftID := firstNonEmpty(items[i].Slug, items[i].ID)
		rightID := firstNonEmpty(items[j].Slug, items[j].ID)
		if compareModelVersion(leftID, rightID) != 0 {
			return compareModelVersion(leftID, rightID) > 0
		}
		return items[i].Priority < items[j].Priority
	})
	models := make([]domain.ModelInfo, 0, len(items))
	for _, item := range items {
		id := firstNonEmpty(item.Slug, item.ID)
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, item.Name, id),
		})
	}
	return finalizeParsedModels(models, "")
}

func parseOpenAICompatibleModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body openAIModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, item.Name, id),
		})
	}
	sortModelsForProvider(providerID, models)
	return finalizeParsedModels(models, "")
}

func parseGoogleModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body googleModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Models))
	for _, item := range body.Models {
		id := strings.TrimPrefix(strings.TrimSpace(item.Name), "models/")
		if id == "" || !supportsGoogleGenerateContent(item.SupportedGenerationMethods) {
			continue
		}
		models = append(models, domain.ModelInfo{
			ID:         id,
			ProviderID: providerID,
			Name:       firstNonEmpty(item.DisplayName, id),
		})
	}
	return finalizeParsedModels(models, "")
}

func finalizeParsedModels(models []domain.ModelInfo, preferredDefault string) ([]domain.ModelInfo, string, error) {
	if len(models) == 0 {
		return nil, "", errors.New("provider model response did not include any models")
	}
	seen := map[string]bool{}
	out := make([]domain.ModelInfo, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		model.ID = id
		model.Name = firstNonEmpty(model.Name, id)
		out = append(out, model)
	}
	if len(out) == 0 {
		return nil, "", errors.New("provider model response did not include any models")
	}
	defaultModel := preferredDefault
	if defaultModel == "" {
		defaultModel = out[0].ID
	}
	markRecommended(out, defaultModel)
	return out, defaultModel, nil
}

func sortModelsForProvider(providerID string, models []domain.ModelInfo) {
	if providerID != "openai" {
		return
	}
	sort.SliceStable(models, func(i, j int) bool {
		if compareModelVersion(models[i].ID, models[j].ID) != 0 {
			return compareModelVersion(models[i].ID, models[j].ID) > 0
		}
		return models[i].ID < models[j].ID
	})
}

func compareModelVersion(left string, right string) int {
	leftParts := modelVersionParts(left)
	rightParts := modelVersionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func modelVersionParts(modelID string) []int {
	lower := strings.ToLower(modelID)
	index := strings.Index(lower, "gpt-")
	if index < 0 {
		return nil
	}
	rest := lower[index+len("gpt-"):]
	parts := []int{}
	current := 0
	hasDigit := false
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
			hasDigit = true
			continue
		}
		if hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		}
		if ch != '.' && ch != '-' {
			break
		}
	}
	if hasDigit {
		parts = append(parts, current)
	}
	return parts
}

func preferredDefaultModel(providerID string, models []domain.ModelInfo) string {
	preferred := defaultModelFor(providerID)
	if preferred == "" {
		return ""
	}
	for _, model := range models {
		if model.ID == preferred {
			return preferred
		}
	}
	return ""
}

func markRecommended(models []domain.ModelInfo, defaultModel string) {
	for i := range models {
		models[i].Recommended = models[i].ID == defaultModel
	}
}

func supportsGoogleGenerateContent(methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	for _, method := range methods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}
