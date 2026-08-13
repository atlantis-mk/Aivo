import type { MCPServerConfig } from "@/services/aivo";
import { emptyMcpServer } from "@/features/projects/extension-settings-mcp-draft";

export const MCP_IMPORT_PLACEHOLDER = `{
  "mcpServers": {
    "filesystem": {
      "description": "读取和管理指定目录中的文件",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/project"]
    }
  }
}`;

export function parseMcpServersImport(raw: string): MCPServerConfig[] {
  const trimmed = raw.trim();
  if (!trimmed) {
    throw new Error("请先粘贴 MCP JSON 配置");
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`JSON 解析失败：${message}`);
  }

  if (!isPlainRecord(parsed)) {
    throw new Error("MCP JSON 必须是对象");
  }

  const serversValue = isPlainRecord(parsed.mcpServers)
    ? parsed.mcpServers
    : parsed;
  const servers = Object.entries(serversValue).map(([id, value]) =>
    mcpServerFromImportEntry(id, value),
  );
  if (servers.length === 0) {
    throw new Error("没有找到可导入的 MCP server");
  }
  return servers;
}

function mcpServerFromImportEntry(
  id: string,
  value: unknown,
): MCPServerConfig {
  if (!isPlainRecord(value)) {
    throw new Error(`${id} 的配置必须是对象`);
  }

  const cleanID = id.trim();
  if (!cleanID) {
    throw new Error("MCP server id 不能为空");
  }

  const command = optionalString(value.command);
  const url = optionalString(value.url);
  const transport =
    normalizeImportedMcpTransport(
      optionalString(value.transport) || optionalString(value.type),
    ) || (command ? "stdio" : "streamable_http");
  const server: MCPServerConfig = {
    ...emptyMcpServer(),
    id: cleanID,
    name: optionalString(value.name) || cleanID,
    displayName: optionalString(value.displayName),
    description: optionalString(value.description) ?? "",
    transport,
    command: transport === "stdio" ? command : "",
    args:
      transport === "stdio"
        ? optionalStringArray(value.args, `${cleanID}.args`)
        : [],
    cwd: optionalString(value.cwd),
    env: optionalStringMap(value.env, `${cleanID}.env`),
    url: transport === "stdio" ? "" : url,
    headers: optionalStringMap(value.headers, `${cleanID}.headers`),
    authType: normalizeImportedMcpAuthType(
      optionalString(value.authType) ||
        (optionalString(value.bearerToken) || optionalString(value.bearerTokenEnv)
          ? "bearer"
          : undefined),
    ),
    bearerTokenEnv: optionalString(value.bearerTokenEnv),
    bearerToken: optionalString(value.bearerToken),
    bearerAuthMode: optionalString(value.bearerTokenEnv) ? "env" : "direct",
    oauthIssuerUrl: optionalString(value.oauthIssuerUrl),
    oauthClientId: optionalString(value.oauthClientId),
    oauthScopes: optionalStringArray(
      value.oauthScopes,
      `${cleanID}.oauthScopes`,
    ),
    roots: optionalStringArray(value.roots, `${cleanID}.roots`),
    timeoutSeconds: optionalNumber(value.timeoutSeconds),
    connectTimeoutSeconds: optionalNumber(value.connectTimeoutSeconds),
    enabled: optionalBoolean(value.enabled) ?? !optionalBoolean(value.disabled),
  };

  if (server.transport === "stdio" && !server.command?.trim()) {
    throw new Error(`${cleanID} 缺少 command`);
  }
  if (server.transport !== "stdio" && !server.url?.trim()) {
    throw new Error(`${cleanID} 缺少 url`);
  }
  return server;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function optionalString(value: unknown) {
  return typeof value === "string" ? value.trim() : undefined;
}

function optionalBoolean(value: unknown) {
  return typeof value === "boolean" ? value : undefined;
}

function optionalNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function optionalStringArray(value: unknown, label: string) {
  if (value == null) return [];
  if (!Array.isArray(value)) {
    throw new Error(`${label} 必须是字符串数组`);
  }
  return value.map((item, index) => {
    if (typeof item !== "string") {
      throw new Error(`${label}[${index}] 必须是字符串`);
    }
    return item;
  });
}

function optionalStringMap(value: unknown, label: string) {
  if (value == null) return {};
  if (!isPlainRecord(value)) {
    throw new Error(`${label} 必须是对象`);
  }
  const out: Record<string, string> = {};
  for (const [key, item] of Object.entries(value)) {
    if (typeof item !== "string") {
      throw new Error(`${label}.${key} 必须是字符串`);
    }
    out[key] = item;
  }
  return out;
}

function normalizeImportedMcpTransport(
  value?: string,
): MCPServerConfig["transport"] | undefined {
  switch (value) {
    case "stdio":
    case "streamable_http":
    case "sse":
      return value;
    case "http":
      return "streamable_http";
    default:
      return undefined;
  }
}

function normalizeImportedMcpAuthType(
  value?: string,
): MCPServerConfig["authType"] {
  switch (value) {
    case "bearer":
    case "oauth":
      return value;
    case "none":
    default:
      return "none";
  }
}
