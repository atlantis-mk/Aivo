import { invoke } from "@/services/aivo/invoke";

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
  activationPolicy?: string;
  selectionGroup?: {
    id: string;
    name: string;
    description?: string;
  };
  enabled: boolean;
};

export type SessionActiveToolsResult = {
  coreToolNames: string[];
  sessionId: string;
  toolNames: string[];
};

export function listToolCatalog(workspaceRoot = "") {
  return invoke<ToolCatalogEntry[]>("ListToolCatalog", { workspaceRoot });
}

export function setGlobalToolEnabled(
  name: string,
  enabled: boolean,
  workspaceRoot = "",
) {
  return invoke<ToolCatalogEntry>("SetGlobalToolEnabled", {
    name,
    enabled,
    workspaceRoot,
  });
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
