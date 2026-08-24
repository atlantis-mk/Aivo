package app

import (
	"testing"

	"aivo/core/domain"
)

func TestApplyChatRequestGenerationSettingsUsesProviderShape(t *testing.T) {
	temperature := 0.35
	topP := 0.72
	openAI := applyChatRequestGenerationSettings(ResolvedModelRoute{Transport: TransportOpenAICompatible}, domain.ChatRequest{
		Temperature: &temperature, TopP: &topP, Options: map[string]any{"reasoning_effort": "high", "temperature": 0.1},
	})
	if openAI.Provider.RequestParams["temperature"] != temperature {
		t.Fatalf("OpenAI params = %#v", openAI.Provider.RequestParams)
	}
	if openAI.Provider.RequestParams["top_p"] != topP || openAI.Provider.RequestParams["reasoning_effort"] != "high" {
		t.Fatalf("OpenAI params = %#v", openAI.Provider.RequestParams)
	}
	google := applyChatRequestGenerationSettings(ResolvedModelRoute{Transport: TransportGoogleGemini}, domain.ChatRequest{Temperature: &temperature, TopP: &topP, Options: map[string]any{"candidateCount": 2}})
	generation, _ := google.Provider.RequestParams["generationConfig"].(map[string]any)
	if generation["temperature"] != temperature || generation["topP"] != topP || google.Provider.RequestParams["candidateCount"] != 2 {
		t.Fatalf("Google params = %#v", google.Provider.RequestParams)
	}
}
