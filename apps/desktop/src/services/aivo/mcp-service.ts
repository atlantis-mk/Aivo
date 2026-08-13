import type { SessionEvent } from "@/services/aivo/session-event-service";
import { invoke } from "@/services/aivo/invoke";

export type MCPDiagnostic = {
  id: string;
  serverId?: string;
  level: string;
  message: string;
  metadata?: Record<string, unknown>;
  timeCreated: string;
};

export type MCPServerConfig = {
  id: string;
  name: string;
  displayName?: string;
  description?: string;
  transport: "stdio" | "streamable_http" | "sse";
  command?: string;
  args?: string[];
  cwd?: string;
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
  authType?: "none" | "bearer" | "oauth";
  bearerTokenEnv?: string;
  bearerTokenRef?: string;
  bearerToken?: string;
  bearerAuthMode?: "direct" | "env";
  oauthIssuerUrl?: string;
  oauthClientId?: string;
  oauthScopes?: string[];
  oauthAccessTokenRef?: string;
  oauthRefreshTokenRef?: string;
  oauthExpiresAt?: string;
  roots?: string[];
  timeoutSeconds?: number;
  connectTimeoutSeconds?: number;
  enabled: boolean;
  status?: string;
  error?: string;
  timeCreated?: string;
  timeUpdated?: string;
};

export type MCPToolRecord = {
  id: string;
  serverId: string;
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  capability?: string;
  riskLevel?: string;
  timeUpdated: string;
};

export type MCPPromptArgument = {
  name: string;
  description?: string;
  required?: boolean;
};

export type MCPPromptRecord = {
  id: string;
  serverId: string;
  name: string;
  description?: string;
  arguments?: MCPPromptArgument[];
  timeUpdated: string;
};

export type MCPResourceRecord = {
  id: string;
  serverId: string;
  uri?: string;
  uriTemplate?: string;
  name: string;
  description?: string;
  mimeType?: string;
  template?: boolean;
  timeUpdated: string;
};

export type MCPContentBlock = {
  type: string;
  text?: string;
  uri?: string;
  mimeType?: string;
  blob?: string;
};

export type MCPPromptMessage = {
  role: string;
  content?: MCPContentBlock[];
};

export type MCPPromptGetResult = {
  serverId: string;
  name: string;
  description?: string;
  messages?: MCPPromptMessage[];
  content?: string;
  structured?: Record<string, unknown>;
};

export type MCPResourceContent = {
  uri?: string;
  mimeType?: string;
  text?: string;
  blob?: string;
};

export type MCPResourceReadResult = {
  serverId: string;
  uri: string;
  contents?: MCPResourceContent[];
  content?: string;
  structured?: Record<string, unknown>;
};

export type MCPServerLogResult = {
  serverId: string;
  content: string;
  offset: number;
  nextOffset: number;
  size: number;
  truncated?: boolean;
};

export type MCPServerListItem = {
  server: MCPServerConfig;
  tools?: MCPToolRecord[];
  prompts?: MCPPromptRecord[];
  resources?: MCPResourceRecord[];
  resourceTemplates?: MCPResourceRecord[];
  diagnostics?: MCPDiagnostic[];
};

export type MCPProbeResult = {
  ok: boolean;
  serverId?: string;
  status?: string;
  error?: string;
  tools?: MCPToolRecord[];
  prompts?: MCPPromptRecord[];
  resources?: MCPResourceRecord[];
  resourceTemplates?: MCPResourceRecord[];
  diagnostics?: MCPDiagnostic[];
};

export type MCPDescriptionGenerateResult = {
  description: string;
};

export type MCPOAuthDiscoveryResult = {
  serverId?: string;
  resource?: string;
  resourceMetadataUrl?: string;
  authorizationServers?: string[];
  scopesSupported?: string[];
  selectedIssuer?: string;
  authorizationEndpoint?: string;
  tokenEndpoint?: string;
  registrationEndpoint?: string;
  introspectionEndpoint?: string;
  revocationEndpoint?: string;
  codeChallengeMethods?: string[];
  responseTypesSupported?: string[];
  grantTypesSupported?: string[];
  authorizationUrl?: string;
  discoveryErrors?: string[];
  resourceMetadata?: Record<string, unknown>;
  authorizationMetadata?: Record<string, unknown>;
  requiresDynamicClientRegistration?: boolean;
};

export type MCPOAuthStartResult = {
  serverId: string;
  status: string;
  url?: string;
  instructions?: string;
  expiresAt?: string;
};

export type MCPOAuthStatus = {
  serverId: string;
  status: string;
  error?: string;
  connected?: boolean;
  expiresAt?: string;
  clientId?: string;
  tokenSource?: string;
};

export function listMCPServers(includeDisabled = true, includeTools = true) {
  return invoke<MCPServerListItem[]>("ListMCPServers", {
    includeDisabled,
    includeTools,
  });
}

export function saveMCPServer(server: MCPServerConfig) {
  const bearerToken = server.bearerToken;
  const configuration = { ...server };
  delete configuration.bearerAuthMode;
  delete configuration.bearerToken;
  return invoke<MCPServerConfig>("SaveMCPServer", {
    server: configuration,
    bearerToken: bearerToken?.trim() || undefined,
  });
}

export function setMCPServerEnabled(serverId: string, enabled: boolean) {
  return invoke<MCPServerConfig>("SetMCPServerEnabled", { serverId, enabled });
}

export function generateMCPDescription(serverId: string) {
  return invoke<MCPDescriptionGenerateResult>("GenerateMCPDescription", {
    serverId,
  });
}

export function probeMCPServer(serverId: string) {
  return invoke<MCPProbeResult>("ProbeMCPServer", { serverId });
}

export function getMCPPrompt(
  serverId: string,
  name: string,
  args: Record<string, string> = {},
) {
  return invoke<MCPPromptGetResult>("GetMCPPrompt", {
    serverId,
    name,
    arguments: args,
  });
}

export function readMCPResource(serverId: string, uri: string) {
  return invoke<MCPResourceReadResult>("ReadMCPResource", { serverId, uri });
}

export function insertMCPPromptIntoSession(
  sessionId: string,
  serverId: string,
  name: string,
  args: Record<string, string> = {},
) {
  return invoke<SessionEvent>("InsertMCPPromptIntoSession", {
    sessionId,
    serverId,
    name,
    arguments: args,
  });
}

export function insertMCPResourceIntoSession(
  sessionId: string,
  serverId: string,
  uri: string,
) {
  return invoke<SessionEvent>("InsertMCPResourceIntoSession", {
    sessionId,
    serverId,
    uri,
  });
}

export function readMCPServerLog(serverId: string, limit = 16_000) {
  return invoke<MCPServerLogResult>("ReadMCPServerLog", {
    serverId,
    limit,
    tail: true,
  });
}

export function discoverMCPOAuth(serverId: string) {
  return invoke<MCPOAuthDiscoveryResult>("DiscoverMCPOAuth", { serverId });
}

export function startMCPOAuth(serverId: string) {
  return invoke<MCPOAuthStartResult>("StartMCPOAuth", { serverId });
}

export function getMCPOAuthStatus(serverId: string) {
  return invoke<MCPOAuthStatus>("GetMCPOAuthStatus", { serverId });
}
