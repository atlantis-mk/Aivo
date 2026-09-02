import { contextBridge, ipcRenderer } from "electron";

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
    cancelLogin: (loginId: string): Promise<void> =>
      ipcRenderer.invoke("account:cancel-login", loginId),
    getAccount: (): Promise<CodexAccount> => ipcRenderer.invoke("account:read"),
    listModels: (): Promise<CodexModel[]> => ipcRenderer.invoke("models:list"),
    login: (): Promise<CodexLoginStart> => ipcRenderer.invoke("account:login"),
    logout: (): Promise<void> => ipcRenderer.invoke("account:logout"),
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
  },
  workspace: {
    choose: (): Promise<string | null> => ipcRenderer.invoke("workspace:choose"),
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
    toggleMaximize: (): Promise<void> => ipcRenderer.invoke("window:toggle-maximize"),
  },
});
