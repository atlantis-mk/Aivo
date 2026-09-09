type CodexApprovalRequest = import("./codex-app-server").CodexApprovalRequest;
type CodexResourceInput = import("./codex-composer-resources").CodexResourceInput;
type CodexSkillCatalog = import("./codex-composer-resources").CodexSkillCatalog;
type CodexMcpServer = import("./codex-composer-resources").CodexMcpServer;
type ComposerLocalSelection = import("./services/aivo/project-service").ComposerLocalSelection;

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

interface CodexModelConfig {
  model: string | null;
  modelProvider: string | null;
  reasoningEffort: string | null;
  serviceTier: string | null;
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
    readModelConfig(): Promise<CodexModelConfig>;
    saveModelPreferences(input: {
      model: string;
      modelProvider: string;
      reasoningEffort: string;
    }): Promise<void>;
    cancelLogin(loginId: string): Promise<void>;
    getAccount(): Promise<CodexAccount>;
    listCodexModels(): Promise<CodexModel[]>;
    listModels(): Promise<CodexModel[]>;
    listSkills(workspaceRoot?: string, forceReload?: boolean): Promise<CodexSkillCatalog>;
    listMcpServers(): Promise<CodexMcpServer[]>;
    listThreads(limit: number, searchTerm?: string): Promise<CodexThread[]>;
    listThreadTurns(threadId: string): Promise<CodexThreadTurn[]>;
    resumeThread(threadId: string): Promise<void>;
    startThread(input: {
      cwd?: string;
      model?: string;
      modelProvider?: string;
      permissionMode?: "request_approval" | "auto_review" | "full_access";
    }): Promise<CodexThreadStart>;
    startTurn(input: {
      resourceInputs?: CodexResourceInput[];
      images?: Array<{ data: string; mimeType: string }>;
      model?: string;
      modelProvider?: string;
      permissionMode?: "request_approval" | "auto_review" | "full_access";
      text: string;
      textElements?: Array<{
        byteRange: { start: number; end: number };
        placeholder: string;
      }>;
      threadId: string;
    }): Promise<CodexTurnStart>;
    interruptTurn(input: CodexTurnStart & CodexThreadStart): Promise<void>;
    steerTurn(input: {
      resourceInputs?: CodexResourceInput[];
      clientUserMessageId: string;
      expectedTurnId: string;
      text: string;
      threadId: string;
    }): Promise<CodexTurnStart>;
    archiveThread(threadId: string): Promise<void>;
    login(): Promise<CodexLoginStart>;
    logout(): Promise<void>;
    listApprovals(): Promise<CodexApprovalRequest[]>;
    resolveApproval(
      requestId: string,
      action: "approve" | "approveForSession" | "deny" | "cancel",
    ): Promise<void>;
    onAccount(listener: (account: CodexAccount) => void): () => void;
    onLoginCompleted(
      listener: (completion: CodexLoginCompletion) => void,
    ): () => void;
    onRuntimeEvent(listener: (event: CodexRuntimeEvent) => void): () => void;
    onApprovalRequest(
      listener: (request: CodexApprovalRequest) => void,
    ): () => void;
    onApprovalResolved(
      listener: (payload: { requestId: string }) => void,
    ): () => void;
    onUserInputRequest(
      listener: (payload: { request: unknown }) => void,
    ): () => void;
    onUserInputResolved(
      listener: (payload: { itemId: string }) => void,
    ): () => void;
    resolveUserInput(
      itemId: string,
      answers: Record<string, { answers: string[] }>,
    ): Promise<void>;
  };
  workspace: {
    choose(): Promise<string | null>;
  };
  composer: {
    selectLocalResource(): Promise<ComposerLocalSelection | null>;
    inspectDroppedResources(files: File[]): Promise<ComposerLocalSelection[]>;
  };
  file: {
    openPath(target: string): Promise<string>;
    openExternal(target: string): Promise<void>;
    openText(name: string, text: string): Promise<string>;
    pathForFile(file: File): string;
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
