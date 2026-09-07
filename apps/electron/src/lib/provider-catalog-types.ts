export type AuthMethod = "api-key" | "oauth-browser" | "oauth-headless";

export type ModelInfo = {
  id: string;
  providerId: string;
  name: string;
  deprecated?: boolean;
  recommended?: boolean;
  contextLength?: number;
  maxContextLength?: number;
  autoCompactTokenLimit?: number;
  capabilities?: string[];
  declaredCapabilities?: string[];
  nativeTools?: string[];
  nativeToolsKnown?: boolean;
  modalities?: string[];
  streaming?: boolean;
  toolSupport?: boolean;
  reasoningControls?: string[];
  supportedReasoningEfforts?: string[];
  defaultReasoningEffort?: string;
  supportsVerbosity?: boolean;
  defaultVerbosity?: string;
  serviceTiers?: string[];
  defaultServiceTier?: string;
  supportsParallelToolCalls?: boolean;
  webSearchToolType?: string;
  webSearchToolTypeKnown?: boolean;
  useResponsesLite?: boolean;
  supportsImageDetailOriginal?: boolean;
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
