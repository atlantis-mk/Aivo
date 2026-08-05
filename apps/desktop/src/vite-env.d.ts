/// <reference types="vite/client" />

interface Window {
  aivo: {
    platform: string;
    versions: {
      chrome: string;
      electron: string;
      node: string;
    };
    coreUrl: string;
    invoke<T>(method: string, ...args: unknown[]): Promise<T>;
    selectProjectDirectory(): Promise<string>;
    openExternal(target: string): Promise<void>;
    openPath(target: string): Promise<void>;
    focusWindow(): Promise<void>;
    toggleMaximize(): Promise<void>;
    exportDiagnostics(): Promise<string>;
    openExtensionView(input: {
      extensionId: string;
      viewId: string;
      surface: "page" | "dialog" | "tool-detail" | "settings" | "notification";
      context?: unknown;
    }): Promise<{ opened: boolean; extensionId: string; viewId: string; surface: string }>;
  };
}
