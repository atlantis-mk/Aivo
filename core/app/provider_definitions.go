package app

import "aivo/core/domain"

func providerDefinitions() []ProviderDefinition {
	return defaultProviderRegistry.Definitions()
}

func nativeWebSearch(toolType string, capabilities ...string) ProviderNativeHostedTools {
	if len(capabilities) == 0 {
		capabilities = []string{"web_search"}
	}
	return ProviderNativeHostedTools{WebSearch: ProviderNativeHostedTool{Type: toolType, Capabilities: append([]string(nil), capabilities...)}}
}

func builtInProviderDefinitions() []ProviderDefinition {
	defs := []ProviderDefinition{
		{
			ID: "openai", DisplayName: "OpenAI", Description: "OpenAI API or ChatGPT account models.",
			Transport: TransportOpenAIResponses, AuthTypes: []AuthType{AuthOAuthBrowser, AuthOAuthDevice, AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.openai.com/v1", BaseURLEnvVar: "OPENAI_BASE_URL", APIKeyEnvVars: []string{"OPENAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.5", BuiltIn: true,
			NativeHostedTools: nativeWebSearch("web_search"),
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
			NativeHostedTools: nativeWebSearch("web_search"),
			RequestProfile:    domain.ProviderRequestProfile{Params: map[string]any{"store": false, "stream": true}},
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
			RequestProfile:    anthropicDefaultRequestProfile(),
			NativeHostedTools: nativeWebSearch("web_search_20250305"),
			Models:            []domain.ModelInfo{model("claude-code", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "anthropic", DisplayName: "Anthropic", Description: "Anthropic Messages API.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.anthropic.com/v1", BaseURLEnvVar: "ANTHROPIC_BASE_URL", APIKeyEnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "claude-sonnet-4", BuiltIn: true,
			RequestProfile:    anthropicDefaultRequestProfile(),
			NativeHostedTools: nativeWebSearch("web_search_20250305"),
			Models:            []domain.ModelInfo{model("anthropic", "claude-sonnet-4", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch"})},
		},
		{
			ID: "gemini", DisplayName: "Gemini", Description: "Google Gemini API.",
			Transport: TransportGoogleGemini, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GEMINI_BASE_URL", APIKeyEnvVars: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
			ModelFetch: ModelFetchGoogle, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile:    googleDefaultRequestProfile(),
			NativeHostedTools: nativeWebSearch("google_search"),
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
			RequestProfile:    googleDefaultRequestProfile(),
			NativeHostedTools: nativeWebSearch("google_search"),
			Models:            []domain.ModelInfo{model("google", "gemini-2.5-pro", "Gemini 2.5 Pro", true, 1000000, []string{"tools", "reasoning", "streaming", "web_search", "web_fetch", "code_execution", "file_search"})},
		},
		{
			ID: "google-vertex", DisplayName: "Google Vertex", Description: "Google Vertex AI Gemini publisher models.",
			Aliases: []string{"vertex", "vertex-ai", "google-vertex-ai"}, Transport: TransportGoogleVertex,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://us-central1-aiplatform.googleapis.com/v1/projects/YOUR_PROJECT_ID/locations/us-central1/publishers/google",
			BaseURLEnvVar:  "GOOGLE_VERTEX_BASE_URL", APIKeyEnvVars: []string{"GOOGLE_VERTEX_ACCESS_TOKEN", "VERTEX_ACCESS_TOKEN", "GOOGLE_OAUTH_ACCESS_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "gemini-2.5-pro", BuiltIn: true,
			RequestProfile:    googleDefaultRequestProfile(),
			NativeHostedTools: nativeWebSearch("google_search"),
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
			DefaultResponsesAPI: true,
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
			NativeHostedTools:          ProviderNativeHostedTools{WebSearch: ProviderNativeHostedTool{Type: "web_search", Capabilities: []string{"web_search"}, MaxAllowedDomains: 5}},
			DefaultResponsesAPI:        true,
			ResponsesAPIForHostedTools: true,
			RequestProfile:             domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("xai", "grok-4.3", "Grok 4.3", true, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming", "web_search", "x_search", "code_interpreter", "file_search", "remote_mcp"}),
				model("xai", "grok-build-0.1", "Grok Build 0.1", false, 256000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "xiaomi", DisplayName: "Xiaomi MiMo", Description: "Xiaomi MiMo models through the OpenAI-compatible Responses API.",
			Aliases: []string{"mimo", "xiaomi-mimo"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.xiaomimimo.com/v1", BaseURLEnvVar: "MIMO_BASE_URL", APIKeyEnvVars: []string{"MIMO_API_KEY", "XIAOMI_MIMO_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "mimo-v2.5-pro", BuiltIn: true,
			DefaultResponsesAPI: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("xiaomi", "mimo-v2.5-pro", "MiMo V2.5 Pro", true, 1048576, []string{"tools", "reasoning", "streaming"}),
				model("xiaomi", "mimo-v2.5", "MiMo V2.5", false, 1048576, []string{"tools", "reasoning", "streaming"}),
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
			DefaultResponsesAPI: true,
			RequestProfile:      domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
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
			ResponsesBaseURL:           "https://api.deepseek.com",
			NativeHostedTools:          nativeWebSearch("web_search"),
			DefaultResponsesAPI:        true,
			ResponsesAPIForHostedTools: true,
			RequestProfile:             domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("deepseek", "deepseek-chat", "DeepSeek Chat", true, 128000, []string{"tools", "streaming"}),
				model("deepseek", "deepseek-reasoner", "DeepSeek Reasoner", false, 128000, []string{"tools", "reasoning", "streaming"}),
				model("deepseek", "deepseek-v4-flash", "DeepSeek V4 Flash", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("deepseek", "deepseek-v4-pro", "DeepSeek V4 Pro", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("deepseek", "deepseek-v4-flash-vision-exp", "DeepSeek V4 Flash Vision Exp", false, 128000, []string{"tools", "vision", "streaming", "web_search"}),
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
			DefaultBaseURL: "https://api.together.ai/v1", BaseURLEnvVar: "TOGETHER_BASE_URL", APIKeyEnvVars: []string{"TOGETHER_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "MiniMaxAI/MiniMax-M3", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("together", "MiniMaxAI/MiniMax-M3", "MiniMax M3", true, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				model("together", "moonshotai/Kimi-K2.5", "Kimi K2.5", false, 256000, []string{"tools", "streaming"}),
				model("together", "zai-org/GLM-5", "GLM-5", false, 256000, []string{"tools", "reasoning", "streaming"}),
				model("together", "openai/gpt-oss-120b", "GPT OSS 120B", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "alibaba-coding-plan", DisplayName: "Alibaba Coding Plan", Description: "Alibaba Cloud Model Studio Coding Plan OpenAI-compatible endpoint.",
			Aliases: []string{"bailian-coding", "dashscope-coding-plan", "qwen-coding-plan"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://coding-intl.dashscope.aliyuncs.com/v1", BaseURLEnvVar: "ALIBABA_CODING_PLAN_BASE_URL", APIKeyEnvVars: []string{"ALIBABA_CODING_PLAN_API_KEY", "DASHSCOPE_CODING_PLAN_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "qwen3-coder-plus", BuiltIn: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("alibaba-coding-plan", "qwen3-coder-plus", "Qwen3 Coder Plus", true, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan", "qwen3.7-plus", "Qwen3.7 Plus", false, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan", "glm-5", "GLM 5", false, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan", "kimi-k2.6", "Kimi K2.6", false, 262144, []string{"tools", "streaming"}),
				model("alibaba-coding-plan", "minimax-m2.7", "MiniMax M2.7", false, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "alibaba-coding-plan-cn", DisplayName: "Alibaba Coding Plan CN", Description: "Alibaba Cloud Model Studio China-region Coding Plan OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://coding.dashscope.aliyuncs.com/v1", BaseURLEnvVar: "ALIBABA_CODING_PLAN_CN_BASE_URL", APIKeyEnvVars: []string{"ALIBABA_CODING_PLAN_CN_API_KEY", "DASHSCOPE_CODING_PLAN_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "qwen3-coder-plus", BuiltIn: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("alibaba-coding-plan-cn", "qwen3-coder-plus", "Qwen3 Coder Plus", true, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan-cn", "qwen3.7-plus", "Qwen3.7 Plus", false, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan-cn", "glm-5", "GLM 5", false, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-coding-plan-cn", "kimi-k2.6", "Kimi K2.6", false, 262144, []string{"tools", "streaming"}),
				model("alibaba-coding-plan-cn", "minimax-m2.7", "MiniMax M2.7", false, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "alibaba", DisplayName: "Alibaba", Description: "Alibaba Cloud Model Studio DashScope OpenAI-compatible endpoint.",
			Aliases: []string{"dashscope", "qwen", "alibaba-cloud"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", BaseURLEnvVar: "ALIBABA_BASE_URL", APIKeyEnvVars: []string{"ALIBABA_API_KEY", "DASHSCOPE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "qwen3-235b-a22b", BuiltIn: true,
			NativeHostedTools:          nativeWebSearch("web_search"),
			ResponsesAPIForHostedTools: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("alibaba", "qwen3-235b-a22b", "Qwen3 235B A22B", true, 262000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba", "qwen3-max", "Qwen3 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba", "qwen3.8-max", "Qwen3.8 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba", "qwen3.7-max", "Qwen3.7 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba", "deepseek-v4-flash", "DeepSeek V4 Flash", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("alibaba", "deepseek-v4-pro", "DeepSeek V4 Pro", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("alibaba", "glm-5.2", "GLM 5.2", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
			},
		},
		{
			ID: "alibaba-cn", DisplayName: "Alibaba CN", Description: "Alibaba Cloud Model Studio China-region DashScope OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", BaseURLEnvVar: "ALIBABA_CN_BASE_URL", APIKeyEnvVars: []string{"ALIBABA_CN_API_KEY", "DASHSCOPE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "qwen3-235b-a22b", BuiltIn: true,
			NativeHostedTools:          nativeWebSearch("web_search"),
			ResponsesAPIForHostedTools: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("alibaba-cn", "qwen3-235b-a22b", "Qwen3 235B A22B", true, 262000, []string{"tools", "reasoning", "streaming"}),
				model("alibaba-cn", "qwen3-max", "Qwen3 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba-cn", "qwen3.8-max", "Qwen3.8 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba-cn", "qwen3.7-max", "Qwen3.7 Max", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("alibaba-cn", "deepseek-v4-flash", "DeepSeek V4 Flash", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("alibaba-cn", "deepseek-v4-pro", "DeepSeek V4 Pro", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("alibaba-cn", "glm-5.2", "GLM 5.2", false, 1000000, []string{"tools", "reasoning", "streaming", "web_search"}),
			},
		},
		{
			ID: "moonshotai", DisplayName: "Moonshot AI", Description: "Kimi API OpenAI-compatible endpoint.",
			Aliases: []string{"moonshot", "kimi-api"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.moonshot.ai/v1", BaseURLEnvVar: "MOONSHOTAI_BASE_URL", APIKeyEnvVars: []string{"MOONSHOT_API_KEY", "MOONSHOTAI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "kimi-k2-0905-preview", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("moonshotai", "kimi-k3", "Kimi K3", false, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("moonshotai", "kimi-k2.7-code-highspeed", "Kimi K2.7 Code Highspeed", false, 262144, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("moonshotai", "kimi-k2.6", "Kimi K2.6", false, 262144, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				model("moonshotai", "kimi-k2-0905-preview", "Kimi K2 0905 Preview", true, 262144, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "moonshotai-cn", DisplayName: "Moonshot AI CN", Description: "Kimi API China-region OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.moonshot.cn/v1", BaseURLEnvVar: "MOONSHOTAI_CN_BASE_URL", APIKeyEnvVars: []string{"MOONSHOT_CN_API_KEY", "MOONSHOT_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "kimi-k2-thinking", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("moonshotai-cn", "kimi-k3", "Kimi K3", false, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("moonshotai-cn", "kimi-k2.7-code-highspeed", "Kimi K2.7 Code Highspeed", false, 262144, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("moonshotai-cn", "kimi-k2.6", "Kimi K2.6", false, 262144, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				model("moonshotai-cn", "kimi-k2-thinking", "Kimi K2 Thinking", true, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "kimi-for-coding", DisplayName: "Kimi For Coding", Description: "Kimi coding endpoint using the Anthropic Messages-compatible API.",
			Aliases: []string{"kimi", "kimi-coding"}, Transport: TransportAnthropicMessages,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.kimi.com/coding/v1", BaseURLEnvVar: "KIMI_FOR_CODING_BASE_URL", APIKeyEnvVars: []string{"KIMI_FOR_CODING_API_KEY", "KIMI_API_KEY", "MOONSHOT_API_KEY"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "k2p6", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("kimi-for-coding", "k2p6", "Kimi For Coding", true, 262144, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "minimax", DisplayName: "MiniMax", Description: "MiniMax Anthropic Messages-compatible endpoint.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.minimax.io/anthropic/v1", BaseURLEnvVar: "MINIMAX_BASE_URL", APIKeyEnvVars: []string{"MINIMAX_API_KEY"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "MiniMax-M2", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				modelWithOutput("minimax", "MiniMax-M3", "MiniMax M3", false, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				model("minimax", "MiniMax-M2.7", "MiniMax M2.7", false, 262144, []string{"tools", "reasoning", "streaming"}),
				model("minimax", "MiniMax-M2", "MiniMax M2", true, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "minimax-cn", DisplayName: "MiniMax CN", Description: "MiniMax China-region Anthropic Messages-compatible endpoint.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.minimaxi.com/anthropic/v1", BaseURLEnvVar: "MINIMAX_CN_BASE_URL", APIKeyEnvVars: []string{"MINIMAX_CN_API_KEY", "MINIMAX_API_KEY"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "MiniMax-M2", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models: []domain.ModelInfo{
				modelWithOutput("minimax-cn", "MiniMax-M3", "MiniMax M3", false, 1000000, 0, []string{"tools", "vision", "reasoning", "streaming"}),
				model("minimax-cn", "MiniMax-M2.7", "MiniMax M2.7", false, 262144, []string{"tools", "reasoning", "streaming"}),
				model("minimax-cn", "MiniMax-M2", "MiniMax M2", true, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "minimax-coding-plan", DisplayName: "MiniMax Coding Plan", Description: "MiniMax coding plan Anthropic Messages-compatible endpoint.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.minimax.io/anthropic/v1", BaseURLEnvVar: "MINIMAX_CODING_PLAN_BASE_URL", APIKeyEnvVars: []string{"MINIMAX_CODING_PLAN_API_KEY", "MINIMAX_API_KEY"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "MiniMax-M2", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("minimax-coding-plan", "MiniMax-M2", "MiniMax M2", true, 262144, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "minimax-cn-coding-plan", DisplayName: "MiniMax CN Coding Plan", Description: "MiniMax China coding plan Anthropic Messages-compatible endpoint.",
			Transport: TransportAnthropicMessages, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.minimaxi.com/anthropic/v1", BaseURLEnvVar: "MINIMAX_CN_CODING_PLAN_BASE_URL", APIKeyEnvVars: []string{"MINIMAX_CN_CODING_PLAN_API_KEY", "MINIMAX_CN_API_KEY", "MINIMAX_API_KEY"},
			ModelFetch: ModelFetchAnthropic, DefaultModelID: "MiniMax-M2", BuiltIn: true,
			RequestProfile: anthropicDefaultRequestProfile(),
			Models:         []domain.ModelInfo{model("minimax-cn-coding-plan", "MiniMax-M2", "MiniMax M2", true, 262144, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "siliconflow", DisplayName: "SiliconFlow", Description: "SiliconFlow OpenAI-compatible inference endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.siliconflow.com/v1", BaseURLEnvVar: "SILICONFLOW_BASE_URL", APIKeyEnvVars: []string{"SILICONFLOW_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "nex-agi/DeepSeek-V3.1-Nex-N1", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("siliconflow", "nex-agi/DeepSeek-V3.1-Nex-N1", "DeepSeek V3.1 Nex N1", true, 262000, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "siliconflow-cn", DisplayName: "SiliconFlow CN", Description: "SiliconFlow China-region OpenAI-compatible inference endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.siliconflow.cn/v1", BaseURLEnvVar: "SILICONFLOW_CN_BASE_URL", APIKeyEnvVars: []string{"SILICONFLOW_CN_API_KEY", "SILICONFLOW_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "Kwaipilot/KAT-Dev", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("siliconflow-cn", "Kwaipilot/KAT-Dev", "KAT Dev", true, 262000, []string{"tools", "streaming"})},
		},
		{
			ID: "stepfun", DisplayName: "StepFun", Description: "StepFun OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.stepfun.com/v1", BaseURLEnvVar: "STEPFUN_BASE_URL", APIKeyEnvVars: []string{"STEPFUN_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "step-3.5-flash-2603", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("stepfun", "step-3.5-flash-2603", "Step 3.5 Flash", true, 128000, []string{"tools", "streaming"})},
		},
		{
			ID: "cloudflare-workers-ai", DisplayName: "Cloudflare Workers AI", Description: "Cloudflare Workers AI OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1", BaseURLEnvVar: "CLOUDFLARE_WORKERS_AI_BASE_URL", APIKeyEnvVars: []string{"CLOUDFLARE_API_KEY", "CLOUDFLARE_WORKERS_AI_API_KEY"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "@cf/zai-org/glm-4.7-flash", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("cloudflare-workers-ai", "@cf/zai-org/glm-4.7-flash", "GLM 4.7 Flash", true, 128000, []string{"tools", "streaming"}),
				model("cloudflare-workers-ai", "@cf/openai/gpt-oss-120b", "GPT OSS 120B", false, 131072, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "huggingface", DisplayName: "Hugging Face", Description: "Hugging Face Inference Providers OpenAI-compatible router.",
			Aliases: []string{"hf", "hugging-face"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://router.huggingface.co/v1", BaseURLEnvVar: "HUGGINGFACE_BASE_URL", APIKeyEnvVars: []string{"HF_TOKEN", "HUGGINGFACE_API_KEY"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "Qwen/Qwen3.5-397B-A17B", BuiltIn: true,
			DefaultResponsesAPI: false,
			RequestProfile:      domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("huggingface", "Qwen/Qwen3.5-397B-A17B", "Qwen3.5 397B A17B", true, 262000, []string{"tools", "reasoning", "streaming"}),
				model("huggingface", "openai/gpt-oss-120b", "GPT OSS 120B", false, 131072, []string{"tools", "reasoning", "streaming", "remote_mcp"}),
			},
		},
		{
			ID: "friendli", DisplayName: "Friendli", Description: "FriendliAI serverless OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.friendli.ai/serverless/v1", BaseURLEnvVar: "FRIENDLI_BASE_URL", APIKeyEnvVars: []string{"FRIENDLI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "Qwen/Qwen3-235B-A22B-Instruct-2507", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("friendli", "Qwen/Qwen3-235B-A22B-Instruct-2507", "Qwen3 235B A22B Instruct", true, 262000, []string{"tools", "streaming"})},
		},
		{
			ID: "novita-ai", DisplayName: "Novita AI", Description: "Novita AI OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.novita.ai/openai", BaseURLEnvVar: "NOVITA_AI_BASE_URL", APIKeyEnvVars: []string{"NOVITA_API_KEY", "NOVITA_AI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "deepseek/deepseek-r1-turbo", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("novita-ai", "deepseek/deepseek-r1-turbo", "DeepSeek R1 Turbo", true, 128000, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "nvidia", DisplayName: "NVIDIA", Description: "NVIDIA NIM OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://integrate.api.nvidia.com/v1", BaseURLEnvVar: "NVIDIA_BASE_URL", APIKeyEnvVars: []string{"NVIDIA_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "upstage/solar-10_7b-instruct", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("nvidia", "upstage/solar-10_7b-instruct", "Solar 10.7B Instruct", true, 128000, []string{"tools", "streaming"})},
		},
		{
			ID: "zai", DisplayName: "Z.ai", Description: "Z.AI GLM OpenAI-compatible endpoint.",
			Aliases: []string{"zhipu", "glm", "z-ai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.z.ai/api/paas/v4", BaseURLEnvVar: "ZAI_BASE_URL", APIKeyEnvVars: []string{"ZAI_API_KEY", "Z_AI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "glm-5v-turbo", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("zai", "glm-5.3", "GLM 5.3", false, 1000000, 0, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("zai", "glm-5.3-flash", "GLM 5.3 Flash", false, 1000000, 0, []string{"tools", "reasoning", "streaming"}),
				model("zai", "glm-5v-turbo", "GLM 5V Turbo", true, 128000, []string{"tools", "vision", "streaming"}),
			},
		},
		{
			ID: "zai-coding-plan", DisplayName: "Z.ai Coding Plan", Description: "Z.AI GLM Coding Plan OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.z.ai/api/coding/paas/v4", ResponsesBaseURL: "https://api.z.ai/api/v1",
			BaseURLEnvVar: "ZAI_CODING_PLAN_BASE_URL", APIKeyEnvVars: []string{"ZAI_CODING_PLAN_API_KEY", "ZAI_API_KEY", "Z_AI_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "glm-4.7", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("zai-coding-plan", "glm-4.7", "GLM 4.7", true, 200000, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "zhipuai", DisplayName: "Zhipu AI", Description: "Zhipu AI BigModel OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4", BaseURLEnvVar: "ZHIPUAI_BASE_URL", APIKeyEnvVars: []string{"ZHIPUAI_API_KEY", "ZHIPU_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "glm-5v-turbo", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("zhipuai", "glm-5v-turbo", "GLM 5V Turbo", true, 128000, []string{"tools", "vision", "streaming"})},
		},
		{
			ID: "ollama-cloud", DisplayName: "Ollama Cloud", Description: "Ollama OpenAI-compatible cloud endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://ollama.com/v1", BaseURLEnvVar: "OLLAMA_CLOUD_BASE_URL", APIKeyEnvVars: []string{"OLLAMA_API_KEY", "OLLAMA_CLOUD_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "minimax-m2.7", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("ollama-cloud", "minimax-m2.7", "MiniMax M2.7", true, 262144, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "lmstudio", DisplayName: "LM Studio", Description: "Local LM Studio OpenAI-compatible server.",
			Aliases: []string{"lm-studio"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthNone, AuthAPIKey}, DefaultAuthType: AuthNone,
			DefaultBaseURL: "http://127.0.0.1:1234/v1", BaseURLEnvVar: "LMSTUDIO_BASE_URL", APIKeyEnvVars: []string{"LMSTUDIO_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-oss-20b", BuiltIn: true,
			DefaultResponsesAPI: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models:         []domain.ModelInfo{model("lmstudio", "openai/gpt-oss-20b", "GPT OSS 20B", true, 131072, []string{"tools", "reasoning", "streaming"})},
		},
		{
			ID: "inference", DisplayName: "Inference.net", Description: "Inference.net OpenAI-compatible serverless and gateway API.",
			Aliases: []string{"inference-net"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.inference.net/v1", BaseURLEnvVar: "INFERENCE_BASE_URL", APIKeyEnvVars: []string{"INFERENCE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "glm-5.2", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("inference", "glm-5.2", "GLM 5.2", true, 256000, []string{"tools", "reasoning", "streaming"}),
				model("inference", "claude-haiku-4-5", "Claude Haiku 4.5", false, 200000, []string{"tools", "streaming"}),
				model("inference", "gpt-5-mini", "GPT-5 Mini", false, 400000, []string{"tools", "reasoning", "streaming"}),
				model("inference", "gemini-3.5-flash", "Gemini 3.5 Flash", false, 1000000, []string{"tools", "vision", "reasoning", "streaming"}),
			},
		},
		{
			ID: "tencent-coding-plan", DisplayName: "Tencent Coding Plan", Description: "Tencent Cloud LKEAP Coding Plan OpenAI-compatible endpoint.",
			Aliases: []string{"tencent-tokenhub", "tencent"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.lkeap.cloud.tencent.com/coding/v3", BaseURLEnvVar: "TENCENT_CODING_PLAN_BASE_URL", APIKeyEnvVars: []string{"TENCENT_CODING_PLAN_API_KEY", "TENCENT_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "kimi-k2.6", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("tencent-coding-plan", "kimi-k2.6", "Kimi K2.6", true, 262144, []string{"tools", "reasoning", "streaming"}),
				model("tencent-coding-plan", "minimax-m2.7", "MiniMax M2.7", false, 262144, []string{"tools", "reasoning", "streaming"}),
				model("tencent-coding-plan", "minimax-m3", "MiniMax M3", false, 1000000, []string{"tools", "vision", "reasoning", "streaming"}),
				model("tencent-coding-plan", "deepseek-v4-flash", "DeepSeek V4 Flash", false, 128000, []string{"tools", "streaming"}),
				model("tencent-coding-plan", "deepseek-v4-pro", "DeepSeek V4 Pro", false, 128000, []string{"tools", "reasoning", "streaming"}),
				model("tencent-coding-plan", "glm-5.3", "GLM 5.3", false, 1000000, []string{"tools", "reasoning", "streaming"}),
				model("tencent-coding-plan", "hy4-preview", "Hy4 Preview", false, 1000000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "scaleway", DisplayName: "Scaleway", Description: "Scaleway Generative APIs OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.scaleway.ai/v1", BaseURLEnvVar: "SCALEWAY_BASE_URL", APIKeyEnvVars: []string{"SCW_SECRET_KEY", "SCALEWAY_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "qwen3.5-397b-a17b", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("scaleway", "qwen3.5-397b-a17b", "Qwen3.5 397B A17B", true, 262000, []string{"tools", "reasoning", "streaming"}),
				model("scaleway", "llama-3.1-70b-instruct", "Llama 3.1 70B Instruct", false, 128000, []string{"tools", "streaming"}),
				model("scaleway", "mistral-small-3.2-24b-instruct-2506", "Mistral Small 3.2 24B", false, 128000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "stackit", DisplayName: "STACKIT", Description: "STACKIT AI Model Serving OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1", BaseURLEnvVar: "STACKIT_BASE_URL", APIKeyEnvVars: []string{"STACKIT_API_KEY", "STACKIT_AUTH_TOKEN"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "Qwen/Qwen3.6-27B", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("stackit", "Qwen/Qwen3.6-27B", "Qwen3.6 27B", true, 262000, 16384, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("stackit", "openai/gpt-oss-120b", "GPT OSS 120B", false, 131000, 8192, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("stackit", "google/gemma-4-31B-it", "Gemma 4 31B", false, 256000, 4096, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("stackit", "cortecs/Llama-3.3-70B-Instruct-FP8-Dynamic", "Llama 3.3 70B Instruct", false, 128000, 4096, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "vultr", DisplayName: "Vultr", Description: "Vultr Serverless Inference OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.vultrinference.com/v1", BaseURLEnvVar: "VULTR_BASE_URL", APIKeyEnvVars: []string{"VULTR_INFERENCE_API_KEY", "VULTR_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "kimi-k2-instruct", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("vultr", "kimi-k2-instruct", "Kimi K2 Instruct", true, 128000, []string{"tools", "streaming"}),
				model("vultr", "MiniMax-M2.5", "MiniMax M2.5", false, 262144, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "digitalocean", DisplayName: "DigitalOcean", Description: "DigitalOcean GradientAI Serverless Inference OpenAI-compatible endpoint.",
			Aliases: []string{"do-gradient", "digitalocean-inference"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://inference.do-ai.run/v1", BaseURLEnvVar: "DIGITALOCEAN_BASE_URL", APIKeyEnvVars: []string{"DIGITALOCEAN_TOKEN", "MODEL_ACCESS_KEY", "DIGITALOCEAN_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai-gpt-4o-mini", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("digitalocean", "openai-gpt-4.1", "GPT-4.1", false, 1047576, 32768, []string{"tools", "vision", "streaming"}),
				modelWithOutput("digitalocean", "openai-gpt-4o", "GPT-4o", false, 128000, 16384, []string{"tools", "vision", "streaming"}),
				modelWithOutput("digitalocean", "openai-gpt-4o-mini", "GPT-4o mini", true, 128000, 16384, []string{"tools", "vision", "streaming"}),
				model("digitalocean", "anthropic-claude-4.6-sonnet", "Claude Sonnet 4.6", false, 200000, []string{"tools", "reasoning", "streaming"}),
				model("digitalocean", "kimi-k3", "Kimi K3", false, 1000000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "ovhcloud", DisplayName: "OVHcloud", Description: "OVHcloud AI Endpoints OpenAI-compatible endpoint.",
			Aliases: []string{"ovh", "ovh-ai-endpoints"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://oai.endpoints.kepler.ai.cloud.ovh.net/v1", BaseURLEnvVar: "OVHCLOUD_BASE_URL", APIKeyEnvVars: []string{"OVHCLOUD_API_KEY", "OVH_AI_ENDPOINTS_ACCESS_TOKEN", "AI_ENDPOINT_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-oss-20b", BuiltIn: true,
			DefaultResponsesAPI: true,
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"store": false, "stream": true}},
			Models: []domain.ModelInfo{
				model("ovhcloud", "gpt-oss-20b", "GPT OSS 20B", true, 131000, []string{"tools", "reasoning", "streaming"}),
				model("ovhcloud", "Meta-Llama-3_3-70B-Instruct", "Llama 3.3 70B Instruct", false, 128000, []string{"tools", "streaming"}),
				model("ovhcloud", "Mistral-Small-3_2-24B-Instruct-2506", "Mistral Small 3.2 24B", false, 128000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "helicone", DisplayName: "Helicone", Description: "Helicone AI Gateway OpenAI-compatible routing provider.",
			Aliases: []string{"helicone-ai-gateway"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://ai-gateway.helicone.ai/v1", BaseURLEnvVar: "HELICONE_BASE_URL", APIKeyEnvVars: []string{"HELICONE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-4o-mini", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("helicone", "gpt-4o-mini", "GPT-4o mini", true, 128000, []string{"tools", "vision", "streaming"}),
				model("helicone", "claude-sonnet-4", "Claude Sonnet 4", false, 200000, []string{"tools", "reasoning", "streaming"}),
				model("helicone", "llama-3.3-70b", "Llama 3.3 70B", false, 128000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "clarifai", DisplayName: "Clarifai", Description: "Clarifai OpenAI-compatible inference endpoint.",
			Aliases: []string{"clarifai-openai"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.clarifai.com/v2/ext/openai/v1", BaseURLEnvVar: "CLARIFAI_BASE_URL", APIKeyEnvVars: []string{"CLARIFAI_API_KEY", "CLARIFAI_PAT"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "arcee_ai/AFM/models/trinity-mini", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("clarifai", "arcee_ai/AFM/models/trinity-mini", "Trinity Mini", true, 128000, []string{"tools", "streaming"}),
				model("clarifai", "https://clarifai.com/qwen/qwenLM/models/Qwen3-30B-A3B-Instruct-2507", "Qwen3 30B Instruct", false, 128000, []string{"tools", "streaming"}),
				model("clarifai", "https://clarifai.com/qwen/qwenLM/models/Qwen3-30B-A3B-Thinking-2507", "Qwen3 30B Thinking", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "cloudferro-sherlock", DisplayName: "CloudFerro Sherlock", Description: "CloudFerro Sherlock AI OpenAI-compatible endpoint.",
			Aliases: []string{"cloudferro", "sherlock"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api-sherlock.cloudferro.com/openai/v1", BaseURLEnvVar: "CLOUDFERRO_SHERLOCK_BASE_URL", APIKeyEnvVars: []string{"CLOUDFERRO_SHERLOCK_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "meta-llama/Llama-3.3-70B-Instruct", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("cloudferro-sherlock", "meta-llama/Llama-3.3-70B-Instruct", "Llama 3.3 70B Instruct", true, 128000, []string{"streaming"}),
				model("cloudferro-sherlock", "Qwen/Qwen3-32B", "Qwen3 32B", false, 128000, []string{"streaming"}),
			},
		},
		{
			ID: "upstage", DisplayName: "Upstage", Description: "Upstage Solar OpenAI-compatible endpoint.",
			Aliases: []string{"upstage-solar"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.upstage.ai/v1/solar", BaseURLEnvVar: "UPSTAGE_BASE_URL", APIKeyEnvVars: []string{"UPSTAGE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "solar-pro4", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("upstage", "solar-pro4", "Solar Pro 4", true, 200000, []string{"tools", "reasoning", "streaming"}),
				model("upstage", "solar-pro3", "Solar Pro 3", false, 200000, []string{"reasoning", "streaming"}),
				model("upstage", "solar-pro2", "Solar Pro 2", false, 128000, []string{"tools", "reasoning", "streaming"}),
				model("upstage", "solar-mini", "Solar Mini", false, 128000, []string{"reasoning", "streaming"}),
			},
		},
		{
			ID: "poe", DisplayName: "Poe", Description: "Poe OpenAI-compatible gateway for hosted models and bots.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.poe.com/v1", BaseURLEnvVar: "POE_BASE_URL", APIKeyEnvVars: []string{"POE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "GPT-5.4", BuiltIn: true,
			NativeHostedTools:          nativeWebSearch("web_search_preview"),
			ResponsesAPIForHostedTools: true,
			RequestProfile:             domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("poe", "GPT-5.4", "GPT-5.4", true, 400000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("poe", "Claude-Sonnet-4.6", "Claude Sonnet 4.6", false, 200000, []string{"tools", "reasoning", "streaming"}),
				model("poe", "Grok-4.20-Multi-Agent", "Grok 4.20 Multi-Agent", false, 1000000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "requesty", DisplayName: "Requesty", Description: "Requesty OpenAI-compatible routing gateway.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://router.requesty.ai/v1", BaseURLEnvVar: "REQUESTY_BASE_URL", APIKeyEnvVars: []string{"REQUESTY_API_KEY"},
			ModelFetch:        ModelFetchOpenAICompatible,
			DefaultModelID:    "anthropic/claude-sonnet-4-20250514",
			BuiltIn:           true,
			NativeHostedTools: nativeWebSearch("web_search"),
			RequestProfile: domain.ProviderRequestProfile{
				Headers: map[string]string{"HTTP-Referer": "https://aivo.local", "X-Title": "Aivo"},
				Params:  map[string]any{"stream": true},
			},
			Models: []domain.ModelInfo{
				model("requesty", "openai/gpt-4o", "GPT-4o", false, 128000, []string{"tools", "vision", "streaming"}),
				model("requesty", "openai-responses/gpt-4.1", "GPT-4.1 Responses", false, 1000000, []string{"tools", "vision", "streaming", "web_search"}),
				model("requesty", "anthropic/claude-sonnet-4-20250514", "Claude Sonnet 4", true, 200000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("requesty", "xai/grok-3", "Grok 3", false, 131000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("requesty", "perplexity/sonar-pro", "Sonar Pro", false, 200000, []string{"tools", "search", "web_search", "streaming"}),
			},
		},
		{
			ID: "nebius", DisplayName: "Nebius", Description: "Nebius Token Factory OpenAI-compatible endpoint.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.tokenfactory.nebius.com/v1", BaseURLEnvVar: "NEBIUS_BASE_URL", APIKeyEnvVars: []string{"NEBIUS_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "moonshotai/Kimi-K2.5", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("nebius", "moonshotai/Kimi-K2.5", "Kimi K2.5", true, 256000, []string{"tools", "streaming"}),
				model("nebius", "meta-llama/Meta-Llama-3.1-8B-Instruct-fast", "Llama 3.1 8B Instruct Fast", false, 128000, []string{"tools", "streaming"}),
				model("nebius", "deepseek-ai/DeepSeek-R1-0528", "DeepSeek R1 0528", false, 128000, []string{"tools", "reasoning", "streaming"}),
			},
		},
		{
			ID: "wandb", DisplayName: "Weights & Biases", Description: "W&B Serverless Inference OpenAI-compatible endpoint.",
			Aliases: []string{"weights-and-biases", "wandb-inference"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.inference.wandb.ai/v1", BaseURLEnvVar: "WANDB_BASE_URL", APIKeyEnvVars: []string{"WANDB_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-oss-20b", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("wandb", "openai/gpt-oss-20b", "GPT OSS 20B", true, 131000, []string{"tools", "reasoning", "streaming"}),
				modelWithOutput("wandb", "Qwen/Qwen3.6-27B", "Qwen3.6 27B", false, 262000, 16384, []string{"tools", "vision", "reasoning", "streaming"}),
				model("wandb", "meta-llama/Llama-3.1-8B-Instruct", "Llama 3.1 8B Instruct", false, 128000, []string{"tools", "streaming"}),
			},
		},
		{
			ID: "venice", DisplayName: "Venice", Description: "Venice OpenAI-compatible endpoint with Venice-specific search parameters.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.venice.ai/api/v1", BaseURLEnvVar: "VENICE_BASE_URL", APIKeyEnvVars: []string{"VENICE_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "zai-org-glm-5", BuiltIn: true,
			NativeHostedTools: nativeWebSearch("venice_web_search"),
			RequestProfile:    domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("venice", "zai-org-glm-5", "GLM 5", true, 128000, []string{"tools", "reasoning", "streaming", "web_search"}),
				model("venice", "venice-uncensored", "Venice Uncensored", false, 128000, []string{"tools", "streaming", "web_search"}),
				model("venice", "default", "Default", false, 128000, []string{"tools", "streaming", "web_search"}),
			},
		},
		{
			ID: "vivgrid", DisplayName: "Vivgrid", Description: "Vivgrid OpenAI-compatible Model API for coding agents.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.vivgrid.com/v1", BaseURLEnvVar: "VIVGRID_BASE_URL", APIKeyEnvVars: []string{"VIVGRID_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "gpt-5.6-terra", BuiltIn: true,
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("vivgrid", "gpt-5.6-terra", "GPT-5.6 Terra", true, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("vivgrid", "gpt-5.6-sol", "GPT-5.6 Sol", false, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("vivgrid", "gpt-5.6-luna", "GPT-5.6 Luna", false, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming"}),
				modelWithOutput("vivgrid", "gpt-5.5", "GPT-5.5", false, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming"}),
			},
		},
		{
			ID: "perplexity", DisplayName: "Perplexity", Description: "Perplexity Sonar search-grounded chat models.",
			Aliases: []string{"pplx"}, Transport: TransportOpenAICompatible,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.perplexity.ai", BaseURLEnvVar: "PERPLEXITY_BASE_URL", APIKeyEnvVars: []string{"PERPLEXITY_API_KEY"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "sonar-pro", BuiltIn: true,
			NativeHostedTools: nativeWebSearch("perplexity_search", "web_search", "search"),
			RequestProfile:    domain.ProviderRequestProfile{Params: map[string]any{"stream": true}},
			Models: []domain.ModelInfo{
				model("perplexity", "sonar-pro", "Sonar Pro", true, 200000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar", "Sonar", false, 127000, []string{"tools", "search", "web_search", "streaming"}),
				model("perplexity", "sonar-reasoning-pro", "Sonar Reasoning Pro", false, 128000, []string{"tools", "search", "web_search", "reasoning", "streaming"}),
				model("perplexity", "sonar-deep-research", "Sonar Deep Research", false, 128000, []string{"search", "web_search", "reasoning", "streaming"}),
			},
		},
		{
			ID: "perplexity-agent", DisplayName: "Perplexity Agent", Description: "Perplexity Agent API through the OpenAI-compatible Responses endpoint.",
			Aliases: []string{"perplexity-router-agent"}, Transport: TransportOpenAIResponses,
			AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://api.perplexity.ai/v1", BaseURLEnvVar: "PERPLEXITY_AGENT_BASE_URL", APIKeyEnvVars: []string{"PERPLEXITY_API_KEY", "PERPLEXITY_AGENT_API_KEY"},
			ModelFetch: ModelFetchOpenAICompatible, DefaultModelID: "openai/gpt-5.6-terra", BuiltIn: true,
			NativeHostedTools: nativeWebSearch("web_search"),
			ResponsesDefaults: ProviderResponsesDefaults{
				DisableEncryptedReasoningInclude: true,
				DisableReasoningSummary:          true,
			},
			RequestProfile: domain.ProviderRequestProfile{Params: map[string]any{"store": false, "stream": true}},
			Models: []domain.ModelInfo{
				modelWithOutput("perplexity-agent", "openai/gpt-5.6-terra", "GPT-5.6 Terra", true, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming", "web_search"}),
				modelWithOutput("perplexity-agent", "openai/gpt-5.6-sol", "GPT-5.6 Sol", false, 1050000, 128000, []string{"tools", "vision", "reasoning", "streaming", "web_search"}),
				model("perplexity-agent", "preset/low", "Low preset", false, 0, []string{"tools", "reasoning", "streaming", "web_search"}),
			},
		},
		{
			ID: "github-models", DisplayName: "GitHub Models", Description: "Retired GitHub Models inference API.",
			Transport: TransportOpenAICompatible, AuthTypes: []AuthType{AuthAPIKey}, DefaultAuthType: AuthAPIKey,
			DefaultBaseURL: "https://models.github.ai/inference", BaseURLEnvVar: "GITHUB_MODELS_BASE_URL", APIKeyEnvVars: []string{"GITHUB_MODELS_API_KEY", "GITHUB_TOKEN"},
			ModelFetch: ModelFetchStatic, DefaultModelID: "github-models-retired", BuiltIn: true, Deprecated: true,
			Models: []domain.ModelInfo{{
				ID: "github-models-retired", ProviderID: "github-models", Name: "GitHub Models retired",
				Recommended: true, Deprecated: true, Status: "deprecated", Modalities: []string{"text"},
			}},
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
	defs = appendProviderTemplateDefinitions(defs)
	return defs
}

func appendProviderTemplateDefinitions(defs []ProviderDefinition) []ProviderDefinition {
	known := map[string]bool{}
	for _, def := range defs {
		id := normalizeProviderKey(def.ID)
		if id == "" {
			continue
		}
		known[id] = true
		for _, alias := range def.Aliases {
			if normalized := normalizeProviderKey(alias); normalized != "" {
				known[normalized] = true
			}
		}
	}
	for _, def := range providerTemplateDefinitions() {
		id := normalizeProviderKey(def.ID)
		if id == "" || known[id] {
			continue
		}
		if canonical, ok := builtInProviderAliases[id]; ok && known[canonical] {
			continue
		}
		defs = append(defs, def)
		known[id] = true
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
