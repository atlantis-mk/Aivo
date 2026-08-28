package app

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"aivo/core/domain"
)

const (
	codexShellCapability     = "codex_shell"
	codexWebSearchCapability = "codex_web_search"
	codexRuntimeCapability   = "codex_runtime"
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
		model := domain.ModelInfo{
			ID:                    id,
			ProviderID:            providerID,
			Name:                  firstNonEmpty(item.DisplayName, item.Name, id),
			ContextLength:         firstPositiveInt(item.ContextWindow, item.MaxContextWindow),
			MaxContextLength:      item.MaxContextWindow,
			AutoCompactTokenLimit: item.AutoCompactTokenLimit,
			Modalities:            normalizeCodexModalities(item.InputModalities),
		}
		applyCodexRuntimeDeclarations(&model, item)
		applyCodexLocalToolDeclarations(&model, item)
		models = append(models, model)
	}
	return finalizeParsedModels(models, "")
}

func applyCodexRuntimeDeclarations(model *domain.ModelInfo, item codexModelInfo) {
	if model == nil {
		return
	}
	declareCapability(model, codexRuntimeCapability, true)
	if effort, ok := declaredCodexEnum(item.DefaultReasoningLevel); ok && codexReasoningEffortSupported(effort) {
		model.DefaultReasoningEffort = effort
	}
	for _, level := range item.SupportedReasoningLevels {
		effort := strings.TrimSpace(level.Effort)
		if codexReasoningEffortSupported(effort) {
			model.SupportedReasoningEfforts = appendUniqueStrings(model.SupportedReasoningEfforts, effort)
		}
	}
	if len(model.SupportedReasoningEfforts) > 0 {
		declareCapability(model, "reasoning", true)
		model.ReasoningControls = appendUniqueStrings(model.ReasoningControls, "effort")
	}
	model.SupportsVerbosity = item.SupportVerbosity
	if verbosity, ok := declaredCodexEnum(item.DefaultVerbosity); ok && codexVerbositySupported(verbosity) {
		model.DefaultVerbosity = verbosity
	}
	for _, tier := range item.ServiceTiers {
		if id := strings.TrimSpace(tier.ID); id != "" {
			model.ServiceTiers = appendUniqueStrings(model.ServiceTiers, id)
		}
	}
	for _, tier := range item.AdditionalSpeedTiers {
		if tier = strings.TrimSpace(tier); tier != "" {
			model.ServiceTiers = appendUniqueStrings(model.ServiceTiers, tier)
		}
	}
	model.DefaultServiceTier = strings.TrimSpace(item.DefaultServiceTier)
	model.SupportsParallelToolCalls = item.SupportsParallelToolCalls
	model.SupportsImageDetailOriginal = item.SupportsImageDetailOriginal
	model.UseResponsesLite = item.UseResponsesLite
	if searchType, ok := declaredCodexEnum(item.WebSearchToolType); ok {
		model.WebSearchToolTypeKnown = true
		supported := item.SupportsSearchTool == nil || *item.SupportsSearchTool
		switch searchType {
		case "text", "text_and_image":
			model.WebSearchToolType = searchType
			declareCapability(model, codexWebSearchCapability, supported)
		default:
			declareCapability(model, codexWebSearchCapability, false)
		}
	} else if item.SupportsSearchTool != nil {
		model.WebSearchToolTypeKnown = true
		if *item.SupportsSearchTool {
			model.WebSearchToolType = "text"
		}
		declareCapability(model, codexWebSearchCapability, *item.SupportsSearchTool)
	}
}

func normalizeCodexModalities(values []string) []string {
	out := []string{}
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "text", "image", "audio":
			out = appendUniqueStrings(out, strings.ToLower(strings.TrimSpace(value)))
		}
	}
	return out
}

func codexReasoningEffortSupported(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
		return true
	default:
		return false
	}
}

