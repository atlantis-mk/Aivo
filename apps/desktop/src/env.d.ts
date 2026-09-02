interface RuntimeStatus {
  state: "stopped" | "starting" | "ready" | "error";
  detail: string;
}

interface AivoDesktopApi {
  runtime: {
    getStatus(): Promise<RuntimeStatus>;
    start(): Promise<RuntimeStatus>;
    stop(): Promise<RuntimeStatus>;
    onStatus(listener: (status: RuntimeStatus) => void): () => void;
  };
}

interface Window {
  aivoDesktop: AivoDesktopApi;
}
