import { app, BrowserWindow, dialog, ipcMain, shell } from "electron";
import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";

type RuntimeState = "stopped" | "starting" | "ready" | "error";

interface RuntimeStatus {
  state: RuntimeState;
  detail: string;
}

interface PendingRequest {
  reject(error: Error): void;
  resolve(result: unknown): void;
}

interface CodexAccount {
  authMode: string | null;
  email: string | null;
  planType: string | null;
}

interface CodexLoginStart {
  loginId: string;
}

class AppServerRuntime {
  private child?: ChildProcessWithoutNullStreams;
  private nextRequestId = 1;
  private pendingRequests = new Map<number, PendingRequest>();
  private stdoutBuffer = "";
  private runtimeStatus: RuntimeStatus = {
    state: "stopped",
    detail: "Local runtime has not been started.",
  };

  status(): RuntimeStatus {
    return this.runtimeStatus;
  }

  async start(): Promise<RuntimeStatus> {
    if (
      this.runtimeStatus.state === "ready" ||
      this.runtimeStatus.state === "starting"
    ) {
      return this.runtimeStatus;
    }

    const executable = this.resolveExecutable();
    if (!existsSync(executable)) {
      this.setStatus({
        state: "error",
        detail: `Codex runtime was not found at ${executable}. Build it first or set AIVO_CODEX_BIN.`,
      });
      return this.runtimeStatus;
    }

    this.setStatus({
      state: "starting",
      detail: "Starting the local Codex app-server…",
    });

    try {
      const child = spawn(executable, ["app-server", "--stdio"], {
        stdio: ["pipe", "pipe", "pipe"],
        windowsHide: true,
      });
      this.child = child;
      child.stdout.setEncoding("utf8");
      child.stderr.setEncoding("utf8");
      child.stdout.on("data", (chunk: string) => this.handleStdout(chunk));
      child.stderr.on("data", (chunk: string) => {
        console.warn("app-server stderr:", chunk.trim());
      });
      child.once("error", (error) => {
        this.handleExit(`Could not start the local runtime: ${error.message}`);
      });
      child.once("exit", (code, signal) => {
        this.handleExit(
          `Local runtime stopped${code === null ? "" : ` (exit ${code})`}${signal ? ` (${signal})` : ""}.`,
        );
      });

      await this.request("initialize", {
        clientInfo: {
          name: "aivo_desktop",
          title: "Aivo Desktop",
          version: app.getVersion(),
        },
      });
      this.notify("initialized", {});
      this.setStatus({
        state: "ready",
        detail: "Local Codex app-server is ready.",
      });
    } catch (error) {
      this.stop();
      this.setStatus({
        state: "error",
        detail:
          error instanceof Error
            ? error.message
            : "Could not initialize the local runtime.",
      });
    }

    return this.runtimeStatus;
  }

  async stop(): Promise<RuntimeStatus> {
    const child = this.child;
    this.child = undefined;
    this.rejectPending(new Error("Local runtime stopped."));
    if (child && !child.killed) {
      child.kill();
    }
    this.setStatus({ state: "stopped", detail: "Local runtime stopped." });
    return this.runtimeStatus;
  }

  async readAccount(): Promise<CodexAccount> {
    await this.start();
    const result = await this.request("account/read", { refreshToken: false });
    const account = isRecord(result) && isRecord(result.account) ? result.account : null;
    const accountType = stringOrNull(account?.type);

    return {
      authMode: accountType,
      email: accountType === "chatgpt" ? stringOrNull(account?.email) : null,
      planType: accountType === "chatgpt" ? stringOrNull(account?.planType) : null,
    };
  }

  async loginWithChatGpt(): Promise<CodexLoginStart> {
    await this.start();
    const result = await this.request("account/login/start", {
      type: "chatgpt",
      useHostedLoginSuccessPage: true,
      appBrand: "codex",
    });
    if (!isRecord(result) || result.type !== "chatgpt") {
      throw new Error("The local Codex runtime returned an unexpected login response.");
    }

    const loginId = stringOrNull(result.loginId);
    const authUrl = stringOrNull(result.authUrl);
    if (!loginId || !authUrl) {
      throw new Error("The local Codex runtime did not return a login URL.");
    }

    await shell.openExternal(authUrl);
    return { loginId };
  }

  async cancelLogin(loginId: string): Promise<void> {
    await this.request("account/login/cancel", { loginId });
  }

  async logout(): Promise<void> {
    await this.request("account/logout", {});
  }

  private resolveExecutable(): string {
    if (process.env.AIVO_CODEX_BIN) {
      return process.env.AIVO_CODEX_BIN;
    }

    if (app.isPackaged) {
      return path.join(process.resourcesPath, "bin", "codex");
    }

    return path.resolve(
      app.getAppPath(),
      "..",
      "..",
      "codex-rs",
      "target",
      "debug",
      "codex",
    );
  }

