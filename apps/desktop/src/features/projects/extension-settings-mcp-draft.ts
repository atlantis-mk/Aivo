import type { MCPServerConfig } from "@/services/aivo";
import type { KeyValueRow } from "@/features/projects/extension-settings-types";

export function emptyMcpServer(): MCPServerConfig {
  return {
    id: "",
    name: "",
    description: "",
    transport: "stdio",
    command: "",
    args: [],
    roots: [],
    authType: "none",
    bearerAuthMode: "direct",
    enabled: false,
  };
}

export function mcpServerToDraft(server: MCPServerConfig): MCPServerConfig {
  return {
    ...emptyMcpServer(),
    ...server,
    args: server.args ?? [],
    env: server.env ?? {},
    headers: server.headers ?? {},
    roots: server.roots ?? [],
    authType: server.authType ?? "none",
    bearerAuthMode: server.bearerTokenEnv?.trim() ? "env" : "direct",
  };
}

export function applyGeneratedMcpDescription(
  draft: MCPServerConfig,
  description: string,
): MCPServerConfig {
  return { ...draft, description };
}

export function normalizeMcpDraft(draft: MCPServerConfig): MCPServerConfig {
  const httpAuthType = draft.authType ?? "none";
  return {
    ...draft,
    name: draft.name || draft.id,
    args: draft.transport === "stdio" ? (draft.args ?? []) : [],
    command: draft.transport === "stdio" ? draft.command : "",
    url: draft.transport === "stdio" ? "" : draft.url,
    authType: draft.transport === "stdio" ? "none" : httpAuthType,
    bearerTokenEnv:
      draft.transport !== "stdio" &&
      (httpAuthType === "oauth" ||
        (httpAuthType === "bearer" && draft.bearerAuthMode === "env"))
        ? draft.bearerTokenEnv
        : "",
    bearerTokenRef:
      draft.transport !== "stdio" &&
      httpAuthType === "bearer" &&
      draft.bearerAuthMode !== "env"
        ? draft.bearerTokenRef
        : "",
    bearerToken:
      draft.transport !== "stdio" &&
      httpAuthType === "bearer" &&
      draft.bearerAuthMode !== "env"
        ? draft.bearerToken
        : "",
    oauthIssuerUrl:
      draft.transport !== "stdio" && httpAuthType === "oauth"
        ? draft.oauthIssuerUrl
        : "",
    oauthClientId:
      draft.transport !== "stdio" && httpAuthType === "oauth"
        ? draft.oauthClientId
        : "",
    oauthScopes:
      draft.transport !== "stdio" && httpAuthType === "oauth"
        ? (draft.oauthScopes ?? [])
        : [],
    roots: draft.roots ?? [],
  };
}

export function nonEmptyStrings(values?: string[]) {
  const next = values && values.length > 0 ? values : [""];
  return [...next];
}

export function compactStrings(values: string[]) {
  return values.map((value) => value.trim()).filter(Boolean);
}

export function mapToRows(value?: Record<string, string>): KeyValueRow[] {
  const rows = Object.entries(value ?? {}).map(([key, rowValue]) => ({
    key,
    value: rowValue,
  }));
  return rows.length > 0 ? rows : [{ key: "", value: "" }];
}

export function rowsToMap(rows: KeyValueRow[]) {
  const next: Record<string, string> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (key) {
      next[key] = row.value;
    }
  }
  return next;
}

export function mcpTransportLabel(transport: MCPServerConfig["transport"]) {
  switch (transport) {
    case "streamable_http":
      return "Streamable HTTP";
    case "sse":
      return "SSE";
    case "stdio":
    default:
      return "stdio";
  }
}

export function parseWords(value: string) {
  return value
    .split(/\s+/g)
    .map((word) => word.trim())
    .filter(Boolean);
}

export function canSaveMcpDraft(draft: MCPServerConfig) {
  if (!draft.id.trim()) return false;
  if ((draft.description?.trim().length ?? 0) > 500) {
    return false;
  }
  if (draft.transport === "stdio") return Boolean(draft.command?.trim());
  if (
    draft.authType === "bearer" &&
    (draft.bearerAuthMode === "env"
      ? !draft.bearerTokenEnv?.trim()
      : !draft.bearerToken?.trim() && !draft.bearerTokenRef?.trim())
  ) {
    return false;
  }
  return Boolean(draft.url?.trim());
}
