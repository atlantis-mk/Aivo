package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) toolsForModelRoute(ctx context.Context, cfg domain.AppConfig, route ResolvedModelRoute, specs []domain.ToolSpec) []domain.ToolSpec {
	webSearch := normalizeWebSearchRuntimeConfig(cfg.WebSearch)
	nativeTools := normalizeNativeToolsRuntimeConfig(cfg.NativeTools)
	hostedSearch := webSearch.Mode != domain.WebSearchModeDisabled && routeSupportsHostedWebSearch(ctx, s, route, specs)
	hostedFetch := routeSupportsHostedWebFetch(ctx, s, route, specs)
	out := make([]domain.ToolSpec, 0, len(specs))
	model, modelKnown := s.modelInfoForRoute(ctx, route)
	accountCapabilities := capabilitiesForProviderAccount(route)
	for _, spec := range specs {
		if (spec.ActivationPolicy == providerDeclarationActivationPolicy || spec.ActivationPolicy == providerAccountActivationPolicy) && nativeToolDisabled(nativeTools, spec.Name) {
			continue
		}
		if spec.ActivationPolicy == providerDeclarationActivationPolicy && !isChatGPTCodexRoute(route) {
			continue
		}
		if spec.ActivationPolicy == providerAccountActivationPolicy && !isChatGPTCodexRoute(route) {
			continue
		}
		if isChatGPTCodexRoute(route) {
			switch spec.Name {
			case ExecCommandToolName:
				if !modelKnown || !declaredModelCapabilitySupported(model, codexShellCapability) {
					continue
				}
			case "web_search":
				if spec.ActivationPolicy != providerDeclarationActivationPolicy || webSearch.Mode == domain.WebSearchModeDisabled || !modelKnown || !declaredModelCapabilitySupported(model, codexWebSearchCapability) {
					continue
				}
			case CodexImagegenToolName:
				if spec.ActivationPolicy != providerAccountActivationPolicy || !accountCapabilities.ImageGeneration || !accountCapabilities.NamespaceTools {
					continue
				}
			}
		}
		switch spec.Name {
		case "web_search":
			if webSearch.Mode == domain.WebSearchModeDisabled {
				continue
			}
			switch webSearch.Route {
			case domain.WebSearchRouteProvider:
				if hostedSearch {
					out = append(out, hostedWebSearchToolSpec(webSearch, route, model))
				} else if isChatGPTCodexRoute(route) && modelKnown && codexModelUsesResponsesLite(model) {
					out = append(out, localToolSpec(spec))
				}
			case domain.WebSearchRouteLocal:
				out = append(out, localToolSpec(spec))
			default:
				if hostedSearch {
					out = append(out, hostedWebSearchToolSpec(webSearch, route, model))
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
	if isChatGPTCodexRoute(route) && modelKnown && webSearch.Mode != domain.WebSearchModeDisabled && webSearch.Route != domain.WebSearchRouteLocal && !nativeToolDisabled(nativeTools, "web_search") && !codexModelUsesResponsesLite(model) && declaredModelCapabilitySupported(model, codexWebSearchCapability) && !toolSpecNamed(out, "web_search") {
		out = append(out, hostedWebSearchToolSpec(webSearch, route, model))
	}
	return appendDynamicallyDiscoveredHostedToolSpecs(ctx, s, route, webSearch, nativeTools, out)
}

func declaredModelCapabilitySupported(model domain.ModelInfo, capability string) bool {
	return containsString(model.DeclaredCapabilities, capability) && modelSupportsCapability(model, capability)
}

func (s *Service) providerDeclaredLocalToolActivations(ctx context.Context, requestedModel *domain.ModelRef) map[string]string {
	if s == nil {
		return nil
	}
	cfg, err := s.AppConfig(ctx)
	if err != nil {
		return nil
	}
	routes, err := s.ResolveModelRoutes(ctx, cfg, requestedModel)
	if err != nil {
		return nil
	}
	policy := normalizeProviderRuntimePolicy(cfg.ProviderPolicy)
	webSearch := normalizeWebSearchRuntimeConfig(cfg.WebSearch)
	if !providerPolicyBool(policy.EnableFallback) && len(routes) > 1 {
		routes = routes[:1]
	}
	activations := map[string]string{}
	for _, route := range routes {
		if !isChatGPTCodexRoute(route) {
			continue
		}
		accountCapabilities := capabilitiesForProviderAccount(route)
		if accountCapabilities.ImageGeneration && accountCapabilities.NamespaceTools && !nativeToolDisabled(normalizeNativeToolsRuntimeConfig(cfg.NativeTools), CodexImagegenToolName) {
			activations[CodexImagegenToolName] = "providerAccount"
		}
		s.ensureDynamicProviderCapabilities(ctx, route)
		model, ok := s.modelInfoForRoute(ctx, route)
		if ok && webSearch.Mode != domain.WebSearchModeDisabled && !nativeToolDisabled(normalizeNativeToolsRuntimeConfig(cfg.NativeTools), "web_search") && declaredModelCapabilitySupported(model, codexWebSearchCapability) && (webSearch.Route == domain.WebSearchRouteLocal || codexModelUsesResponsesLite(model)) {
			activations["web_search"] = "providerCapability"
		}
	}
	return activations
}

func appendDynamicallyDiscoveredHostedToolSpecs(ctx context.Context, service *Service, route ResolvedModelRoute, webSearch domain.WebSearchConfig, nativeConfig domain.NativeToolsConfig, specs []domain.ToolSpec) []domain.ToolSpec {
	if service == nil {
		return specs
	}
	model, ok := service.modelInfoForRoute(ctx, route)
	if !ok {
		return specs
	}
	discovered := dynamicallyDiscoveredNativeTools(model)
	if len(discovered) == 0 {
		return specs
	}
	disabled := map[string]bool{}
	for _, name := range nativeConfig.Disabled {
		disabled[strings.ToLower(strings.TrimSpace(name))] = true
	}
	providerID := normalizedRouteProviderID(route)
	if (providerID != "anthropic" && providerID != "claude-code") || route.Transport != TransportAnthropicMessages {
		return specs
	}
	if discovered["web_search"] && !disabled["web_search"] && webSearch.Mode != domain.WebSearchModeDisabled && !toolSpecNamed(specs, "web_search") {
		specs = append(specs, hostedWebSearchToolSpec(webSearch, route, model))
	}
	if discovered["web_fetch"] && !disabled["web_fetch"] && !toolSpecNamed(specs, "web_fetch") {
		specs = append(specs, hostedWebFetchToolSpec(route))
	}
	if discovered["code_execution"] && !disabled["code_execution"] && !toolSpecNamed(specs, "code_execution") {
		specs = append(specs, domain.ToolSpec{
			Name: "code_execution", Description: "Run code in the active model provider's hosted sandbox.",
			Hosted:     &domain.HostedToolSpec{Type: "code_execution_20250825"},
			Capability: "hosted.code", Category: "hosted", RequiresNetwork: true,
		})
	}
	return specs
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
	if isChatGPTCodexRoute(route) {
		return !codexModelUsesResponsesLite(model) && declaredModelCapabilitySupported(model, codexWebSearchCapability)
	}
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

func hostedWebSearchToolSpec(config domain.WebSearchConfig, route ResolvedModelRoute, model domain.ModelInfo) domain.ToolSpec {
	externalWebAccess := config.Mode != domain.WebSearchModeCached
	var indexedWebAccess *bool
	if config.Mode == domain.WebSearchModeIndexed {
		indexed := true
		indexedWebAccess = &indexed
	}
	hostedType := "web_search"
	providerID := normalizedRouteProviderID(route)
	allowedDomains := append([]string(nil), config.AllowedDomains...)
	var searchContentTypes []string
	if isChatGPTCodexRoute(route) && model.WebSearchToolType == "text_and_image" {
		searchContentTypes = []string{"text", "image"}
	}
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
			Type:               hostedType,
			ExternalWebAccess:  &externalWebAccess,
			IndexedWebAccess:   indexedWebAccess,
			SearchContextSize:  config.SearchContextSize,
			SearchContentTypes: searchContentTypes,
			AllowedDomains:     allowedDomains,
			UserLocation:       cloneWebSearchUserLocation(config.UserLocation),
		},
		Capability:      "web.search",
		Category:        "web",
		RequiresNetwork: true,
	}
}

func codexModelUsesResponsesLite(model domain.ModelInfo) bool {
	return model.UseResponsesLite != nil && *model.UseResponsesLite
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
	if config.Mode != domain.WebSearchModeDisabled && config.Mode != domain.WebSearchModeCached && config.Mode != domain.WebSearchModeIndexed && config.Mode != domain.WebSearchModeLive {
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
	config.Disabled = normalizeStringList(config.Disabled)
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

func nativeToolDisabled(config domain.NativeToolsConfig, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, disabled := range config.Disabled {
		if strings.ToLower(strings.TrimSpace(disabled)) == name {
			return true
		}
	}
	return false
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
