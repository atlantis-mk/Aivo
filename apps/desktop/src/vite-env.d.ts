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
    browser: {
      getState(tabId?: string): Promise<AivoBrowserState>;
      setVisible(tabId: string, visible: boolean): Promise<AivoBrowserState>;
      close(tabId: string): Promise<AivoBrowserState>;
      setBounds(tabId: string, bounds: AivoBrowserBounds): Promise<AivoBrowserState>;
      navigate(tabId: string, target: string): Promise<AivoBrowserState>;
      goBack(tabId: string): Promise<AivoBrowserState>;
      goForward(tabId: string): Promise<AivoBrowserState>;
      reload(tabId: string): Promise<AivoBrowserState>;
      stop(tabId: string): Promise<AivoBrowserState>;
      loadFavicon(favicons: string[], pageURL: string): Promise<string>;
      onNavigateCurrent(
        callback: (payload: AivoBrowserNavigateCurrentPayload) => void,
      ): () => void;
      onStateChange(
        callback: (state: AivoBrowserState, tabId: string) => void,
        tabId?: string,
      ): () => void;
      onOpenRequest(
        callback: (payload: AivoBrowserOpenRequestPayload) => void | Promise<void>,
      ): () => void;
    };
  };
}

type AivoBrowserBounds = {
  x: number;
  y: number;
  width: number;
  height: number;
};

type AivoBrowserState = {
  url: string;
  title: string;
  favicon: string;
  loading: boolean;
  canGoBack: boolean;
  canGoForward: boolean;
  error: string;
};

type AivoBrowserNavigateCurrentPayload = {
  sourceWebContentsId: number;
  url: string;
};

type AivoBrowserOpenRequestPayload = {
  requestId: string;
  tabId: string;
  url?: string;
};

type AivoWebviewElement = HTMLElement & {
  canGoBack?: () => boolean;
  canGoForward?: () => boolean;
  getTitle?: () => string;
  getURL?: () => string;
  getWebContentsId?: () => number;
  goBack?: () => void;
  goForward?: () => void;
  isLoading?: () => boolean;
  loadURL?: (url: string) => Promise<void>;
  reload?: () => void;
  stop?: () => void;
};

declare namespace JSX {
  interface IntrinsicElements {
    webview: React.DetailedHTMLProps<
      React.HTMLAttributes<AivoWebviewElement>,
      AivoWebviewElement
    > & {
      allowpopups?: boolean | string;
      partition?: string;
      src?: string;
      webpreferences?: string;
    };
  }
}
