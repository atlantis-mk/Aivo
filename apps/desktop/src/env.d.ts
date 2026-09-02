interface RuntimeStatus {
  state: "stopped" | "starting" | "ready" | "error";
  detail: string;
}

interface CodexAccount {
  authMode: string | null;
  email: string | null;
  planType: string | null;
}

interface CodexLoginStart {
  loginId: string;
}

interface CodexModel {
  id: string;
  name: string;
  description: string;
}

interface CodexLoginCompletion {
  error: string | null;
  loginId: string | null;
  success: boolean;
}

type DesktopUpdatePhase =
  | "idle"
  | "checking"
  | "up-to-date"
  | "available"
  | "downloading"
  | "ready"
  | "unsupported"
  | "error";

interface DesktopUpdateState {
  phase: DesktopUpdatePhase;
  currentVersion: string;
  availableVersion: string;
  progress: number;
  message: string;
  errorCode: string;
  automaticChecksEnabled: boolean;
}

interface AivoDesktopApi {
  platform: NodeJS.Platform;
  runtime: {
    getStatus(): Promise<RuntimeStatus>;
    start(): Promise<RuntimeStatus>;
    stop(): Promise<RuntimeStatus>;
    onStatus(listener: (status: RuntimeStatus) => void): () => void;
  };
  codex: {
    cancelLogin(loginId: string): Promise<void>;
    getAccount(): Promise<CodexAccount>;
    listModels(): Promise<CodexModel[]>;
    login(): Promise<CodexLoginStart>;
    logout(): Promise<void>;
    onAccount(listener: (account: CodexAccount) => void): () => void;
    onLoginCompleted(
      listener: (completion: CodexLoginCompletion) => void,
    ): () => void;
  };
  workspace: {
    choose(): Promise<string | null>;
  };
  updates: {
    cancel(): Promise<DesktopUpdateState | undefined>;
    check(): Promise<DesktopUpdateState | undefined>;
    download(): Promise<DesktopUpdateState | undefined>;
    getState(): Promise<DesktopUpdateState | undefined>;
    install(): Promise<DesktopUpdateState | undefined>;
    onState(listener: (state: DesktopUpdateState) => void): () => void;
  };
  window: {
    toggleMaximize(): Promise<void>;
  };
}

interface Window {
  aivoDesktop: AivoDesktopApi;
}