func codexVerbositySupported(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func applyCodexLocalToolDeclarations(model *domain.ModelInfo, item codexModelInfo) {
	if shellType, ok := declaredCodexEnum(item.ShellType); ok {
		switch shellType {
		case "unified_exec", "default", "local", "shell_command":
			declareCapability(model, codexShellCapability, true)
		case "disabled":
			declareCapability(model, codexShellCapability, false)
		}
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func declaredCodexEnum(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || isExplicitJSONNull(raw) {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func isExplicitJSONNull(raw json.RawMessage) bool {
	return len(raw) > 0 && strings.TrimSpace(string(raw)) == "null"
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

func parseMistralModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body mistralModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || item.Archived {
			continue
		}
		model := domain.ModelInfo{
			ID: id, ProviderID: providerID, Name: firstNonEmpty(item.Name, id),
			ContextLength: item.MaxContextLength,
		}
		applyDeclaredBooleanCapability(&model, item.Capabilities, "function_calling", "tools")
		applyDeclaredBooleanCapability(&model, item.Capabilities, "vision", "vision")
		models = append(models, model)
	}
	return finalizeParsedModels(models, "")
}

func parseOpenRouterModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body openRouterModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		model := domain.ModelInfo{
			ID: id, ProviderID: providerID, Name: firstNonEmpty(item.Name, id),
			ContextLength: item.ContextLength, OutputLimit: item.TopProvider.MaxCompletionTokens,
			Modalities: append([]string(nil), item.Architecture.InputModalities...),
		}
		if item.SupportedParameters != nil {
			parameters := stringSet(*item.SupportedParameters)
			declareCapability(&model, "tools", parameters["tools"] || parameters["tool_choice"])
			declareCapability(&model, "reasoning", parameters["reasoning"])
			if parameters["stream"] {
				declareCapability(&model, "streaming", true)
			}
		}
		models = append(models, model)
	}
	return finalizeParsedModels(models, "")
}

func parseCerebrasModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body cerebrasModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		model := domain.ModelInfo{
			ID: id, ProviderID: providerID, Name: firstNonEmpty(item.Name, id),
			Deprecated: item.Deprecated, ContextLength: item.Limits.MaxContextLength,
			OutputLimit: item.Limits.MaxCompletionTokens,
		}
		applyDeclaredBooleanAliases(&model, item.Capabilities, "tools", "tools", "function_calling")
		applyDeclaredBooleanAliases(&model, item.Capabilities, "streaming", "streaming")
		applyDeclaredBooleanAliases(&model, item.Capabilities, "reasoning", "reasoning")
		applyDeclaredBooleanAliases(&model, item.Capabilities, "vision", "vision")
		models = append(models, model)
	}
	return finalizeParsedModels(models, "")
}

func applyDeclaredBooleanCapability(model *domain.ModelInfo, raw map[string]json.RawMessage, source, capability string) {
	value, declared := declaredBoolean(raw, source)
	if declared {
		declareCapability(model, capability, value)
	}
}

func applyDeclaredBooleanAliases(model *domain.ModelInfo, raw map[string]json.RawMessage, capability string, sources ...string) {
	declared := false
	supported := false
	for _, source := range sources {
		value, ok := declaredBoolean(raw, source)
		declared = declared || ok
		supported = supported || (ok && value)
	}
	if declared {
		declareCapability(model, capability, supported)
	}
}

func declaredBoolean(raw map[string]json.RawMessage, key string) (bool, bool) {
	value, ok := raw[key]
	if !ok {
		return false, false
	}
	var supported *bool
	if err := json.Unmarshal(value, &supported); err == nil && supported != nil {
		return *supported, true
	}
	var object struct {
		Supported *bool `json:"supported"`
	}
	if err := json.Unmarshal(value, &object); err == nil && object.Supported != nil {
		return *object.Supported, true
	}
	return false, false
}

