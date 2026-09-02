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
    selectExtensionDirectory(): Promise<string>;
    selectComposerFileOrDirectory(): Promise<
      | null
      | { kind: "directory"; path: string }
      | {
          kind: "file";
          name: string;
          mimeType: string;
          size: number;
          data: string;
        }
    >;
    inspectDroppedComposerResources(files: File[]): Promise<
      Array<
        | { kind: "directory"; path: string }
        | {
            kind: "file";
            name: string;
            mimeType: string;
            size: number;
            data: string;
          }
      >
    >;
    openExternal(target: string): Promise<void>;
    openPath(target: string): Promise<void>;
    openNewConversationWindow(): Promise<void>;
    focusWindow(): Promise<void>;
    toggleMaximize(): Promise<void>;
    exportDiagnostics(): Promise<string>;
    updates: {
      getState(): Promise<import("./features/updates/desktop-update-model").DesktopUpdateState>;
      check(): Promise<import("./features/updates/desktop-update-model").DesktopUpdateState>;
      download(): Promise<import("./features/updates/desktop-update-model").DesktopUpdateState>;
      install(): Promise<import("./features/updates/desktop-update-model").DesktopUpdateState>;
      cancel(): Promise<import("./features/updates/desktop-update-model").DesktopUpdateState>;
      onState(
        listener: (state: import("./features/updates/desktop-update-model").DesktopUpdateState) => void,
      ): () => void;
    };
    openExtensionView(input: {
      extensionId: string;
      viewId: string;
      surface: "page" | "dialog" | "tool-detail" | "settings" | "notification";
      context?: unknown;
    }): Promise<{ opened: boolean; extensionId: string; viewId: string; surface: string }>;
    mountEmbeddedExtensionView(input: {
      requestId: string;
      extensionId: string;
      viewId: string;
      surface: "tool-detail" | "page";
      context?: unknown;
      bounds: { x: number; y: number; width: number; height: number };
    }): Promise<{
      mounted: boolean;
      mountId: string;
      extensionId: string;
      viewId: string;
      surface: string;
    }>;
    updateEmbeddedExtensionViewBounds(input: {
      mountId: string;
      bounds: { x: number; y: number; width: number; height: number };
    }): Promise<{
      updated: boolean;
      bounds?: { x: number; y: number; width: number; height: number };
    }>;
    updateEmbeddedExtensionViewContext(input: {
      mountId: string;
      context?: unknown;
    }): Promise<{ updated: boolean; revision?: number }>;
    closeEmbeddedExtensionView(input: {
      mountId?: string;
      requestId?: string;
    }): Promise<{ closed: boolean }>;
    onEmbeddedExtensionViewClosed(
      listener: (event: { mountId: string; reason: string }) => void,
    ): () => void;
  };
}
