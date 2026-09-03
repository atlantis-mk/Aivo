import type { ProviderInfo } from "@/lib/provider-catalog-types";

function openAIProvider(): ProviderInfo {
  return {
    id: "openai",
    name: "OpenAI",
    type: "openai",
    baseUrl: "https://api.openai.com/v1",
    builtIn: true,
    custom: false,
    connected: false,
    environment: "OPENAI_API_KEY",
    defaultModelId: "gpt-5.5",
    models: [
      { id: "gpt-5.5", providerId: "openai", name: "GPT-5.5", recommended: true },
      { id: "gpt-5.4", providerId: "openai", name: "GPT-5.4" },
      { id: "gpt-5.4-mini", providerId: "openai", name: "GPT-5.4-Mini" },
      { id: "gpt-5.3-codex-spark", providerId: "openai", name: "GPT-5.3-Codex-Spark" },
    ],
    authMethods: [
      {
        id: "oauth-browser",
        label: "ChatGPT Pro/Plus (browser)",
        stable: false,
        experimental: true,
        available: true,
        description: "OpenAI browser OAuth with PKCE and localhost callback",
      },
      {
        id: "oauth-headless",
        label: "ChatGPT Pro/Plus (headless)",
        stable: false,
        experimental: true,
        available: true,
        description: "OpenAI device authorization flow",
      },
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

function claudeCodeProvider(): ProviderInfo {
  return {
    id: "claude-code",
    name: "Claude Code",
    type: "anthropic",
    builtIn: true,
    custom: false,
    connected: false,
    environment: "ANTHROPIC_API_KEY",
    defaultModelId: "claude-sonnet-4",
    models: [
      { id: "claude-sonnet-4", providerId: "claude-code", name: "Claude Sonnet 4", recommended: true },
    ],
    authMethods: [
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

function geminiProvider(): ProviderInfo {
  return {
    id: "gemini",
    name: "Gemini",
    type: "google",
    builtIn: true,
    custom: false,
    connected: false,
    environment: "GEMINI_API_KEY",
    defaultModelId: "gemini-2.5-pro",
    models: [
      { id: "gemini-2.5-pro", providerId: "gemini", name: "Gemini 2.5 Pro", recommended: true },
      { id: "gemini-2.5-flash", providerId: "gemini", name: "Gemini 2.5 Flash" },
    ],
    authMethods: [
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

function volcengineAgentPlanProvider(): ProviderInfo {
  return {
    id: "volcengine-agent-plan",
    name: "火山方舟 Agent Plan",
    type: "openai",
    baseUrl: "https://ark.cn-beijing.volces.com/api/plan/v3",
    builtIn: true,
    custom: false,
    connected: false,
    environment: "ARK_API_KEY",
    defaultModelId: "ark-code-latest",
    models: [
      {
        id: "ark-code-latest",
        providerId: "volcengine-agent-plan",
        name: "ark-code-latest（自动路由）",
        recommended: true,
      },
      {
        id: "doubao-seed-2.0-lite",
        providerId: "volcengine-agent-plan",
        name: "doubao-seed-2.0-lite",
      },
      {
        id: "doubao-seed-2.0-mini",
        providerId: "volcengine-agent-plan",
        name: "doubao-seed-2.0-mini",
      },
      {
        id: "kimi-k2.7-code",
        providerId: "volcengine-agent-plan",
        name: "kimi-k2.7-code",
      },
      {
        id: "minimax-m3",
        providerId: "volcengine-agent-plan",
        name: "minimax-m3",
      },
      {
        id: "doubao-seed-evolving",
        providerId: "volcengine-agent-plan",
        name: "doubao-seed-evolving",
      },
      {
        id: "kimi-k3",
        providerId: "volcengine-agent-plan",
        name: "kimi-k3",
      },
      {
        id: "doubao-seed-2.1-turbo",
        providerId: "volcengine-agent-plan",
        name: "doubao-seed-2.1-turbo",
      },
      {
        id: "deepseek-v4-flash",
        providerId: "volcengine-agent-plan",
        name: "deepseek-v4-flash",
      },
      {
        id: "glm-5.3",
        providerId: "volcengine-agent-plan",
        name: "glm-5.3",
      },
      {
        id: "deepseek-v4-pro",
        providerId: "volcengine-agent-plan",
        name: "deepseek-v4-pro",
      },
      {
        id: "glm-5.3-flash",
        providerId: "volcengine-agent-plan",
        name: "glm-5.3-flash",
      },
    ],
    authMethods: [
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

function openRouterProvider(): ProviderInfo {
  return {
    id: "openrouter",
    name: "OpenRouter",
    type: "openrouter",
    baseUrl: "https://openrouter.ai/api/v1",
    builtIn: true,
    custom: false,
    connected: false,
    environment: "OPENROUTER_API_KEY",
    defaultModelId: "openai/gpt-5-codex",
    models: [
      { id: "openai/gpt-5-codex", providerId: "openrouter", name: "GPT-5 Codex", recommended: true },
    ],
    authMethods: [
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

function customAPIProvider(): ProviderInfo {
  return {
    id: "custom-api",
    name: "Custom API",
    type: "openai-compatible",
    builtIn: true,
    custom: false,
    connected: false,
    models: [],
    authMethods: [
      { id: "api-key", label: "API Key", stable: true, available: true },
    ],
  };
}

export function fallbackProviders(): ProviderInfo[] {
  return [
    openAIProvider(),
    claudeCodeProvider(),
    geminiProvider(),
    volcengineAgentPlanProvider(),
    openRouterProvider(),
    customAPIProvider(),
  ];
}

export function fallbackPopularProviders(): ProviderInfo[] {
  return [
    openAIProvider(),
    claudeCodeProvider(),
    geminiProvider(),
    volcengineAgentPlanProvider(),
    openRouterProvider(),
  ];
}