func declareCapability(model *domain.ModelInfo, capability string, supported bool) {
	model.DeclaredCapabilities = appendUniqueStrings(model.DeclaredCapabilities, capability)
	if supported {
		model.Capabilities = appendUniqueStrings(model.Capabilities, capability)
	}
	switch capability {
	case "tools":
		model.ToolSupport = supported
	case "streaming":
		model.Streaming = supported
	case "reasoning":
		if supported {
			model.ReasoningControls = appendUniqueStrings(model.ReasoningControls, "effort")
		}
	case "vision":
		if supported {
			model.Modalities = appendUniqueStrings(model.Modalities, "image")
		}
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[strings.TrimSpace(value)] = true
	}
	return out
}

func parseAnthropicModels(raw []byte, providerID string) ([]domain.ModelInfo, string, error) {
	var body anthropicModelsResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, "", errors.New("provider model response could not be parsed")
	}
	models := make([]domain.ModelInfo, 0, len(body.Data))
	for _, item := range body.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		capabilities, nativeTools, reasoningControls, modalities := parseAnthropicModelCapabilities(item.Capabilities)
		models = append(models, domain.ModelInfo{
			ID: id, ProviderID: providerID, Name: firstNonEmpty(item.DisplayName, id),
			ContextLength: item.MaxInputTokens, OutputLimit: item.MaxTokens,
			Capabilities: capabilities, DeclaredCapabilities: anthropicDeclaredCapabilityDimensions(item.Capabilities),
			NativeTools: nativeTools, NativeToolsKnown: item.Capabilities != nil,
			Modalities: modalities, Streaming: true, ToolSupport: true, ReasoningControls: reasoningControls,
		})
	}
	return finalizeParsedModels(models, "")
}

func anthropicDeclaredCapabilityDimensions(raw map[string]json.RawMessage) []string {
	declared := []string{}
	for key, value := range raw {
		if _, known := anthropicCapabilitySupported(key, value); !known {
			continue
		}
		switch key {
		case "thinking", "effort":
			declared = appendUniqueStrings(declared, "reasoning")
		case "image_input":
			declared = appendUniqueStrings(declared, "vision")
		case "web_search", "web_fetch", "code_execution", "code_interpreter", "x_search":
			declared = appendUniqueStrings(declared, key)
		}
	}
	sort.Strings(declared)
	return declared
}

func parseAnthropicModelCapabilities(raw map[string]json.RawMessage) ([]string, []string, []string, []string) {
	capabilities := []string{"streaming", "tools"}
	nativeTools := []string{}
	reasoningControls := []string{}
	modalities := []string{"text"}
	if raw == nil {
		return capabilities, nil, nil, modalities
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		supported, known := anthropicCapabilitySupported(key, raw[key])
		if !known || !supported {
			continue
		}
		capabilities = appendUniqueStrings(capabilities, key)
		switch key {
		case "thinking", "effort":
			capabilities = appendUniqueStrings(capabilities, "reasoning")
			reasoningControls = []string{"effort"}
		case "image_input":
			capabilities = appendUniqueStrings(capabilities, "vision")
			modalities = appendUniqueStrings(modalities, "image")
		case "web_search", "web_fetch", "code_execution", "code_interpreter", "x_search":
			nativeTools = append(nativeTools, key)
		}
	}
	return capabilities, nativeTools, reasoningControls, modalities
}

func anthropicCapabilitySupported(key string, raw json.RawMessage) (bool, bool) {
	var capability struct {
		Supported *bool `json:"supported"`
	}
	if err := json.Unmarshal(raw, &capability); err != nil {
		return false, false
	}
	if capability.Supported != nil {
		return *capability.Supported, true
	}
	if key != "effort" {
		return false, false
	}
	var levels map[string]struct {
		Supported *bool `json:"supported"`
	}
	if err := json.Unmarshal(raw, &levels); err != nil {
		return false, false
	}
	known := false
	for _, level := range levels {
		if level.Supported == nil {
			continue
		}
		known = true
		if *level.Supported {
			return true, true
		}
	}
	return false, known
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
