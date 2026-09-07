import { useState } from "react";

import {
  discoverMCPOAuth,
  getMCPPrompt,
  getMCPOAuthStatus,
  insertMCPPromptIntoSession,
  insertMCPResourceIntoSession,
  probeMCPServer,
  readMCPResource,
  readMCPServerLog,
  startMCPOAuth,
  type MCPPromptGetResult,
  type MCPPromptRecord,
  type MCPResourceReadResult,
  type MCPResourceRecord,
  type MCPOAuthDiscoveryResult,
  type MCPOAuthStartResult,
  type MCPOAuthStatus,
} from "@/services/aivo";

export function useMcpServerCapabilityState({
  onReload,
  serverId,
  sessionId,
}: {
  onReload: () => Promise<void>;
  serverId: string;
  sessionId?: string;
}) {
  const [log, setLog] = useState("");
  const [logError, setLogError] = useState("");
  const [loadingLog, setLoadingLog] = useState(false);
  const [oauthDiscovery, setOauthDiscovery] =
    useState<MCPOAuthDiscoveryResult | null>(null);
  const [oauthStart, setOauthStart] = useState<MCPOAuthStartResult | null>(
    null,
  );
  const [oauthStatus, setOauthStatus] = useState<MCPOAuthStatus | null>(null);
  const [oauthError, setOauthError] = useState("");
  const [loadingOAuth, setLoadingOAuth] = useState(false);
  const [promptInputs, setPromptInputs] = useState<
    Record<string, Record<string, string>>
  >({});
  const [promptResult, setPromptResult] =
    useState<MCPPromptGetResult | null>(null);
  const [promptError, setPromptError] = useState("");
  const [loadingPromptId, setLoadingPromptId] = useState("");
  const [insertingPromptId, setInsertingPromptId] = useState("");
  const [templateInputs, setTemplateInputs] = useState<
    Record<string, Record<string, string>>
  >({});
  const [resourceResult, setResourceResult] =
    useState<MCPResourceReadResult | null>(null);
  const [resourceError, setResourceError] = useState("");
  const [loadingResourceId, setLoadingResourceId] = useState("");
  const [insertingResourceId, setInsertingResourceId] = useState("");

  async function loadLog() {
    setLoadingLog(true);
    setLogError("");
    try {
      const result = await readMCPServerLog(serverId);
      setLog(result.content || "没有日志输出");
    } catch (err) {
      setLogError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingLog(false);
    }
  }

  async function loadOAuthDiscovery() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      setOauthDiscovery(await discoverMCPOAuth(serverId));
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function connectOAuth() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      const result = await startMCPOAuth(serverId);
      setOauthStart(result);
      if (result.url) {
        window.open(result.url, "_blank", "noopener,noreferrer");
      }
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function refreshOAuthStatus() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      setOauthStatus(await getMCPOAuthStatus(serverId));
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function probeServer() {
    await probeMCPServer(serverId);
    await onReload();
  }

  async function loadPrompt(prompt: MCPPromptRecord) {
    setLoadingPromptId(prompt.id);
    setPromptError("");
    try {
      setPromptResult(
        await getMCPPrompt(
          serverId,
          prompt.name,
          promptArgumentValues(prompt, promptInputs[prompt.id] ?? {}),
        ),
      );
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingPromptId("");
    }
  }

  async function insertPrompt(prompt: MCPPromptRecord) {
    if (!sessionId) return;
    setInsertingPromptId(prompt.id);
    setPromptError("");
    try {
      await insertMCPPromptIntoSession(
        sessionId,
        serverId,
        prompt.name,
        promptArgumentValues(prompt, promptInputs[prompt.id] ?? {}),
      );
      setPromptResult({
        serverId,
        name: prompt.name,
        content: "已插入当前会话",
      });
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : String(err));
    } finally {
      setInsertingPromptId("");
    }
  }

  async function loadResource(resource: MCPResourceRecord, uri: string) {
    setLoadingResourceId(resource.id);
    setResourceError("");
    try {
      setResourceResult(await readMCPResource(serverId, uri));
    } catch (err) {
      setResourceError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingResourceId("");
    }
  }

  async function insertResource(resource: MCPResourceRecord, uri: string) {
    if (!sessionId) return;
    setInsertingResourceId(resource.id);
    setResourceError("");
    try {
      await insertMCPResourceIntoSession(sessionId, serverId, uri);
      setResourceResult({
        serverId,
        uri,
        content: "已插入当前会话",
      });
    } catch (err) {
      setResourceError(err instanceof Error ? err.message : String(err));
    } finally {
      setInsertingResourceId("");
    }
  }

  function updatePromptInput(promptId: string, name: string, value: string) {
    setPromptInputs((current) => ({
      ...current,
      [promptId]: {
        ...(current[promptId] ?? {}),
        [name]: value,
      },
    }));
  }

  function updateTemplateInput(templateId: string, name: string, value: string) {
    setTemplateInputs((current) => ({
      ...current,
      [templateId]: {
        ...(current[templateId] ?? {}),
        [name]: value,
      },
    }));
  }

  return {
    insertingPromptId,
    insertingResourceId,
    loadingLog,
    loadingOAuth,
    loadingPromptId,
    loadingResourceId,
    log,
    logError,
    oauthDiscovery,
    oauthError,
    oauthStart,
    oauthStatus,
    promptError,
    promptInputs,
    promptResult,
    resourceError,
    resourceResult,
    templateInputs,
    connectOAuth,
    insertPrompt,
    insertResource,
    loadLog,
    loadOAuthDiscovery,
    loadPrompt,
    loadResource,
    probeServer,
    refreshOAuthStatus,
    updatePromptInput,
    updateTemplateInput,
  };
}

function promptArgumentValues(
  prompt: MCPPromptRecord,
  values: Record<string, string>,
) {
  return Object.fromEntries(
    (prompt.arguments ?? []).map((argument) => [
      argument.name,
      values[argument.name] ?? "",
    ]),
  );
}
