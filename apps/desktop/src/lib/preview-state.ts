import type { domain } from "../../bridge/go/models";
import {
  fallbackCatalogState,
  type BrowserAuthSessionInfo,
  type CatalogState,
  type ProviderConnectInput,
} from "@/lib/provider-catalog";

const PREVIEW_STATE_KEY = "aivo.preview.state";
const OPENAI_AUTH_ISSUER =
  (import.meta.env.VITE_AIVO_OPENAI_AUTH_ISSUER as string | undefined)?.trim() ||
  "https://auth.openai.com";
const OPENAI_AUTH_URL =
  (import.meta.env.VITE_AIVO_OPENAI_BROWSER_AUTH_URL as string | undefined)?.trim() ||
  `${OPENAI_AUTH_ISSUER}/oauth/authorize`;
const OPENAI_TOKEN_URL =
  (import.meta.env.VITE_AIVO_OPENAI_BROWSER_TOKEN_URL as string | undefined)?.trim() ||
  `${OPENAI_AUTH_ISSUER}/oauth/token`;
const OPENAI_CLIENT_ID =
  (import.meta.env.VITE_AIVO_OPENAI_BROWSER_CLIENT_ID as string | undefined)?.trim() ||
  "app_EMoamEEZ73f0CkXaXp7hrann";
const OPENAI_SCOPE =
  (import.meta.env.VITE_AIVO_OPENAI_BROWSER_SCOPES as string | undefined)?.trim() ||
  "openid profile email offline_access";

type PreviewStoredAuth = {
  id?: string;
  type: string;
  secret?: string;
  refreshSecret?: string;
  environment?: string;
  accountId?: string;
  lastValidatedAt?: string;
  connectedAt?: string;
};

type PreviewPendingAuth = BrowserAuthSessionInfo & {
  stateToken: string;
  codeVerifier: string;
};

type PreviewAppConfig = domain.AppConfig & {
  appName?: string;
  initialWorkspacePath?: string;
  defaultInitialWorkspacePath?: string;
  auxiliaryModel?: domain.ModelRef;
};

type PreviewState = {
  config?: PreviewAppConfig;
  auth?: Record<string, PreviewStoredAuth | PreviewStoredAuth[]>;
  pendingAuth?: PreviewPendingAuth | null;
};

function safeJSONParse<T>(raw: string | null, fallback: T): T {
  if (!raw) return fallback;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function readPreviewState(): PreviewState {
  if (typeof window === "undefined") return {};
  return safeJSONParse<PreviewState>(window.localStorage.getItem(PREVIEW_STATE_KEY), {});
}

function writePreviewState(nextState: PreviewState) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(PREVIEW_STATE_KEY, JSON.stringify(nextState));
}

function randomString(bytes: number) {
  const buffer = new Uint8Array(bytes);
  window.crypto.getRandomValues(buffer);
  return base64UrlEncode(buffer);
}