  private request(method: string, params: unknown): Promise<unknown> {
    const child = this.child;
    if (!child || !child.stdin.writable) {
      return Promise.reject(new Error("Local runtime is not available."));
    }

    return new Promise((resolve, reject) => {
      const id = this.nextRequestId++;
      this.pendingRequests.set(id, { resolve, reject });
      child.stdin.write(`${JSON.stringify({ id, method, params })}\n`);
    });
  }

  private notify(method: string, params: unknown): void {
    if (this.child?.stdin.writable) {
      this.child.stdin.write(`${JSON.stringify({ method, params })}\n`);
    }
  }

  private handleStdout(chunk: string): void {
    this.stdoutBuffer += chunk;
    const lines = this.stdoutBuffer.split("\n");
    this.stdoutBuffer = lines.pop() ?? "";

    for (const line of lines) {
      if (!line.trim()) {
        continue;
      }

      try {
        const message = JSON.parse(line) as {
          id?: number;
          error?: { message?: string };
          method?: string;
          params?: unknown;
          result?: unknown;
        };
        if (typeof message.id === "number") {
          const pending = this.pendingRequests.get(message.id);
          if (!pending) {
            continue;
          }
          this.pendingRequests.delete(message.id);
          if (message.error) {
            pending.reject(
              new Error(message.error.message ?? "app-server request failed."),
            );
          } else {
            pending.resolve(message.result);
          }
          continue;
        }

        this.handleNotification(message.method, message.params);
      } catch {
        console.warn("Ignoring malformed app-server output.");
      }
    }
  }

  private handleExit(detail: string): void {
    this.child = undefined;
    this.rejectPending(new Error(detail));
    if (this.runtimeStatus.state !== "stopped") {
      this.setStatus({ state: "error", detail });
    }
  }

  private handleNotification(method: string | undefined, params: unknown): void {
    if (method === "account/updated") {
      const payload = isRecord(params) ? params : {};
      this.sendToWindows("account:updated", {
        authMode: stringOrNull(payload.authMode),
        email: null,
        planType: stringOrNull(payload.planType),
      } satisfies CodexAccount);
      return;
    }

    if (method === "account/login/completed") {
      const payload = isRecord(params) ? params : {};
      const completion = {
        error: stringOrNull(payload.error),
        loginId: stringOrNull(payload.loginId),
        success: payload.success === true,
      };
      this.sendToWindows("account:login-completed", completion);
      if (completion.success) {
        void this.publishCurrentAccount();
      }
    }
  }

  private async publishCurrentAccount(): Promise<void> {
    try {
      this.sendToWindows("account:updated", await this.readAccount());
    } catch (error) {
      console.warn(
        "Could not refresh the local Codex account after login:",
        error instanceof Error ? error.message : "unknown error",
      );
    }
  }

  private rejectPending(error: Error): void {
    for (const pending of this.pendingRequests.values()) {
      pending.reject(error);
    }
    this.pendingRequests.clear();
  }

  private setStatus(status: RuntimeStatus): void {
    this.runtimeStatus = status;
    this.sendToWindows("runtime:status", status);
  }

  private sendToWindows(channel: string, payload: unknown): void {
    for (const window of BrowserWindow.getAllWindows()) {
      window.webContents.send(channel, payload);
    }
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const stringOrNull = (value: unknown): string | null =>
  typeof value === "string" ? value : null;

const runtime = new AppServerRuntime();
const devServerURL = process.env.VITE_DEV_SERVER_URL;

const createWindow = async (): Promise<void> => {
  const window = new BrowserWindow({
    width: 1180,
    height: 760,
    minWidth: 900,
    minHeight: 600,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, "../preload/preload.cjs"),
      sandbox: true,
    },
  });

  if (devServerURL) {
    await window.loadURL(devServerURL);
    return;
  }

  await window.loadFile(path.join(__dirname, "../renderer/index.html"));
};

app.whenReady().then(async () => {
  ipcMain.handle("runtime:get-status", () => runtime.status());
  ipcMain.handle("runtime:start", () => runtime.start());
  ipcMain.handle("runtime:stop", () => runtime.stop());
  ipcMain.handle("account:read", () => runtime.readAccount());
  ipcMain.handle("account:login", () => runtime.loginWithChatGpt());
  ipcMain.handle("account:cancel-login", (_event, loginId: string) =>
    runtime.cancelLogin(loginId),
  );
  ipcMain.handle("account:logout", () => runtime.logout());
  ipcMain.handle("workspace:choose", async () => {
    const result = await dialog.showOpenDialog({
      properties: ["openDirectory", "createDirectory"],
      title: "选择 Aivo 工作目录",
    });
    return result.canceled ? null : (result.filePaths[0] ?? null);
  });

  await createWindow();

  app.on("activate", async () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      await createWindow();
    }
  });
});

app.on("window-all-closed", async () => {
  await runtime.stop();
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("before-quit", () => {
  void runtime.stop();
});
