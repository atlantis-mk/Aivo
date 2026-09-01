package app

import "aivo/core/domain"

func providerDefinitions() []ProviderDefinition {
	return defaultProviderRegistry.Definitions()
}

func builtInProviderDefinitions() []ProviderDefinition {
	defs := []ProviderDefinition{
		{
			ID: "openai", DisplayName: "OpenAI", Description: "OpenAI API or ChatGPT account models.",
			Transport: TransportOpenAIResponses, AuthTypes: []AuthType{AuthOAuthBrowser, AuthOAuthDevice, AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.openai.com/v1", BaseURLEnvVar: "OPENAI_BASE_URL", APIKeyEnvVars: []string{"OPENAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{
				Params: map[string]any{"store": false, "stream": true},
				ModelOverrides: map[string]domain.ProviderRequestOverride{
					"gpt-5-chat-latest":   {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5.1-chat-latest": {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5.2-chat-latest": {Params: map[string]any{"reasoning": map[string]any{"effort": "medium", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
					"gpt-5-search-api":    {Params: map[string]any{"reasoning": map[string]any{"effort": "none", "summary": "auto"}, "include": []any{"reasoning.encrypted_content"}}},
				},
			},
			Models: []domain.ModelInfo{
				model("openai", "gpt-5.5", "GPT-5.5", true, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.4", "GPT-5.4", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.4-mini", "GPT-5.4-Mini", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("openai", "gpt-5.3-codex-spark", "GPT-5.3-Codex-Spark", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search", "remote_mcp"}),
			},
		},
		{
			ID: "azure-openai", DisplayName: "Azure OpenAI", Description: "Azure OpenAI / Microsoft Foundry v1 OpenAI-compatible endpoint.",
			Aliases: []string{"azure", "azure-ai", "azure-openai-v1"}, Transport: TransportAzureOpenAI,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1", BaseURLEnvVar: "AZURE_OPENAI_BASE_URL", APIKeyEnvVars: []string{"AZURE_OPENAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"store": false, "stream": true}},
			Models: []domain.ModelInfo{
				model("azure-openai", "gpt-5.5", "GPT-5.5 deployment", true, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.4", "GPT-5.4 deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.4-mini", "GPT-5.4 Mini deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
				model("azure-openai", "gpt-5.3-codex", "GPT-5.3 Codex deployment", false, 400000, []string{"tools", "reasoning", "streaming", "web_search", "code_interpreter", "file_search"}),
			},
		},
		{
			ID: "claude-code", DisplayName: "Claude Code", Description: "Anthropic Messages API compatible Claude coding models.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.anthropic.com/v1", BaseURLEnvVar: "ANTHROPIC_BASE_URL", APIKeyEnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "claude-sonnet-4", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("claude-code", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "anthropic", DisplayName: "Anthropic", Description: "Anthropic Messages API.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.anthropic.com/v1", BaseURLEnvVar: "ANTHROPIC_BASE_URL", APIKeyEnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "claude-sonnet-4", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("anthropic", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "gemini", DisplayName: "Gemini", Description: "Google Gemini API.",
			Transport: TransportGoogleGemini, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GEMINI_BASE_URL", APIKeyEnvVars: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
			ModelFetch: ModelFetchGoogle, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				model("gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("gemini", "gemini-2.5-flash", "Gemini 2.5 Flash", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
			},
		},
		{
			ID: "google", DisplayName: "Google", Description: "Google Generative Language API.",
			Transport: TransportGoogleGemini, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GOOGLE_BASE_URL", APIKeyEnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"},
			ModelFetch: ModelFetchGoogle, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("google", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"})},
		},
		{
			ID: "google-vertex", DisplayName: "Google Vertex", Description: "Google Vertex AI Gemini publisher models.",
			Aliases: []string{"vertex", "vertex-ai", "google-vertex-ai"}, Transport: TransportGoogleVertex,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://us-central1-aiplatform.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/publishers/google",
			BaseURLEnvVar:  "GOOGLE_VERTEX_BASE_URL", APIKeyEnvVars: []string{"GOOGLE_VERTEX_ACCESS_TOKEN", "VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile: googleDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				model("google-vertex", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("google-vertex", "gemini-2.5-flash", "Gemini 2.5 Flash", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
				model("google-vertex", "gemini-3-pro-preview", "Gemini 3 Pro Preview", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"}),
			},
		},
		{
			ID: "amazon-bedrock", DisplayName: "Amazon Bedrock", Description: "AWS Bedrock Runtime Converse API.",
			Aliases: []string{"aws", "bedrock", "aws-bedrock"}, Transport: TransportBedrockConverse,
			AuthTypes: []AuthType{AuthAWSSDK}, DefaultAuthType: AuthAWSSDK,
			DefaultBaseURL: "https://bedrock-runtime.us-east-1.amazonaws.com", BaseURLEnvVar: "BEDROCK_BASE_URL",
			ModelFetch: ModelFetchStatic, DefaultModelID: "anthropic.claude-sonnet-4-20250514-v1:0", BuiltIn: true,
			Models: []domain.ModelInfo{
				model("amazon-bedrock", "anthropic.claude-sonnet-4-20250514-v1:0", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning"}),
				model("amazon-bedrock", "anthropic.claude-3-7-sonnet-20250219-v1:0", "Claude 3.7 Sonnet", false, 200000, []string{"tools", "reasoning"}),
				model("amazon-bedrock", "amazon.nova-pro-v1:0", "Amazon Nova Pro", false, 300000, []string{"tools"}),
			},
		},
		{
			ID: "openrouter", DisplayName: "OpenRouter", Description: "OpenAI-compatible routing provider.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://openrouter.ai/api/v1", BaseURLEnvVar: "OPENROUTER_BASE_URL", APIKeyEnvVars: []string{"OPENROUTER_API_KEY"},
			ModelFetch: ModelFetchOpenRouter, DefaultModelID: "openai/gpt-5-codex", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{
				Headers: map[string]string{"HTTP-Referer": "https://aivo.local", "X-Title": "Aivo"},
				Params:  map[string]any{"stream": true},
				ModelOverrides: map[string]domain.ProviderRequestOverride{
					"openai/gpt":         {Params: map[string]any{"reasoning": map[string]any{"effort": "medium"}}},
					"z-ai/glm-5.2":       {Params: map[string]any{"reasoning": map[string]any{"effort": "high"}}},
					"grok-3-mini":        {Params: map[string]any{"reasoning": map[string]any{"effort": "low"}}},
					"anthropic/claude-*": {Params: map[string]any{"reasoning": map[string]any{"effort": "none"}}},
				},
			},
			Models: []domain.ModelInfo{model("openrouter", "openai/gpt-5-codex", "GPT-5 Codex", true, 400000, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "xai", DisplayName: "xAI", Description: "xAI Grok models through the OpenAI-compatible API.",
			Aliases: []string{"grok", "x-ai", "x.ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.x.ai/v1", BaseURLEnvVar: "XAI_BASE_URL", APIKeyEnvVars: []string{"XAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "grok-4.3", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("xai", "grok-4.3", "Grok 4.3", true, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming", "web_search", "x_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("xai", "grok-build-0.1", "Grok Build 0.1", false, 256000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "mistral", DisplayName: "Mistral AI", Description: "Mistral La Plateforme chat models.",
			Aliases: []string{"mistral-ai", "la-plateforme"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.mistral.ai/v1", BaseURLEnvVar: "MISTRAL_BASE_URL", APIKeyEnvVars: []string{"MISTRAL_API_KEY"},
			ModelFetch: ModelFetchMistral, DefaultModelID: "mistral-medium-latest", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("mistral", "mistral-medium-latest", "Mistral Medium", true, 256000, 0, []string{"tools", "vision", "streaming"}),
				modelWithOutput("mistral", "devstral-latest", "Devstral", false, 256000, 0, []string{"tools", "streaming"}),
				modelWithOutput("mistral", "codestral-latest", "Codestral", false, 128000, 0, []string{"tools", "streaming"}),
				modelWithOutput("mistral", "magistral-medium-latest", "Magistral Medium", false, 128000, 0, []string{"reasoning", "streaming"}),
			},
		},
		{
			ID: "groq", DisplayName: "Groq", Description: "GroqCloud OpenAI-compatible high-speed inference.",
			Aliases: []string{"groqcloud"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.groq.com/openai/v1", BaseURLEnvVar: "GROQ_BASE_URL", APIKeyEnvVars: []string{"GROQ_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-oss-120b", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("groq", "openai/gpt-oss-120b", "GPT OSS 120B", true, 131072, 65536, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("groq", "openai/gpt-oss-20b", "GPT OSS 20B", false, 131072, 65536, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("groq", "qwen/qwen3.6-27b", "Qwen3.6 27B", false, 131072, 32768, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "deepinfra", DisplayName: "DeepInfra", Description: "DeepInfra OpenAI-compatible model catalog.",
			Aliases: []string{"deep-infra"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.deepinfra.com/v1/openai", BaseURLEnvVar: "DEEPINFRA_BASE_URL", APIKeyEnvVars: []string{"DEEPINFRA_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("deepinfra", "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", "Qwen3 Coder 480B Turbo", true, 256000, 0, []string{"tools", "reasoning", "streaming"}),
				model("deepinfra", "deepseek-ai/DeepSeek-V3.1", "DeepSeek V3.1", false, 128000, []string{"tools", "streaming"}),
				model("deepinfra", "openai/gpt-oss-120b", "GPT OSS 120B", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "cerebras", DisplayName: "Cerebras", Description: "Cerebras Inference OpenAI-compatible endpoint.",
			Aliases: []string{"cerebras-ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.cerebras.ai/v1", BaseURLEnvVar: "CEREBRAS_BASE_URL", APIKeyEnvVars: []string{"CEREBRAS_API_KEY"},
			ModelFetch: ModelFetchCerebras, DefaultModelID: "zai-glm-4.7", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("cerebras", "zai-glm-4.7", "Z.ai GLM 4.7", true, 128000, 8192, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("cerebras", "gpt-oss-120b", "GPT OSS 120B", false, 128000, 8192, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("cerebras", "gemma-4-31b", "Gemma 4 31B", false, 128000, 8192, []string{"tools", "vision", "streaming"}),
			},
		},
		{
			ID: "baseten", DisplayName: "Baseten", Description: "Baseten OpenAI-compatible inference endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://inference.baseten.co/v1", BaseURLEnvVar: "BASETEN_BASE_URL", APIKeyEnvVars: []string{"BASETEN_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "custom-profile", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("baseten", "custom-profile", "Custom profile", true, 0, []string{"streaming"})},
		},
		{
			ID: "deepseek", DisplayName: "DeepSeek", Description: "DeepSeek OpenAI-compatible chat models.",
			Aliases: []string{"deep-seek"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.deepseek.com/v1", BaseURLEnvVar: "DEEPSEEK_BASE_URL", APIKeyEnvVars: []string{"DEEPSEEK_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "deepseek-chat", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("deepseek", "deepseek-chat", "DeepSeek Chat", true, 128000, []string{"tools", "streaming"}),
				model("deepseek", "deepseek-reasoner", "DeepSeek Reasoner", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "fireworks", DisplayName: "Fireworks AI", Description: "Fireworks AI OpenAI-compatible inference endpoint.",
			Aliases: []string{"fireworks-ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.fireworks.ai/inference/v1", BaseURLEnvVar: "FIREWORKS_BASE_URL", APIKeyEnvVars: []string{"FIREWORKS_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "accounts/fireworks/models/glm-5p2", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("fireworks", "accounts/fireworks/models/glm-5p2", "GLM 5.2", true, 256000, 65536, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "together", DisplayName: "Together AI", Description: "Together AI serverless OpenAI-compatible inference.",
			Aliases: []string{"together-ai", "togetherai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.together.xyz/v1", BaseURLEnvVar: "TOGETHER_BASE_URL", APIKeyEnvVars: []string{"TOGETHER_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "moonshotai/Kimi-K2.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("together", "moonshotai/Kimi-K2.5", "Kimi K2.5", true, 256000, []string{"tools", "streaming"}),
				model("together", "zai-org/GLM-5", "GLM-5", false, 256000, []string{"tools", "reasoning", "streaming"}),
				model("together", "openai/gpt-oss-120b", "GPT OSS 120B", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "perplexity", DisplayName: "Perplexity", Description: "Perplexity Sonar search-grounded chat models.",
			Aliases: []string{"pplx"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.perplexity.ai", BaseURLEnvVar: "PERPLEXITY_BASE_URL", APIKeyEnvVars: []string{"PERPLEXITY_API_KEY"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "sonar-pro", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("perplexity", "sonar-pro", "Sonar Pro", true, 200000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar", "Sonar", false, 127000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar-reasoning-pro", "Sonar Reasoning Pro", false, 128000, []string{"tools", "search", "web_search", "reasoning", "streaming"}),
				model("perplexity", "sonar-deep-research", "Sonar Deep Research", false, 128000, []string{"search", "web_search", "reasoning", "streaming"}),
			},
		},
		{
			ID: "github-copilot", DisplayName: "GitHub Copilot", Description: "GitHub Copilot Chat completions endpoint using a Copilot bearer token.",
			Aliases: []string{"copilot", "github-copilot-api"}, Transport: TransportGitHubCopilot,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.githubcopilot.com", BaseURLEnvVar: "GITHUB_COPILOT_BASE_URL", APIKeyEnvVars: []string{"GITHUB_COPILOT_TOKEN", "COPILOT_TOKEN", "GH_COPILOT_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "gpt-5.1-codex-max", BuiltIn: true,
			RequestProfile: githubCopilotRequestProfile(),
			Models: []domain.ModelInfo{
				model("github-copilot", "gpt-5.1-codex-max", "GPT-5.1 Codex Max", true, 400000, []string{"tools", "reasoning", "streaming"}),
				model("github-copilot", "gpt-5.1-codex", "GPT-5.1 Codex", false, 400000, []string{"tools", "reasoning", "streaming"}),
				model("github-copilot", "gpt-4.1", "GPT-4.1", false, 1000000, []string{"tools", "streaming"}),
				model("github-copilot", "claude-sonnet-4.5", "Claude Sonnet 4.5", false, 200000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "custom-api", DisplayName: "Custom API", Description: "User-configured OpenAI, Anthropic, Google, or compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey, AuthNone}, DefaultAuthType: AuthAPIKey,
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "custom-profile", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("custom-api", "custom-profile", "Custom profile", true, 0, []string{"streaming"})},
		},
	}
	return defs
}

func model(providerID, id, name string, recommended bool, context int, capabilities []string) domain.ModelInfo {
	modalities := []string{"text"}
	if containsString(capabilities, "vision") {
		modalities = append(modalities, "image")
	}
	var reasoningControls []string
	if containsString(capabilities, "reasoning") {
		reasoningControls = []string{"effort"}
	}
	return domain.ModelInfo{
		ID: id, ProviderID: providerID, Name: name, Recommended: recommended, ContextLength: context,
		Capabilities: capabilities, Streaming: containsString(capabilities, "streaming"), ToolSupport: containsString(capabilities, "tools"),
		ReasoningControls: reasoningControls, Status: "active", Modalities: modalities,
	}
}

func modelWithOutput(providerID, id, name string, recommended bool, context int, output int, capabilities []string) domain.ModelInfo {
	model := model(providerID, id, name, recommended, context, capabilities)
	model.OutputLimit = output
	return model
}

func anthropicDefaultRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		Headers: map[string]string{"anthropic-version": "2023-06-01"},
		Params:  map[string]any{"max_tokens": 4096, "stream": true},
	}
}

func googleDefaultRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{}
}

func githubCopilotRequestProfile() domain.ProviderRequestProfile {
	return domain.ProviderRequestProfile{
		Headers: map[string]string{
			"Copilot-Integration-Id": "vscode-chat",
			"Editor-Plugin-Version":  "aivo/0.0.0",
			"Editor-Version":         "Aivo/0.0.0",
			"OpenAI-Intent":          "conversation-panel",
		},
		Params: map[string]any{"stream": true},
	}
}
