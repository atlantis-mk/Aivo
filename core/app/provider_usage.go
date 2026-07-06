package app

import (
	"math"
	"strings"

	"aivo/core/domain"
)

func estimateRouteInputTokens(req modelRequirement) int {
	return req.InputTokens
}

func estimateTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	tokens := int(math.Ceil(float64(len([]rune(text))) / 4.0))
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateRouteCostMicros(route ResolvedModelRoute, inputTokens int, outputTokens int) int64 {
	model, ok := findModelInfo(route.Definition.Models, route.Model.ModelID)
	if !ok && route.Model.ModelID != "" {
		model = domain.ModelInfo{ID: route.Model.ModelID, ProviderID: route.Model.ProviderID}
	}
	return estimateCostMicros(model, inputTokens, outputTokens)
}

func estimateCostMicros(model domain.ModelInfo, inputTokens int, outputTokens int) int64 {
	if len(model.Pricing) == 0 {
		return 0
	}
	inputRate := firstPrice(model.Pricing, "input", "prompt", "input_per_million", "prompt_per_million")
	outputRate := firstPrice(model.Pricing, "output", "completion", "output_per_million", "completion_per_million")
	total := (float64(inputTokens)*inputRate + float64(outputTokens)*outputRate) / 1_000_000.0
	return int64(math.Round(total * 1_000_000.0))
}

func firstPrice(pricing map[string]float64, keys ...string) float64 {
	for _, key := range keys {
		if value := pricing[key]; value > 0 {
			return value
		}
	}
	return 0
}
