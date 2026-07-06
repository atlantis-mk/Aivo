export type AuthMethod = "api-key" | "oauth-browser" | "oauth-headless";

export type ModelInfo = {
  id: string;
  providerId: string;
  name: string;
  deprecated?: boolean;
  recommended?: boolean;
  contextLength?: number;
  capabilities?: string[];
  modalities?: string[];
  streaming?: boolean;
  toolSupport?: boolean;
  reasoningControls?: string[];
  pricing?: Record<string, number>;
  lastRefreshed?: string;
};

export type ProviderAuthMethod = {
  id: string;
  label: string;
  stable: boolean;
  experimental?: boolean;
  available: boolean;
  description?: string;
};

export type AuthInfo = {
  type: string;
  connected: boolean;
  source: string;
  environment?: string;
  lastValidatedAt?: string;
  connectedAt?: string;
};

export type ProviderAccountInfo = {
  id: string;
  providerId: string;
  method: string;
  accountId: string;
  displayName: string;
  connectedAt?: string;
};

export type ProviderInfo = {
  id: string;
  name: string;
  type: string;
  baseUrl?: string;
  builtIn: boolean;
  custom: boolean;
  experimental?: boolean;
  connected: boolean;
  connectionSource?: string;
  environment?: string;
  defaultModelId?: string;
  models: ModelInfo[];
  authMethods: ProviderAuthMethod[];
  auth?: AuthInfo;
  accounts?: ProviderAccountInfo[];
  readiness?: {
    ready: boolean;
    authMode?: string;
    source?: string;
    environment?: string;
    reason?: string;
  };
  modelRefresh?: {
    strategy: string;
    status: string;
    lastRefresh?: string;
    error?: string;
    modelCount: number;
    refreshable: boolean;
    cacheSource?: string;
    parserType?: string;
    endpoint?: string;
    stale?: boolean;
  };
  profile?: {
    id: string;
    displayName: string;
    providerType: string;
    modelEndpoint?: string;
    modelFetch?: string;
    parserType?: string;
    cacheTtlSeconds?: number;
    paginated?: boolean;
    messageShape?: string;
    supportedExtras?: string[];
  };
};

export type ModelRef = {
  providerId: string;
  modelId: string;
};

export type BrowserAuthSessionInfo = {
  id: string;
  providerId: string;
  method: string;
  status: string;
  authUrl?: string;
  callbackUrl?: string;
  instructions?: string;
  error?: string;
  expiresAt: string;
};

export type CatalogState = {
  providers: ProviderInfo[];
  models: ModelInfo[];
  connected: string[];
  defaultModel?: ModelRef;
  connectedProviders?: ProviderInfo[];
  popularProviders?: ProviderInfo[];
  customProviders?: ProviderInfo[];
  pendingAuth?: BrowserAuthSessionInfo;
};

export type ProviderConnectInput = {
  providerId: string;
  name?: string;
  type?: string;
  baseUrl?: string;
  apiKey?: string;
  apiKeyEnv?: string;
  modelId?: string;
  method?: string;
  headers?: Record<string, string>;
};

export function fallbackCatalogState(): CatalogState {
  return {
    providers: [
      {
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
          { id: "oauth-browser", label: "ChatGPT Pro/Plus (browser)", stable: false, experimental: true, available: true, description: "OpenAI browser OAuth with PKCE and localhost callback" },
          { id: "oauth-headless", label: "ChatGPT Pro/Plus (headless)", stable: false, experimental: true, available: true, description: "OpenAI device authorization flow" },
          { id: "api-key", label: "API Key", stable: true, available: true },
        ],
      },
      {
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
      },
      {
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
      },
      {
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
      },
      {
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
      },
    ],
    models: [],
    connected: [],
    connectedProviders: [],
    popularProviders: [
      {
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
          { id: "oauth-browser", label: "ChatGPT Pro/Plus (browser)", stable: false, experimental: true, available: true, description: "OpenAI browser OAuth with PKCE and localhost callback" },
          { id: "oauth-headless", label: "ChatGPT Pro/Plus (headless)", stable: false, experimental: true, available: true, description: "OpenAI device authorization flow" },
          { id: "api-key", label: "API Key", stable: true, available: true },
        ],
      },
      {
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
      },
      {
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
      },
      {
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
      },
    ],
    customProviders: [],
  };
}

export function providerTypeLabel(type?: string) {
  switch (type) {
    case "openai-compatible":
      return "OpenAI Compatible";
    case "openai":
      return "OpenAI";
    case "anthropic":
      return "Anthropic";
    case "google":
      return "Google Gemini";
    case "openrouter":
      return "OpenRouter";
    case "claude-code":
      return "Claude Code";
    case "gemini":
      return "Gemini";
    default:
      return type || "Unspecified";
  }
}

export function providerConnectionLabel(provider: ProviderInfo) {
  switch (provider.connectionSource) {
    case "api-key":
      return "API key";
    case "env":
      return provider.environment ? `Env: ${provider.environment}` : "Environment";
    case "oauth-browser":
      return "Browser auth";
    case "oauth":
      return "OAuth";
    case "oauth-headless":
      return "Headless auth";
    default:
      return provider.connected ? "Connected" : "Not connected";
  }
}
