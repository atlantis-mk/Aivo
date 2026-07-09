import { invoke } from "@/services/aivo/invoke";

export type PluginManifest = {
  id: string;
  name: string;
  version?: string;
  displayName?: string;
  description?: string;
  author?: string;
  keywords?: string[];
  hooks?: string[];
  tools?: ToolCatalogEntry[];
};

export type PluginInstall = {
  id: string;
  manifest: PluginManifest;
  rootPath: string;
  manifestPath: string;
  enabled: boolean;
  status: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type PluginDiagnostic = {
  id: string;
  pluginId?: string;
  serverId?: string;
  level: string;
  message: string;
  metadata?: Record<string, unknown>;
  timeCreated: string;
};

export type ToolCatalogEntry = {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  namespace?: string;
  capability?: string;
  riskLevel?: string;
  category?: string;
  toolsets?: string[];
  source: string;
  sourceId?: string;
  registrationId?: string;
  enabled: boolean;
};

export type SessionActiveToolsResult = {
  sessionId: string;
  toolNames: string[];
};

export type PluginListItem = {
  plugin: PluginInstall;
  diagnostics?: PluginDiagnostic[];
  tools?: ToolCatalogEntry[];
};

export function listPlugins(includeDisabled = true) {
  return invoke<PluginListItem[]>("ListPlugins", {
    includeDisabled,
    includeDiagnostics: true,
    limit: 100,
  });
}

export function installPluginFromPath(path: string, enable = true) {
  return invoke<PluginInstall>("InstallPluginFromPath", { path, enable });
}

export function setPluginEnabled(pluginId: string, enabled: boolean) {
  return invoke<PluginInstall>("SetPluginEnabled", { pluginId, enabled });
}

export function reloadPlugins() {
  return invoke<PluginListItem[]>("ReloadPlugins");
}

export function listToolCatalog(workspaceRoot = "") {
  return invoke<ToolCatalogEntry[]>("ListToolCatalog", { workspaceRoot });
}

export function getSessionActiveTools(sessionId: string) {
  return invoke<SessionActiveToolsResult>("GetSessionActiveTools", sessionId);
}

export function setSessionActiveTools(sessionId: string, toolNames: string[]) {
  return invoke<SessionActiveToolsResult>("SetSessionActiveTools", {
    sessionId,
    toolNames,
  });
}
