import { contextBridge, ipcRenderer, webUtils } from "electron";

contextBridge.exposeInMainWorld("aivoDesktop", {
  platform: process.platform,
  runtime: {
    getStatus: (): Promise<RuntimeStatus> =>
      ipcRenderer.invoke("runtime:get-status"),
    start: (): Promise<RuntimeStatus> => ipcRenderer.invoke("runtime:start"),
    stop: (): Promise<RuntimeStatus> => ipcRenderer.invoke("runtime:stop"),
    onStatus: (listener: (status: RuntimeStatus) => void): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        status: RuntimeStatus,
      ): void => listener(status);
      ipcRenderer.on("runtime:status", handler);
      return (): void => {
        ipcRenderer.removeListener("runtime:status", handler);
      };
    },
  },
  codex: {
    configureProvider: (input: BackendProviderConnectionInput): Promise<void> =>
      ipcRenderer.invoke("provider:configure", input),
    deleteProvider: (providerId: string): Promise<void> =>
      ipcRenderer.invoke("provider:delete", providerId),
    readModelConfig: (): Promise<CodexModelConfig> =>
      ipcRenderer.invoke("modelPreferences:read"),
    saveModelPreferences: (input: {
      model: string;
      modelProvider: string;
      reasoningEffort: string;
    }): Promise<void> =>
      ipcRenderer.invoke("modelPreferences:save", input),
    cancelLogin: (loginId: string): Promise<void> =>
      ipcRenderer.invoke("account:cancel-login", loginId),
    getAccount: (): Promise<CodexAccount> => ipcRenderer.invoke("account:read"),
    listModels: (): Promise<CodexModel[]> => ipcRenderer.invoke("models:list"),
    listSkills: (workspaceRoot?: string, forceReload?: boolean): Promise<CodexSkillCatalog> =>
      ipcRenderer.invoke("skills:list", workspaceRoot, forceReload),
    listMcpServers: (): Promise<CodexMcpServer[]> => ipcRenderer.invoke("mcp:servers:list"),
    listCodexModels: (): Promise<CodexModel[]> =>
      ipcRenderer.invoke("models:codex:list"),
    listThreads: (limit: number, searchTerm?: string): Promise<CodexThread[]> =>
      ipcRenderer.invoke("threads:list", limit, searchTerm),
    listThreadTurns: (threadId: string): Promise<CodexThreadTurn[]> =>
      ipcRenderer.invoke("thread:turns:list", threadId),
    resumeThread: (threadId: string): Promise<void> =>
      ipcRenderer.invoke("thread:resume", threadId),
    startThread: (input: {
      cwd?: string;
      model?: string;
      modelProvider?: string;
      permissionMode?: "request_approval" | "auto_review" | "full_access";
    }): Promise<CodexThreadStart> => ipcRenderer.invoke("thread:start", input),
    startTurn: (input: {
      resourceInputs?: CodexResourceInput[];
      images?: Array<{ data: string; mimeType: string }>;
      model?: string;
      permissionMode?: "request_approval" | "auto_review" | "full_access";
      modelProvider?: string;
      text: string;
      textElements?: Array<{
        byteRange: { start: number; end: number };
        placeholder: string;
      }>;
      threadId: string;
    }): Promise<CodexTurnStart> => ipcRenderer.invoke("turn:start", input),
    interruptTurn: (input: CodexTurnStart & CodexThreadStart): Promise<void> =>
      ipcRenderer.invoke("turn:interrupt", input),
    steerTurn: (input: {
      resourceInputs?: CodexResourceInput[];
      clientUserMessageId: string;
      expectedTurnId: string;
      text: string;
      threadId: string;
    }): Promise<CodexTurnStart> => ipcRenderer.invoke("turn:steer", input),
    archiveThread: (threadId: string): Promise<void> =>
      ipcRenderer.invoke("thread:archive", threadId),
    login: (): Promise<CodexLoginStart> => ipcRenderer.invoke("account:login"),
    logout: (): Promise<void> => ipcRenderer.invoke("account:logout"),
    listApprovals: (): Promise<CodexApprovalRequest[]> =>
      ipcRenderer.invoke("approvals:list"),
    resolveApproval: (
      requestId: string,
      action: "approve" | "approveForSession" | "deny" | "cancel",
    ): Promise<void> =>
      ipcRenderer.invoke("approvals:resolve", { action, requestId }),
    onAccount: (listener: (account: CodexAccount) => void): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        account: CodexAccount,
      ): void => listener(account);
      ipcRenderer.on("account:updated", handler);
      return (): void => {
        ipcRenderer.removeListener("account:updated", handler);
      };
    },
    onLoginCompleted: (
      listener: (completion: CodexLoginCompletion) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        completion: CodexLoginCompletion,
      ): void => listener(completion);
      ipcRenderer.on("account:login-completed", handler);
      return (): void => {
        ipcRenderer.removeListener("account:login-completed", handler);
      };
    },
    onRuntimeEvent: (
      listener: (event: CodexRuntimeEvent) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        runtimeEvent: CodexRuntimeEvent,
      ): void => listener(runtimeEvent);
      ipcRenderer.on("codex:event", handler);
      return (): void => {
        ipcRenderer.removeListener("codex:event", handler);
      };
    },
    onApprovalRequest: (
      listener: (request: CodexApprovalRequest) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        payload: { request: CodexApprovalRequest },
      ): void => listener(payload.request);
      ipcRenderer.on("codex:approval-request", handler);
      return (): void => {
        ipcRenderer.removeListener("codex:approval-request", handler);
      };
    },
    onApprovalResolved: (
      listener: (payload: { requestId: string }) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        payload: { requestId: string },
      ): void => listener(payload);
      ipcRenderer.on("codex:approval-resolved", handler);
      return (): void => {
        ipcRenderer.removeListener("codex:approval-resolved", handler);
      };
    },
    onUserInputRequest: (
      listener: (payload: { request: unknown }) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        payload: { request: unknown },
      ): void => listener(payload);
      ipcRenderer.on("codex:user-input-request", handler);
      return (): void => {
        ipcRenderer.removeListener("codex:user-input-request", handler);
      };
    },
    onUserInputResolved: (
      listener: (payload: { itemId: string }) => void,
    ): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        payload: { itemId: string },
      ): void => listener(payload);
      ipcRenderer.on("codex:user-input-resolved", handler);
      return (): void => {
        ipcRenderer.removeListener("codex:user-input-resolved", handler);
      };
    },
    resolveUserInput: (
      itemId: string,
      answers: Record<string, { answers: string[] }>,
    ): Promise<void> =>
      ipcRenderer.invoke("user-input:resolve", { itemId, answers }),
  },
  workspace: {
    choose: (): Promise<string | null> =>
      ipcRenderer.invoke("workspace:choose"),
  },
  composer: {
    selectLocalResource: (): Promise<ComposerLocalSelection | null> =>
      ipcRenderer.invoke("composer:select-local-resource"),
    inspectDroppedResources: (
      files: File[],
    ): Promise<ComposerLocalSelection[]> =>
      ipcRenderer.invoke(
        "composer:inspect-dropped-resources",
        files
          .map((file) => {
            try {
              return webUtils.getPathForFile(file);
            } catch {
              return "";
            }
          })
          .filter(Boolean),
      ),
  },
  file: {
    openPath: (target: string): Promise<string> =>
      ipcRenderer.invoke("file:open-path", target),
    openExternal: (target: string): Promise<void> =>
      ipcRenderer.invoke("file:open-external", target),
    openText: (name: string, text: string): Promise<string> =>
      ipcRenderer.invoke("file:open-temp-text", { name, text }),
    pathForFile: (file: File): string => {
      try {
        return webUtils.getPathForFile(file);
      } catch {
        return "";
      }
    },
  },
  updates: {
    cancel: (): Promise<DesktopUpdateState | undefined> =>
      ipcRenderer.invoke("update:cancel"),
    check: (): Promise<DesktopUpdateState | undefined> =>
      ipcRenderer.invoke("update:check"),
    download: (): Promise<DesktopUpdateState | undefined> =>
      ipcRenderer.invoke("update:download"),
    getState: (): Promise<DesktopUpdateState | undefined> =>
      ipcRenderer.invoke("update:get-state"),
    install: (): Promise<DesktopUpdateState | undefined> =>
      ipcRenderer.invoke("update:install"),
    onState: (listener: (state: DesktopUpdateState) => void): (() => void) => {
      const handler = (
        _event: Electron.IpcRendererEvent,
        state: DesktopUpdateState,
      ): void => listener(state);
      ipcRenderer.on("update:state", handler);
      return (): void => {
        ipcRenderer.removeListener("update:state", handler);
      };
    },
  },
  window: {
    toggleMaximize: (): Promise<void> =>
      ipcRenderer.invoke("window:toggle-maximize"),
  },
});
