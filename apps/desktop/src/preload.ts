import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("aivoDesktop", {
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
});
