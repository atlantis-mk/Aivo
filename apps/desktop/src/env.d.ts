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

interface CodexLoginCompletion {
  error: string | null;
  loginId: string | null;
  success: boolean;
}

interface AivoDesktopApi {
  runtime: {
    getStatus(): Promise<RuntimeStatus>;
    start(): Promise<RuntimeStatus>;
    stop(): Promise<RuntimeStatus>;
    onStatus(listener: (status: RuntimeStatus) => void): () => void;
  };
  codex: {
    cancelLogin(loginId: string): Promise<void>;
    getAccount(): Promise<CodexAccount>;
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
}

interface Window {
  aivoDesktop: AivoDesktopApi;
}