function base64UrlEncode(bytes: Uint8Array) {
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return window
    .btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

async function createCodeChallenge(verifier: string) {
  const data = new TextEncoder().encode(verifier);
  const digest = await window.crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(new Uint8Array(digest));
}

function currentTimestamp() {
  return new Date().toISOString();
}

function normalizePreviewConfig(config?: domain.AppConfig | null): PreviewAppConfig {
  return {
    appName: "Aivo",
    initialized: false,
    defaultInitialWorkspacePath: "~/Documents/Aivo-Workspaces",
    providers: { custom: {}, disabled: [] },
    ...(config ?? {}),
  } as unknown as PreviewAppConfig;
}

function buildPreviewCatalog(state: PreviewState): CatalogState {
  const catalog = fallbackCatalogState();
  const config = normalizePreviewConfig(state.config);
  const auth = state.auth ?? {};
  const connectedProviders = new Set<string>();

  for (const provider of catalog.providers) {
    const storedAccounts = normalizeStoredAuth(auth[provider.id]);
    const displayAccounts = storedAccounts.filter((account) => account.type !== "env");
    const stored = storedAccounts[0];
    if (!stored) continue;
    provider.connected = true;
    provider.connectionSource = stored.type;
    provider.accounts = displayAccounts.map((account) => ({
      id: account.id || `${provider.id}:${account.type}:${account.connectedAt || account.accountId || account.environment || "account"}`,
      providerId: provider.id,
      method: account.type,
      accountId: account.accountId || accountLabelForStoredAuth(account),
      displayName: account.accountId || accountLabelForStoredAuth(account),
      connectedAt: account.connectedAt,
    }));
    provider.auth = {
      type: stored.type,
      connected: true,
      source: stored.type,
      environment: stored.environment,
      connectedAt: stored.connectedAt,
      lastValidatedAt: stored.lastValidatedAt,
    };
    connectedProviders.add(provider.id);
  }

  catalog.connected = Array.from(connectedProviders);
  catalog.providers = catalog.providers.map((provider) => {
    if (provider.id === config.provider?.id && config.provider?.model) {
      provider.defaultModelId = config.provider.model;
    }
    return provider;
  });
  catalog.connectedProviders = catalog.providers.filter((provider) => provider.connected);
  catalog.popularProviders = catalog.providers.filter((provider) =>
    ["openai", "claude-code", "gemini"].includes(provider.id),
  );
  catalog.pendingAuth = state.pendingAuth ?? undefined;
  if (config.defaultModel) {
    catalog.defaultModel = config.defaultModel;
  }
  return catalog;
}

function normalizeStoredAuth(value?: PreviewStoredAuth | PreviewStoredAuth[]) {
  if (!value) return [];
  return Array.isArray(value) ? value : [value];
}

function accountLabelForStoredAuth(auth: PreviewStoredAuth) {
  if (auth.accountId?.trim()) return auth.accountId.trim();
  if (auth.secret?.trim()) {
    const secret = auth.secret.trim();
    return secret.length > 8 ? `...${secret.slice(-6)}` : "API Key";
  }
  if (auth.environment?.trim()) return auth.environment.trim();
  return "默认账号";
}

function persistPreviewProvider(input: ProviderConnectInput, authType: string, authSecret?: PreviewStoredAuth) {
  const state = readPreviewState();
  const config = normalizePreviewConfig(state.config);
  const providerID =
    input.providerId === "custom-openai-compatible"
      ? (input.name?.trim() || "custom-provider")
      : input.providerId.trim();

  const nextAuth: PreviewStoredAuth = {
    id: window.crypto.randomUUID(),
    type: authType,
    secret: authSecret?.secret ?? input.apiKey?.trim(),
    refreshSecret: authSecret?.refreshSecret,
    environment: authSecret?.environment ?? input.apiKeyEnv?.trim(),
    connectedAt: authSecret?.connectedAt ?? currentTimestamp(),
    lastValidatedAt: authSecret?.lastValidatedAt ?? currentTimestamp(),
  };
  nextAuth.accountId = authSecret?.accountId ?? accountLabelForStoredAuth(nextAuth);
  const existingAuth = normalizeStoredAuth(state.auth?.[providerID]);
  state.auth = {
    ...(state.auth ?? {}),
    [providerID]: [nextAuth, ...existingAuth],
  };

  config.provider = {
    id: providerID,
    type: input.type?.trim() || (providerID === "openai" ? "openai" : "openai-compatible"),
    baseUrl: input.baseUrl?.trim(),
    apiKeyEnv: input.apiKeyEnv?.trim(),
    model: input.modelId?.trim(),
  } as domain.ProviderConfig;
  if (input.modelId?.trim()) {
    config.defaultModel = {
      providerId: providerID,
      modelId: input.modelId.trim(),
    } as domain.ModelRef;
  }

  state.config = config;
  state.pendingAuth = null;
  writePreviewState(state);
  return {
    config,
    catalog: buildPreviewCatalog(state),
  };
}

function parseAuthorizationInput(raw: string) {
  const value = raw.trim();
  if (!value) {
    return { code: "", state: "" };
  }

  try {
    const parsedURL = new URL(value);
    return {
      code: parsedURL.searchParams.get("code")?.trim() || "",
      state: parsedURL.searchParams.get("state")?.trim() || "",
    };
  } catch {
    // Ignore non-URL input.
  }

  if (value.includes("#")) {
    const [code, state] = value.split("#", 2);
    return { code: code.trim(), state: state.trim() };
  }

  if (value.includes("code=")) {
    const params = new URLSearchParams(value);
    return {
      code: params.get("code")?.trim() || "",
      state: params.get("state")?.trim() || "",
    };
  }

  return { code: value, state: "" };
}

export function getPreviewAppConfig() {
  return normalizePreviewConfig(readPreviewState().config);
}

export function getPreviewCatalog() {
  return buildPreviewCatalog(readPreviewState());
}

export function setPreviewInitialized(config: domain.AppConfig) {
  const state = readPreviewState();
  state.config = normalizePreviewConfig(config);
  writePreviewState(state);
}

export function completePreviewInitialization(initialWorkspacePath: string, appName: string) {
  const state = readPreviewState();
  const config = normalizePreviewConfig(state.config) as PreviewAppConfig;
  config.initialized = true;
  config.appName = appName.trim() || "Aivo";
  config.initialWorkspacePath = initialWorkspacePath;
  state.config = config;
  writePreviewState(state);
  return config;
}

export function connectPreviewProvider(input: ProviderConnectInput) {
  if (input.method?.trim() === "env") {
    const state = readPreviewState();
    const config = normalizePreviewConfig(state.config);
    config.provider = {
      id: input.providerId.trim(),
      type: input.type?.trim() || input.providerId.trim(),
      baseUrl: input.baseUrl?.trim(),
      apiKeyEnv: input.apiKeyEnv?.trim(),
      model: input.modelId?.trim(),
    } as domain.ProviderConfig;
    if (input.modelId?.trim()) {
      config.defaultModel = { providerId: input.providerId.trim(), modelId: input.modelId.trim() } as domain.ModelRef;
    }
    state.config = config;
    writePreviewState(state);
    return { config, catalog: buildPreviewCatalog(state) };
  }
  return persistPreviewProvider(input, input.method?.trim() || "api-key");
}

export function deletePreviewProviderAccount(accountId: string) {
  const state = readPreviewState();
  const auth = state.auth ?? {};
  for (const [providerId, value] of Object.entries(auth)) {
    const accounts = normalizeStoredAuth(value);
    const nextAccounts = accounts.filter((account) => account.id !== accountId);
    if (nextAccounts.length === accounts.length) continue;
    if (nextAccounts.length === 0) {
      delete auth[providerId];
    } else {
      auth[providerId] = nextAccounts;
    }
    break;
  }
  state.auth = auth;
  writePreviewState(state);
  return buildPreviewCatalog(state);
}

export function deletePreviewProvider(providerId: string) {
  const normalizedProviderId = providerId.trim();
  const state = readPreviewState();
  const config = normalizePreviewConfig(state.config);
  const auth = state.auth ?? {};
  delete auth[normalizedProviderId];
  state.auth = auth;

  if (config.provider?.id === normalizedProviderId) {
    config.provider = undefined;
  }
  if (config.defaultModel?.providerId === normalizedProviderId) {
    config.defaultModel = undefined;
  }
  if (config.auxiliaryModel?.providerId === normalizedProviderId) {
    config.auxiliaryModel = undefined;
  }
  if (config.providers?.custom) {
    delete config.providers.custom[normalizedProviderId];
  }

  state.config = config;
  writePreviewState(state);
  return {
    catalog: buildPreviewCatalog(state),
    config,
  };
}

export async function startPreviewOpenAIBrowserAuth() {
  const stateToken = randomString(24);
  const codeVerifier = randomString(32);
  const codeChallenge = await createCodeChallenge(codeVerifier);
  const callbackURL = `${window.location.origin}/setup`;
  const authURL = new URL(OPENAI_AUTH_URL);
  authURL.searchParams.set("response_type", "code");
  authURL.searchParams.set("client_id", OPENAI_CLIENT_ID);
  authURL.searchParams.set("redirect_uri", callbackURL);
  authURL.searchParams.set("scope", OPENAI_SCOPE);
  authURL.searchParams.set("state", stateToken);
  authURL.searchParams.set("code_challenge", codeChallenge);
  authURL.searchParams.set("code_challenge_method", "S256");
  authURL.searchParams.set("id_token_add_organizations", "true");
  authURL.searchParams.set("codex_cli_simplified_flow", "true");
  authURL.searchParams.set("originator", "opencode");

  const pendingAuth: PreviewPendingAuth = {
    id: window.crypto.randomUUID(),
    providerId: "openai",
    method: "oauth-browser",
    status: "pending",
    authUrl: authURL.toString(),
    callbackUrl: callbackURL,
    instructions:
      "Complete the OpenAI browser flow in this tab. If the automatic return fails, paste the full callback URL or code manually.",
    expiresAt: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
    stateToken,
    codeVerifier,
  };

  const state = readPreviewState();
  state.pendingAuth = pendingAuth;
  writePreviewState(state);
  return pendingAuth;
}

export async function completePreviewOpenAIBrowserAuth(rawInput?: string) {
  const state = readPreviewState();
  const pendingAuth = state.pendingAuth;
  if (!pendingAuth) {
    throw new Error("没有进行中的 OpenAI 授权会话。");
  }
  if (new Date(pendingAuth.expiresAt).getTime() < Date.now()) {
    state.pendingAuth = null;
    writePreviewState(state);
    throw new Error("OpenAI 授权会话已过期，请重新开始。");
  }

  const source = rawInput?.trim() || window.location.href;
  const parsed = parseAuthorizationInput(source);
  if (parsed.state && parsed.state !== pendingAuth.stateToken) {
    throw new Error("授权 state 不匹配，请重新开始。");
  }
  if (!parsed.code) {
    throw new Error("缺少 authorization code。");
  }

  const body = new URLSearchParams();
  body.set("grant_type", "authorization_code");
  body.set("client_id", OPENAI_CLIENT_ID);
  body.set("code", parsed.code);
  body.set("redirect_uri", pendingAuth.callbackUrl || `${window.location.origin}/setup`);
  body.set("code_verifier", pendingAuth.codeVerifier);

  const response = await fetch(OPENAI_TOKEN_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: body.toString(),
  });

  if (!response.ok) {
    const errorText = (await response.text()).trim();
    throw new Error(errorText || `OpenAI token exchange failed (${response.status})`);
  }

  const token = (await response.json()) as {
    access_token?: string;
    refresh_token?: string;
  };
  if (!token.access_token) {
    throw new Error("OpenAI token exchange 未返回 access token。");
  }

  const result = persistPreviewProvider(
    {
      providerId: "openai",
      type: "openai",
      baseUrl: "https://api.openai.com/v1",
      modelId: "gpt-5.5",
      method: "oauth-browser",
    },
    "oauth-browser",
    {
      type: "oauth-browser",
      secret: token.access_token,
      refreshSecret: token.refresh_token,
      connectedAt: currentTimestamp(),
      lastValidatedAt: currentTimestamp(),
    },
  );

  const cleanURL = new URL(window.location.href);
  cleanURL.searchParams.delete("code");
  cleanURL.searchParams.delete("state");
  cleanURL.searchParams.delete("error");
  cleanURL.searchParams.delete("error_description");
  window.history.replaceState({}, "", cleanURL.toString());

  return result;
}

export function cancelPreviewOpenAIBrowserAuth() {
  const state = readPreviewState();
  state.pendingAuth = null;
  writePreviewState(state);
  return buildPreviewCatalog(state);
}

export function getPreviewPendingAuth() {
  return readPreviewState().pendingAuth ?? null;
}
