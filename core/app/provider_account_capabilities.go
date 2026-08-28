package app

type providerAccountCapabilities struct {
	NamespaceTools  bool
	ImageGeneration bool
	WebSearch       bool
}

func capabilitiesForProviderAccount(route ResolvedModelRoute) providerAccountCapabilities {
	if !isChatGPTCodexRoute(route) {
		return providerAccountCapabilities{}
	}
	// These are capabilities of the authenticated ChatGPT Codex provider
	// surface, not model-catalog claims. Keep this adapter contract aligned with
	// Codex modelProvider/capabilities/read.
	return providerAccountCapabilities{
		NamespaceTools:  true,
		ImageGeneration: true,
		WebSearch:       true,
	}
}
