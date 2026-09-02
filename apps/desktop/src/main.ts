import { app, BrowserWindow, ipcMain } from "electron";
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

    const id = this.nextRequestId++;
    const message = JSON.stringify({ id, method, params });
    child.stdin.write(`${message}\n`);

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });
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
        }
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

  private rejectPending(error: Error): void {
    for (const pending of this.pendingRequests.values()) {
      pending.reject(error);
    }
    this.pendingRequests.clear();
  }

  private setStatus(status: RuntimeStatus): void {
    this.runtimeStatus = status;
    for (const window of BrowserWindow.getAllWindows()) {
      window.webContents.send("runtime:status", status);
    }
  }
}

const runtime = new AppServerRuntime();

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

  await window.loadFile(path.join(__dirname, "../renderer/index.html"));
};

app.whenReady().then(async () => {
  ipcMain.handle("runtime:get-status", () => runtime.status());
  ipcMain.handle("runtime:start", () => runtime.start());
  ipcMain.handle("runtime:stop", () => runtime.stop());

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
