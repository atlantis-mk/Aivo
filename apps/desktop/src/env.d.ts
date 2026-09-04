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

interface CodexThreadStart {
  threadId: string;
}

interface CodexThread {
  id: string;
  cwd: string;
  model: string | null;
  modelProvider: string;
  name: string | null;
  parentThreadId: string | null;
  preview: string;
  source: string;
  status: string;
  timeCreated: string;
  timeUpdated: string;
}

interface CodexThreadTurn {
  model?: string | null;
  modelProvider?: string | null;
  completedAt: string | null;
  durationMs: number | null;
  error: string | null;
  id: string;
  items: unknown[];
  startedAt: string | null;
  status: string;
}

interface CodexTurnStart {
  turnId: string;
}

interface BackendProviderConnectionInput {
  apiKey: string;
  baseUrl: string;
  model: string;
  name: string;
  providerId: string;
}

interface CodexRuntimeEvent {
  method: string;
  params: unknown;
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
    configureProvider(input: BackendProviderConnectionInput): Promise<void>;
    deleteProvider(providerId: string): Promise<void>;
    cancelLogin(loginId: string): Promise<void>;
    getAccount(): Promise<CodexAccount>;
    listCodexModels(): Promise<CodexModel[]>;
    listModels(): Promise<CodexModel[]>;
    listThreads(limit: number): Promise<CodexThread[]>;
    listThreadTurns(threadId: string): Promise<CodexThreadTurn[]>;
    resumeThread(threadId: string): Promise<void>;
    startThread(input: {
      cwd?: string;
      model?: string;
      modelProvider?: string;
    }): Promise<CodexThreadStart>;
    startTurn(input: {
      model?: string;
      modelProvider?: string;
      text: string;
      threadId: string;
    }): Promise<CodexTurnStart>;
    interruptTurn(input: CodexTurnStart & CodexThreadStart): Promise<void>;
    archiveThread(threadId: string): Promise<void>;
    login(): Promise<CodexLoginStart>;
    logout(): Promise<void>;
    onAccount(listener: (account: CodexAccount) => void): () => void;
    onLoginCompleted(
      listener: (completion: CodexLoginCompletion) => void,
    ): () => void;
    onRuntimeEvent(listener: (event: CodexRuntimeEvent) => void): () => void;
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
