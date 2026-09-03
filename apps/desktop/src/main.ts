import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  type IpcMainInvokeEvent,
  Menu,
  safeStorage,
  shell,
} from "electron";
import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import {
  createDesktopUpdater,
  type DesktopUpdateState,
  type DesktopUpdater,
} from "./desktop-updater.cjs";

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

class AppServerRuntime {
  private child?: ChildProcessWithoutNullStreams;
  private nextRequestId = 1;
  private pendingRequests = new Map<number, PendingRequest>();
  private stdoutBuffer = "";
  private runtimeStatus: RuntimeStatus = {
    state: "stopped",
    detail: "Local runtime has not been started.",
  };

  constructor(
    private readonly commandPrefix: string[] = [],
    private readonly publishEvents = true,
  ) {}

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
      const child = spawn(
        executable,
        [...this.commandPrefix, "app-server", "--stdio"],
        {
          env: this.runtimeEnvironment(),
          stdio: ["pipe", "pipe", "pipe"],
          windowsHide: true,
        },
      );
      this.child = child;
      child.stdout.setEncoding("utf8");
      child.stderr.setEncoding("utf8");
      child.stdout.on("data", (chunk: string) => this.handleStdout(chunk));
      child.stderr.on("data", (chunk: string) => {
        console.warn("app-server stderr:", chunk.trim());
      });
      child.once("error", (error) => {
        this.handleExit(child, `Could not start the local runtime: ${error.message}`);
      });
      child.once("exit", (code, signal) => {
        this.handleExit(
          child,
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
    if (child && child.exitCode === null && child.signalCode === null) {
      await new Promise<void>((resolve) => {
        child.once("error", resolve);
        child.once("exit", resolve);
        child.kill();
      });
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

  async listModels(): Promise<CodexModel[]> {
    await this.start();
    const models: CodexModel[] = [];
    let cursor: string | null = null;

    do {
      const result = await this.request("model/list", {
        cursor,
        includeHidden: false,
        limit: 100,
      });
      if (!isRecord(result)) {
        throw new Error("The local Codex runtime returned an unexpected model list.");
      }

      const data = Array.isArray(result.data) ? result.data : [];
      for (const item of data) {
        if (!isRecord(item)) continue;
        const id = stringOrNull(item.model);
        if (!id) continue;
        models.push({
          id,
          name: stringOrNull(item.displayName) ?? id,
          description: stringOrNull(item.description) ?? "",
        });
      }
      cursor = stringOrNull(result.nextCursor);
    } while (cursor);

    return models;
  }

  async configureProvider({
    apiKey,
    baseUrl,
    model,
    name,
    providerId,
  }: BackendProviderConnectionInput): Promise<void> {
    const normalizedProviderId = normalizeBackendProviderId(providerId);
    const normalizedApiKey = normalizeProviderApiKey(normalizedProviderId, apiKey);
    const normalizedBaseUrl = baseUrl.trim();
    const trimmedModel = model.trim();
    const normalizedName = name.trim() || normalizedProviderId;
    if (!normalizedBaseUrl || !trimmedModel) {
      throw new Error("请提供 Provider 的 API 地址和模型 ID。");
    }

    saveProviderApiKey(normalizedProviderId, normalizedApiKey);
    const initialStatus = await this.start();
    if (initialStatus.state !== "ready") {
      throw new Error(initialStatus.detail);
    }
    await this.request("config/batchWrite", {
      edits: [
        configEdit(
          `model_providers.${normalizedProviderId}.name`,
          normalizedName,
        ),
        configEdit(
          `model_providers.${normalizedProviderId}.base_url`,
          normalizedBaseUrl,
        ),
        configEdit(
          `model_providers.${normalizedProviderId}.env_key`,
          providerApiKeyEnvironmentName(normalizedProviderId),
        ),
        configEdit(
          `model_providers.${normalizedProviderId}.wire_api`,
          "responses",
        ),
        configEdit(
          `model_providers.${normalizedProviderId}.requires_openai_auth`,
          false,
        ),
        configEdit(
          `model_providers.${normalizedProviderId}.supports_websockets`,
          false,
        ),
        ...(normalizedProviderId === "volcengine-agent-plan"
          ? [
              configEdit(
                `model_providers.${normalizedProviderId}.request_max_retries`,
                0,
              ),
              configEdit(
                `model_providers.${normalizedProviderId}.stream_max_retries`,
                0,
              ),
              configEdit(
                `model_providers.${normalizedProviderId}.stream_idle_timeout_ms`,
                45_000,
              ),
            ]
          : []),
      ],
      reloadUserConfig: false,
    });

    await this.stop();
    const status = await this.start();
    if (status.state !== "ready") {
      throw new Error(status.detail);
    }
  }

  async deleteProvider(providerId: string): Promise<void> {
    const normalizedProviderId = normalizeBackendProviderId(providerId);
    removeProviderApiKey(normalizedProviderId);
    const status = await this.start();
    if (status.state !== "ready") {
      throw new Error(status.detail);
    }
    await this.request("config/batchWrite", {
      edits: [configEdit(`model_providers.${normalizedProviderId}`, null)],
      reloadUserConfig: false,
    });
    await this.stop();
    const restarted = await this.start();
    if (restarted.state !== "ready") {
      throw new Error(restarted.detail);
    }
  }

  async startThread({
    cwd,
    model,
    modelProvider,
  }: {
    cwd?: string;
    model?: string;
    modelProvider?: string;
  }): Promise<CodexThreadStart> {
    await this.start();
    const result = await this.request("thread/start", {
      cwd: cwd || undefined,
      model: model || undefined,
      modelProvider: modelProvider || undefined,
    });
    const thread = isRecord(result) && isRecord(result.thread) ? result.thread : null;
    const threadId = stringOrNull(thread?.id);
    if (!threadId) {
      throw new Error("The local Codex runtime did not return a thread id.");
    }
    return { threadId };
  }

  async listThreads(limit: number): Promise<CodexThread[]> {
    await this.start();
    const result = await this.request("thread/list", {
      limit,
      sortKey: "recency_at",
      sortDirection: "desc",
      sourceKinds: [
        "cli",
        "vscode",
        "exec",
        "appServer",
        "subAgent",
        "subAgentReview",
        "subAgentCompact",
        "subAgentThreadSpawn",
        "subAgentOther",
        "unknown",
      ],
    });
    if (!isRecord(result) || !Array.isArray(result.data)) {
      throw new Error("The local Codex runtime returned an unexpected thread list.");
    }
    return result.data.flatMap((thread) => {
      if (!isRecord(thread)) return [];
      const id = stringOrNull(thread.id);
      const cwd = stringOrNull(thread.cwd);
      if (!id || !cwd) return [];

      return [{
        id,
        cwd,
        model: stringOrNull(thread.model),
        modelProvider: stringOrNull(thread.modelProvider) ?? "openai",
        name: stringOrNull(thread.name),
        parentThreadId: stringOrNull(thread.parentThreadId),
        preview: stringOrNull(thread.preview) ?? "",
        source: stringOrNull(thread.source) ?? "codex",
        status: isRecord(thread.status)
          ? stringOrNull(thread.status.type) ?? "notLoaded"
          : "notLoaded",
        timeCreated: unixSecondsToIso(thread.createdAt),
        timeUpdated: unixSecondsToIso(thread.recencyAt ?? thread.updatedAt ?? thread.createdAt),
      } satisfies CodexThread];
    });
  }

  async archiveThread(threadId: string): Promise<void> {
    await this.start();
    await this.request("thread/archive", { threadId });
  }

  async resumeThread(threadId: string): Promise<void> {
    await this.start();
    await this.request("thread/resume", { threadId, excludeTurns: true });
  }

  async listThreadTurns(threadId: string): Promise<CodexThreadTurn[]> {
    await this.start();
    const result = await this.request("thread/turns/list", {
      threadId,
      limit: 100,
      sortDirection: "asc",
      itemsView: "full",
    });
    if (!isRecord(result) || !Array.isArray(result.data)) {
      throw new Error("The local Codex runtime returned an unexpected turn list.");
    }
    return result.data.flatMap((turn) => {
      if (!isRecord(turn)) return [];
      const id = stringOrNull(turn.id);
      if (!id) return [];
      const error = isRecord(turn.error) ? stringOrNull(turn.error.message) : null;
      return [{
        completedAt: unixSecondsToIsoOrNull(turn.completedAt),
        durationMs:
          typeof turn.durationMs === "number" && Number.isFinite(turn.durationMs)
            ? turn.durationMs
            : null,
        error,
        id,
        items: Array.isArray(turn.items) ? turn.items : [],
        startedAt: unixSecondsToIsoOrNull(turn.startedAt),
        status: isRecord(turn.status)
          ? stringOrNull(turn.status.type) ?? "unknown"
          : "unknown",
      } satisfies CodexThreadTurn];
    });
  }

  async startTurn({
    model,
    text,
    threadId,
  }: {
    model?: string;
    text: string;
    threadId: string;
  }): Promise<CodexTurnStart> {
    await this.start();
    const result = await this.request("turn/start", {
      threadId,
      model: model || undefined,
      input: [{ type: "text", text, textElements: [] }],
    });
    const turn = isRecord(result) && isRecord(result.turn) ? result.turn : null;
    const turnId = stringOrNull(turn?.id);
    if (!turnId) {
      throw new Error("The local Codex runtime did not return a turn id.");
    }
    return { turnId };
  }

  async interruptTurn({ threadId, turnId }: CodexTurnStart & CodexThreadStart): Promise<void> {
    await this.request("turn/interrupt", { threadId, turnId });
  }

  private runtimeEnvironment(): NodeJS.ProcessEnv {
    const credentials = readProviderApiKeys();
    const environment = { ...process.env };
    for (const [providerId, apiKey] of Object.entries(credentials)) {
      environment[providerApiKeyEnvironmentName(providerId)] = apiKey;
    }
    return environment;
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

  private handleExit(child: ChildProcessWithoutNullStreams, detail: string): void {
    if (this.child !== child) return;
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
      return;
    }

    if (
      method === "item/agentMessage/delta" ||
      method === "item/started" ||
      method === "item/completed" ||
      method === "item/commandExecution/outputDelta" ||
      method === "turn/completed"
    ) {
      this.sendToWindows("codex:event", {
        method,
        params,
      } satisfies CodexRuntimeEvent);
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
    if (!this.publishEvents) return;
    for (const window of BrowserWindow.getAllWindows()) {
      window.webContents.send(channel, payload);
    }
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const stringOrNull = (value: unknown): string | null =>
  typeof value === "string" ? value : null;

const unixSecondsToIso = (value: unknown): string =>
  typeof value === "number" && Number.isFinite(value)
    ? new Date(value * 1000).toISOString()
    : new Date(0).toISOString();

const unixSecondsToIsoOrNull = (value: unknown): string | null =>
  typeof value === "number" && Number.isFinite(value)
    ? new Date(value * 1000).toISOString()
    : null;

const runtime = new AppServerRuntime();
const codexModelsRuntime = new AppServerRuntime(
  ["-c", 'model_provider="openai"'],
  /*publishEvents*/ false,
);
let codexModelsRequest: Promise<CodexModel[]> | undefined;

function listCurrentCodexModels(): Promise<CodexModel[]> {
  if (codexModelsRequest) return codexModelsRequest;

  const request = (async () => {
    try {
      return await codexModelsRuntime.listModels();
    } finally {
      await codexModelsRuntime.stop();
    }
  })();
  codexModelsRequest = request;
  void request.then(
    () => {
      if (codexModelsRequest === request) {
        codexModelsRequest = undefined;
      }
    },
    () => {
      if (codexModelsRequest === request) {
        codexModelsRequest = undefined;
      }
    },
  );
  return request;
}
const devServerURL = process.env.VITE_DEV_SERVER_URL;
const isMac = process.platform === "darwin";
let desktopUpdater: DesktopUpdater | undefined;

app.setName("Aivo");

function mainWindowForMenu(): BrowserWindow | undefined {
  return BrowserWindow.getFocusedWindow() ?? BrowserWindow.getAllWindows()[0];
}

function requireMainRenderer(event: IpcMainInvokeEvent): void {
  if (!event.senderFrame || event.senderFrame !== event.sender.mainFrame) {
    throw new Error("Desktop capability is available only to the main renderer.");
  }
}

function configureApplicationMenu(): void {
  if (!isMac) {
    Menu.setApplicationMenu(null);
    return;
  }

  Menu.setApplicationMenu(
    Menu.buildFromTemplate([
      {
        label: "Aivo",
        submenu: [
          { role: "about", label: "关于 Aivo" },
          { type: "separator" },
          {
            label: "检查更新…",
            click: () => {
              const window = mainWindowForMenu();
              if (window && desktopUpdater) {
                void checkAndOfferUpdate(window, true);
              }
            },
          },
          { type: "separator" },
          { role: "services", label: "服务" },
          { type: "separator" },
          { role: "hide", label: "隐藏 Aivo" },
          { role: "hideOthers", label: "隐藏其他" },
          { role: "unhide", label: "全部显示" },
          { type: "separator" },
          { role: "quit", label: "退出 Aivo" },
        ],
      },
      { label: "文件", submenu: [{ role: "close", label: "关闭窗口" }] },
      {
        label: "编辑",
        submenu: [
          { role: "undo", label: "撤销" },
          { role: "redo", label: "重做" },
          { type: "separator" },
          { role: "cut", label: "剪切" },
          { role: "copy", label: "复制" },
          { role: "paste", label: "粘贴" },
          { role: "delete", label: "删除" },
          { role: "selectAll", label: "全选" },
        ],
      },
      {
        label: "视图",
        submenu: [
          { role: "reload", label: "重新加载" },
          { role: "forceReload", label: "强制重新加载" },
          { role: "toggleDevTools", label: "切换开发者工具" },
          { type: "separator" },
          { role: "resetZoom", label: "实际大小" },
          { role: "zoomIn", label: "放大" },
          { role: "zoomOut", label: "缩小" },
          { type: "separator" },
          { role: "togglefullscreen", label: "切换全屏" },
        ],
      },
      {
        label: "窗口",
        submenu: [
          { role: "minimize", label: "最小化" },
          { role: "zoom", label: "缩放" },
          { type: "separator" },
          { role: "front", label: "全部置于最前面" },
        ],
      },
    ]),
  );
}

function publishUpdateState(state: DesktopUpdateState): void {
  for (const window of BrowserWindow.getAllWindows()) {
    if (!window.isDestroyed()) window.webContents.send("update:state", state);
  }
}

async function checkAndOfferUpdate(
  window: BrowserWindow,
  reportCurrent = false,
): Promise<void> {
  if (!desktopUpdater) return;
  const state = await desktopUpdater.check();
  if (window.isDestroyed()) return;

  if (state.phase !== "available") {
    if (!reportCurrent) return;
    await dialog.showMessageBox(window, {
      type: state.phase === "error" ? "error" : "info",
      title: state.phase === "error" ? "无法检查 Aivo 更新" : "Aivo 软件更新",
      message:
        state.phase === "up-to-date"
          ? `Aivo v${state.currentVersion} 已是最新版本`
          : state.message,
      buttons: ["好"],
      defaultId: 0,
      noLink: true,
    });
    return;
  }

  const offer = await dialog.showMessageBox(window, {
    type: "info",
    title: "Aivo 更新可用",
    message: `发现 Aivo v${state.availableVersion}`,
    detail: "下载后会校验发布记录和 SHA-256，安装仍会显示 macOS 安全提示。",
    buttons: ["下载更新", "稍后"],
    defaultId: 0,
    cancelId: 1,
    noLink: true,
  });
  if (offer.response !== 0) return;

  const downloaded = await desktopUpdater.download();
  if (downloaded.phase !== "ready" || window.isDestroyed()) return;
  const ready = await dialog.showMessageBox(window, {
    type: "info",
    title: "Aivo 更新已验证",
    message: `Aivo v${downloaded.availableVersion} 已准备好`,
    detail: "将打开已验证的更新包；请按 macOS 提示完成安装。",
    buttons: ["打开安装包", "稍后"],
    defaultId: 0,
    cancelId: 1,
    noLink: true,
  });
  if (ready.response === 0) await desktopUpdater.install();
}

const createWindow = async (): Promise<BrowserWindow> => {
  const window = new BrowserWindow({
    width: 1180,
    height: 760,
    minWidth: 900,
    minHeight: 600,
    backgroundColor: "#ffffff",
    title: "Aivo",
    frame: true,
    ...(isMac
      ? {
          titleBarStyle: "hidden" as const,
          trafficLightPosition: { x: 10, y: 10 },
        }
      : {}),
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, "../preload/preload.cjs"),
      sandbox: true,
    },
  });

  if (devServerURL) {
    await window.loadURL(devServerURL);
    return window;
  }

  await window.loadFile(path.join(__dirname, "../renderer/index.html"));
  return window;
};

app.whenReady().then(async () => {
  ipcMain.handle("runtime:get-status", () => runtime.status());
  ipcMain.handle("runtime:start", () => runtime.start());
  ipcMain.handle("runtime:stop", () => runtime.stop());
  ipcMain.handle("account:read", () => runtime.readAccount());
  ipcMain.handle(
    "provider:configure",
    (event, input: BackendProviderConnectionInput) => {
      requireMainRenderer(event);
      return runtime.configureProvider(input);
    },
  );
  ipcMain.handle("provider:delete", (event, providerId: string) => {
    requireMainRenderer(event);
    return runtime.deleteProvider(providerId);
  });
  ipcMain.handle("models:list", () => runtime.listModels());
  ipcMain.handle("models:codex:list", () => listCurrentCodexModels());
  ipcMain.handle("threads:list", (event, limit: number) => {
    requireMainRenderer(event);
    return runtime.listThreads(limit);
  });
  ipcMain.handle("thread:turns:list", (event, threadId: string) => {
    requireMainRenderer(event);
    return runtime.listThreadTurns(threadId);
  });
  ipcMain.handle("thread:resume", (event, threadId: string) => {
    requireMainRenderer(event);
    return runtime.resumeThread(threadId);
  });
  ipcMain.handle("thread:start", (event, input: { cwd?: string; model?: string; modelProvider?: string }) => {
    requireMainRenderer(event);
    return runtime.startThread(input);
  });
  ipcMain.handle(
    "turn:start",
    (event, input: { model?: string; text: string; threadId: string }) => {
      requireMainRenderer(event);
      return runtime.startTurn(input);
    },
  );
  ipcMain.handle(
    "turn:interrupt",
    (event, input: CodexTurnStart & CodexThreadStart) => {
      requireMainRenderer(event);
      return runtime.interruptTurn(input);
    },
  );
  ipcMain.handle("thread:archive", (event, threadId: string) => {
    requireMainRenderer(event);
    return runtime.archiveThread(threadId);
  });
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
  ipcMain.handle("window:toggle-maximize", (event) => {
    requireMainRenderer(event);
    const window = BrowserWindow.fromWebContents(event.sender);
    if (!window) return;
    if (window.isMaximized()) {
      window.unmaximize();
    } else {
      window.maximize();
    }
  });
  ipcMain.handle("update:get-state", (event) => {
    requireMainRenderer(event);
    return desktopUpdater?.getState();
  });
  ipcMain.handle("update:check", (event) => {
    requireMainRenderer(event);
    return desktopUpdater?.check();
  });
  ipcMain.handle("update:download", (event) => {
    requireMainRenderer(event);
    return desktopUpdater?.download();
  });
  ipcMain.handle("update:install", (event) => {
    requireMainRenderer(event);
    return desktopUpdater?.install();
  });
  ipcMain.handle("update:cancel", (event) => {
    requireMainRenderer(event);
    return desktopUpdater?.cancel();
  });

  desktopUpdater = createDesktopUpdater({
    appVersion: app.getVersion(),
    arch: process.arch,
    isPackaged: app.isPackaged,
    platform: process.platform,
    shell,
    tempRoot: app.getPath("temp"),
    onState: publishUpdateState,
  });
  configureApplicationMenu();

  const window = await createWindow();
  if (app.isPackaged) {
    window.webContents.once("did-finish-load", () => {
      setTimeout(() => void checkAndOfferUpdate(window), 3000);
    });
  }

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
  desktopUpdater?.dispose();
  void runtime.stop();
  void codexModelsRuntime.stop();
});

const PROVIDER_CREDENTIALS_FILE = "provider-credentials";
const VOLCENGINE_AGENT_PLAN_SECRET_FILE = "volcengine-agent-plan-secret";

function configEdit(keyPath: string, value: boolean | number | string | null) {
  return { keyPath, mergeStrategy: "replace", value };
}

function providerCredentialsPath() {
  return path.join(app.getPath("userData"), PROVIDER_CREDENTIALS_FILE);
}

function readProviderApiKeys(): Record<string, string> {
  const credentialsPath = providerCredentialsPath();
  if (!existsSync(credentialsPath)) return migrateVolcengineAgentPlanCredential();
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("系统安全存储不可用，无法读取 Provider API Key。");
  }
  const decrypted = safeStorage.decryptString(readFileSync(credentialsPath));
  try {
    const parsed = JSON.parse(decrypted) as unknown;
    if (!isRecord(parsed)) throw new Error("invalid provider credentials");
    return Object.fromEntries(
      Object.entries(parsed).flatMap(([providerId, apiKey]) => {
        if (typeof apiKey !== "string" || !apiKey.trim()) return [];
        return [[normalizeBackendProviderId(providerId), normalizeProviderApiKey(providerId, apiKey)]];
      }),
    );
  } catch {
    throw new Error("已保存的 Provider 凭据不可读取，请在设置中重新连接该 Provider。");
  }
}

function saveProviderApiKey(providerId: string, apiKey: string) {
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("系统安全存储不可用，无法保存 Provider API Key。");
  }
  mkdirSync(app.getPath("userData"), { recursive: true });
  const credentials = readProviderApiKeys();
  credentials[providerId] = apiKey;
  writeFileSync(
    providerCredentialsPath(),
    safeStorage.encryptString(JSON.stringify(credentials)),
  );
}

function removeProviderApiKey(providerId: string) {
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("系统安全存储不可用，无法删除 Provider API Key。");
  }
  const credentials = readProviderApiKeys();
  if (!(providerId in credentials)) return;
  delete credentials[providerId];
  writeFileSync(
    providerCredentialsPath(),
    safeStorage.encryptString(JSON.stringify(credentials)),
  );
}

function migrateVolcengineAgentPlanCredential(): Record<string, string> {
  const legacyPath = path.join(app.getPath("userData"), VOLCENGINE_AGENT_PLAN_SECRET_FILE);
  if (!existsSync(legacyPath)) return {};
  if (!safeStorage.isEncryptionAvailable()) {
    throw new Error("系统安全存储不可用，无法读取 Provider API Key。");
  }
  const apiKey = normalizeProviderApiKey(
    "volcengine-agent-plan",
    safeStorage.decryptString(readFileSync(legacyPath)),
  );
  return { "volcengine-agent-plan": apiKey };
}

function normalizeBackendProviderId(providerId: string) {
  const normalized = providerId.trim().toLowerCase();
  if (!/^[a-z][a-z0-9-]*$/.test(normalized)) {
    throw new Error("Provider ID 只能包含小写字母、数字和连字符，且必须以字母开头。");
  }
  if (normalized === "openai") {
    throw new Error("OpenAI 请使用浏览器登录或其专用 API Key 连接方式。");
  }
  return normalized;
}

function providerApiKeyEnvironmentName(providerId: string) {
  return `AIVO_PROVIDER_${providerId.toUpperCase().replace(/-/g, "_")}_API_KEY`;
}

function normalizeProviderApiKey(providerId: string, apiKey: string) {
  if (providerId !== "volcengine-agent-plan") return apiKey.trim();

  let normalized = apiKey.trim();
  const assignment = normalized.match(/^(?:export\s+)?ARK_API_KEY\s*=\s*(.+)$/i);
  if (assignment) normalized = assignment[1].trim();

  const bearer = normalized.match(/^Bearer\s+(.+)$/i);
  if (bearer) normalized = bearer[1].trim();

  if (
    (normalized.startsWith('"') && normalized.endsWith('"')) ||
    (normalized.startsWith("'") && normalized.endsWith("'"))
  ) {
    normalized = normalized.slice(1, -1).trim();
  }

  if (!normalized || /\s/.test(normalized)) {
    throw new Error(
      "请输入火山方舟 Agent Plan 页面生成的 ARK_API_KEY 原始值；不要粘贴 Access Key、Secret Key 或多行配置。",
    );
  }

  return normalized;
}
