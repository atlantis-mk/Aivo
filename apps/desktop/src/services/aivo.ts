import type { domain } from "../../bridge/go/models";

function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  return window.aivo.invoke<T>(method, ...args);
}

export function getAppConfig() {
  return invoke<domain.AppConfig>("GetAppConfig");
}

export function getProviderCatalog() {
  return invoke<domain.CatalogState>("GetProviderCatalog");
}

export function connectProvider(input: domain.ProviderConnectInput) {
  return invoke<domain.CatalogState>("ConnectProvider", input);
}

export function updateModelPreferences(input: domain.ModelPreferencesInput) {
  return invoke<domain.AppConfig>("UpdateModelPreferences", input);
}

export function refreshProviderModels(input: domain.ProviderConnectInput) {
  return invoke<domain.CatalogState>("RefreshProviderModels", input);
}

export function deleteProviderAccount(accountId: string) {
  return invoke<domain.CatalogState>("DeleteProviderAccount", accountId);
}

export function startProviderAuth(input: domain.ProviderAuthStartInput) {
  return invoke<domain.ProviderAuthStartResult>("StartProviderAuth", input);
}

export function getProviderAuthStatus(providerId: string) {
  return invoke<domain.ProviderAuthStatus>("GetProviderAuthStatus", providerId);
}

export function cancelProviderAuth(providerId: string) {
  return invoke<domain.ProviderAuthStatus>("CancelProviderAuth", providerId);
}

export async function selectProjectDirectory() {
  const selected = await window.aivo.selectProjectDirectory();
  return selected || "";
}

export function exportDiagnostics() {
  return window.aivo.exportDiagnostics();
}

export function listRecentProjects(limit: number) {
  return invoke<domain.AssistantProject[]>("ListRecentProjects", limit);
}

export function upsertProject(path: string) {
  return invoke<domain.AssistantProject>("UpsertProject", path);
}

export function setProjectSidebarHidden(path: string, hidden: boolean) {
  return invoke<domain.AssistantProject>("SetProjectSidebarHidden", path, hidden);
}

export function createSession(input: domain.CreateSessionRequest) {
  return invoke<domain.Session>("CreateSession", input);
}

export function listSessions(limit: number) {
  return invoke<domain.Session[]>("ListSessions", {
    type: "coding",
    status: "active",
    limit,
  } as domain.ListSessionsRequest);
}

export function listSessionToolCalls(sessionId: string) {
  return invoke<domain.ToolCall[]>("ListSessionToolCalls", sessionId);
}

export type ReplaySessionToolCallInput = {
  sessionId?: string;
  toolCallId: string;
  permissionScope?: string;
};

export function replaySessionToolCall(input: ReplaySessionToolCallInput) {
  return invoke<domain.ToolCall>("ReplaySessionToolCall", input);
}

export type RetainedOutputReadInput = {
  ref: string;
  offset?: number;
  limit?: number;
};

export type RetainedOutputReadResult = {
  ref: string;
  content: string;
  offset: number;
  nextOffset: number;
  size: number;
  truncated?: boolean;
};

export function readRetainedOutput(input: RetainedOutputReadInput) {
  return invoke<RetainedOutputReadResult>("ReadRetainedOutput", input);
}

export function listSessionTurns(sessionId: string, limit = 100) {
  return invoke<domain.Turn[]>("ListSessionTurns", sessionId, limit);
}

export type AgentModeId =
  | "code"
  | "assistant"
  | "build"
  | "explore"
  | "plan"
  | "planner"
  | "review"
  | "debug"
  | "summary"
  | "title"
  | "scheduler_worker";

export type AgentModeDefinition = {
  id: AgentModeId;
  displayName: string;
  description: string;
  prompt: string;
  toolsets: string[];
  fileWriteAccess?: boolean;
  commandAccess?: boolean;
  networkAccess?: boolean;
  backgroundTaskAccess?: boolean;
  hidden?: boolean;
};

export type AgentRun = {
  id: string;
  parentSessionId?: string;
  sessionId?: string;
  mode: AgentModeId;
  status: string;
  prompt?: string;
  result?: string;
  error?: string;
  metadata?: Record<string, string>;
  timeCreated: string;
  timeUpdated: string;
  timeCompleted?: string;
};

export type TodoItem = {
  id: string;
  sessionId?: string;
  projectPath?: string;
  title: string;
  status: string;
  ownerMode?: AgentModeId;
  timeCreated: string;
  timeUpdated: string;
};

export type ScheduledJob = {
  id: string;
  sessionId?: string;
  title: string;
  prompt: string;
  schedule: string;
  workerMode: AgentModeId;
  toolsets?: string[];
  permissionScope?: string;
  status: string;
  nextRunAt?: string;
  lastRunAt?: string;
  lastError?: string;
};

export function listAgentModes(includeHidden = false) {
  return invoke<AgentModeDefinition[]>("ListAgentModes", includeHidden);
}

export function setSessionAgentMode(sessionId: string, mode: AgentModeId) {
  return invoke<domain.Session>("SetSessionAgentMode", { sessionId, mode });
}

export function listAgentRuns(sessionId: string, limit = 20) {
  return invoke<AgentRun[]>("ListAgentRuns", { sessionId, limit });
}

export function cancelAgentRun(id: string) {
  return invoke<AgentRun>("CancelAgentRun", id);
}

export function listTodoItems(sessionId: string, projectPath = "", limit = 8) {
  return invoke<TodoItem[]>("ListTodoItems", { sessionId, projectPath, limit });
}

export function listScheduledJobs(sessionId: string, limit = 8) {
  return invoke<ScheduledJob[]>("ListScheduledJobs", { sessionId, limit });
}

export function saveScheduledJob(input: Partial<ScheduledJob>) {
  return invoke<ScheduledJob>("SaveScheduledJob", input);
}

export function deleteScheduledJob(id: string) {
  return invoke<void>("DeleteScheduledJob", id);
}

export type PermissionRequest = {
  id: string;
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  toolName: string;
  action: string;
  paths?: string[];
  arguments?: Record<string, unknown>;
  status: string;
  remember?: boolean;
  reason?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type QuestionOption = {
  label: string;
  description?: string;
};

export type QuestionPrompt = {
  id?: string;
  header?: string;
  question: string;
  options?: QuestionOption[];
  multiple?: boolean;
};

export type QuestionRequest = {
  id: string;
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  toolName: string;
  questions: QuestionPrompt[];
  answers?: string[][];
  status: string;
  reason?: string;
  arguments?: Record<string, unknown>;
  timeCreated: string;
  timeUpdated: string;
};

export type PermissionMode =
  "request_approval" | "auto_approve" | "full_access";

export type PermissionModeState = {
  sessionId?: string;
  workspaceRoot?: string;
  mode: PermissionMode;
  timeUpdated?: string;
};

export function listPermissionRequests(sessionId: string, status = "pending") {
  return invoke<PermissionRequest[]>(
    "ListPermissionRequests",
    sessionId,
    status,
  );
}

export function getPermissionMode(sessionId: string) {
  return invoke<PermissionModeState>("GetPermissionMode", sessionId);
}

export function getCodingContext(sessionId: string) {
  return invoke<domain.CodingContext>("GetCodingContext", sessionId);
}

export function setPermissionMode(sessionId: string, mode: PermissionMode) {
  return invoke<PermissionModeState>("SetPermissionMode", { sessionId, mode });
}

export function approvePermissionRequest(requestId: string, remember = false) {
  return invoke<PermissionRequest>("ApprovePermissionRequest", {
    requestId,
    remember,
  });
}

export function denyPermissionRequest(
  requestId: string,
  remember = false,
  reason = "",
) {
  return invoke<PermissionRequest>("DenyPermissionRequest", {
    requestId,
    remember,
    reason,
  });
}

export function listQuestionRequests(sessionId: string, status = "pending") {
  return invoke<QuestionRequest[]>("ListQuestionRequests", sessionId, status);
}

export function replyQuestionRequest(requestId: string, answers: string[][]) {
  return invoke<QuestionRequest>("ReplyQuestionRequest", {
    requestId,
    answers,
  });
}

export function rejectQuestionRequest(requestId: string, reason = "") {
  return invoke<QuestionRequest>("RejectQuestionRequest", {
    requestId,
    reason,
  });
}

export function archiveSession(sessionId: string) {
  return invoke<domain.Session>("ArchiveSession", sessionId);
}

export function submitSessionMessage(
  input: domain.SubmitSessionMessageRequest,
) {
  return invoke<domain.PreparedSessionTurn>(
    "SubmitSessionMessageStreaming",
    input,
  );
}

export function cancelSessionTurn(input: domain.CancelTurnRequest) {
  return invoke<domain.Turn>("CancelSessionTurn", input);
}

export type SessionExecutionState = {
  id: string;
  sessionId: string;
  turnId?: string;
  status: "idle" | "running" | "interrupted" | "failed" | "compacting";
  reason?: string;
  lastEventId?: string;
  pendingInputIds?: string[];
  metadata?: Record<string, unknown>;
  timeCreated: string;
  timeUpdated: string;
};

export type CompactSessionContextResult = {
  state: SessionExecutionState;
  summary: domain.SessionSummary;
  context: domain.BuildSessionContextResult;
  compactedEventId?: string;
};

export function getSessionExecutionState(sessionId: string) {
  return invoke<SessionExecutionState>("GetSessionExecutionState", sessionId);
}

export function interruptSessionExecution(sessionId: string, reason = "") {
  return invoke<SessionExecutionState>("InterruptSessionExecution", {
    sessionId,
    reason,
  });
}

export function resumeSessionExecution(sessionId: string) {
  return invoke<SessionExecutionState>("ResumeSessionExecution", {
    sessionId,
  });
}

export function compactSessionContext(sessionId: string, characterBudget = 6000) {
  return invoke<CompactSessionContextResult>("CompactSessionContext", {
    sessionId,
    characterBudget,
  });
}

export type UpdateSessionEventInput = {
  eventId: string;
  content: string;
};

export type DeleteSessionEventInput = {
  eventId: string;
};

export type RetrySessionTurnInput = {
  sessionId?: string;
  turnId: string;
  model?: domain.ModelRef;
  agentMode?: string;
  toolsets?: string[];
  permissionScope?: string;
  reasoningEffort?: string;
  serviceTier?: string;
};

export type GetSessionTurnDiffInput = {
  sessionId?: string;
  turnId: string;
};

export type ApplySessionTurnFileStateInput = {
  sessionId?: string;
  turnId: string;
  toolCallId?: string;
  path?: string;
  targetState: "before" | "after";
};

export type SessionTurnDiffFile = {
  toolCallId: string;
  toolName: string;
  path: string;
  movePath?: string;
  type: string;
  additions?: number;
  deletions?: number;
  diff?: string;
  baseHash?: string;
  currentHash?: string;
  currentFileHash?: string;
  revertible: boolean;
  unrevertible: boolean;
  reason?: string;
  timeUpdated?: string;
};

export type SessionTurnDiff = {
  sessionId: string;
  turnId: string;
  files: SessionTurnDiffFile[];
  diff?: string;
};

export interface SessionEvent {
  id: string;
  sessionId: string;
  turnId?: string;
  type: string;
  role?: string;
  visibility: string;
  content?: string;
  payload?: Record<string, unknown>;
  timeCreated: string;
}

export function listSessionEvents(
  sessionId: string,
  includeNonNormal = false,
  limit = 100,
) {
  return invoke<SessionEvent[]>(
    "ListSessionEvents",
    sessionId,
    includeNonNormal,
    limit,
  );
}

export type SessionEventsCursorResult = {
  events: SessionEvent[];
  nextCursor: string;
};

export function listSessionEventsAfterCursor(input: {
  sessionId: string;
  cursor?: string;
  includeNonNormal?: boolean;
  limit?: number;
}) {
  return invoke<SessionEventsCursorResult>("ListSessionEventsAfterCursor", input);
}

export function updateSessionEvent(input: UpdateSessionEventInput) {
  return invoke<SessionEvent>("UpdateSessionEvent", input);
}

export function deleteSessionEvent(input: DeleteSessionEventInput) {
  return invoke<SessionEvent>("DeleteSessionEvent", input);
}

export function retrySessionTurn(input: RetrySessionTurnInput) {
  return invoke<domain.PreparedSessionTurn>("RetrySessionTurn", input);
}

export function getSessionTurnDiff(input: GetSessionTurnDiffInput) {
  return invoke<SessionTurnDiff>("GetSessionTurnDiff", input);
}

export function applySessionTurnFileState(
  input: ApplySessionTurnFileStateInput,
) {
  return invoke<SessionTurnDiff>("ApplySessionTurnFileState", input);
}

export type PluginManifest = {
  id: string;
  name: string;
  version?: string;
  displayName?: string;
  description?: string;
  author?: string;
  keywords?: string[];
  hooks?: string[];
  tools?: ToolCatalogEntry[];
};

export type PluginInstall = {
  id: string;
  manifest: PluginManifest;
  rootPath: string;
  manifestPath: string;
  enabled: boolean;
  status: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type PluginDiagnostic = {
  id: string;
  pluginId?: string;
  serverId?: string;
  level: string;
  message: string;
  metadata?: Record<string, unknown>;
  timeCreated: string;
};

export type ToolCatalogEntry = {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  namespace?: string;
  capability?: string;
  riskLevel?: string;
  category?: string;
  toolsets?: string[];
  source: string;
  sourceId?: string;
  registrationId?: string;
  enabled: boolean;
};

export type SessionActiveToolsResult = {
  sessionId: string;
  toolNames: string[];
};

export type PluginListItem = {
  plugin: PluginInstall;
  diagnostics?: PluginDiagnostic[];
  tools?: ToolCatalogEntry[];
};

export type MCPServerConfig = {
  id: string;
  name: string;
  displayName?: string;
  description?: string;
  transport: "stdio" | "streamable_http" | "sse";
  command?: string;
  args?: string[];
  cwd?: string;
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
  authType?: "none" | "bearer" | "oauth";
  bearerTokenEnv?: string;
  oauthIssuerUrl?: string;
  oauthClientId?: string;
  oauthScopes?: string[];
  oauthAccessTokenRef?: string;
  oauthRefreshTokenRef?: string;
  oauthExpiresAt?: string;
  roots?: string[];
  timeoutSeconds?: number;
  connectTimeoutSeconds?: number;
  enabled: boolean;
  pluginId?: string;
  status?: string;
  error?: string;
  timeCreated?: string;
  timeUpdated?: string;
};

export type MCPToolRecord = {
  id: string;
  serverId: string;
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  capability?: string;
  riskLevel?: string;
  timeUpdated: string;
};

export type MCPPromptArgument = {
  name: string;
  description?: string;
  required?: boolean;
};

export type MCPPromptRecord = {
  id: string;
  serverId: string;
  name: string;
  description?: string;
  arguments?: MCPPromptArgument[];
  timeUpdated: string;
};

export type MCPResourceRecord = {
  id: string;
  serverId: string;
  uri?: string;
  uriTemplate?: string;
  name: string;
  description?: string;
  mimeType?: string;
  template?: boolean;
  timeUpdated: string;
};

export type MCPContentBlock = {
  type: string;
  text?: string;
  uri?: string;
  mimeType?: string;
  blob?: string;
};

export type MCPPromptMessage = {
  role: string;
  content?: MCPContentBlock[];
};

export type MCPPromptGetResult = {
  serverId: string;
  name: string;
  description?: string;
  messages?: MCPPromptMessage[];
  content?: string;
  structured?: Record<string, unknown>;
};

export type MCPResourceContent = {
  uri?: string;
  mimeType?: string;
  text?: string;
  blob?: string;
};

export type MCPResourceReadResult = {
  serverId: string;
  uri: string;
  contents?: MCPResourceContent[];
  content?: string;
  structured?: Record<string, unknown>;
};

export type MCPServerLogResult = {
  serverId: string;
  content: string;
  offset: number;
  nextOffset: number;
  size: number;
  truncated?: boolean;
};

export type MCPServerListItem = {
  server: MCPServerConfig;
  tools?: MCPToolRecord[];
  prompts?: MCPPromptRecord[];
  resources?: MCPResourceRecord[];
  resourceTemplates?: MCPResourceRecord[];
  diagnostics?: PluginDiagnostic[];
};

export type MCPProbeResult = {
  ok: boolean;
  serverId?: string;
  status?: string;
  error?: string;
  tools?: MCPToolRecord[];
  prompts?: MCPPromptRecord[];
  resources?: MCPResourceRecord[];
  resourceTemplates?: MCPResourceRecord[];
  diagnostics?: PluginDiagnostic[];
};

export type MCPOAuthDiscoveryResult = {
  serverId?: string;
  resource?: string;
  resourceMetadataUrl?: string;
  authorizationServers?: string[];
  scopesSupported?: string[];
  selectedIssuer?: string;
  authorizationEndpoint?: string;
  tokenEndpoint?: string;
  registrationEndpoint?: string;
  introspectionEndpoint?: string;
  revocationEndpoint?: string;
  codeChallengeMethods?: string[];
  responseTypesSupported?: string[];
  grantTypesSupported?: string[];
  authorizationUrl?: string;
  discoveryErrors?: string[];
  resourceMetadata?: Record<string, unknown>;
  authorizationMetadata?: Record<string, unknown>;
  requiresDynamicClientRegistration?: boolean;
};

export type MCPOAuthStartResult = {
  serverId: string;
  status: string;
  url?: string;
  instructions?: string;
  expiresAt?: string;
};

export type MCPOAuthStatus = {
  serverId: string;
  status: string;
  error?: string;
  connected?: boolean;
  expiresAt?: string;
  clientId?: string;
  tokenSource?: string;
};

export function listPlugins(includeDisabled = true) {
  return invoke<PluginListItem[]>("ListPlugins", {
    includeDisabled,
    includeDiagnostics: true,
    limit: 100,
  });
}

export function installPluginFromPath(path: string, enable = true) {
  return invoke<PluginInstall>("InstallPluginFromPath", { path, enable });
}

export function setPluginEnabled(pluginId: string, enabled: boolean) {
  return invoke<PluginInstall>("SetPluginEnabled", { pluginId, enabled });
}

export function reloadPlugins() {
  return invoke<PluginListItem[]>("ReloadPlugins");
}

export function listMCPServers(includeDisabled = true, includeTools = true) {
  return invoke<MCPServerListItem[]>("ListMCPServers", {
    includeDisabled,
    includeTools,
  });
}

export function saveMCPServer(server: MCPServerConfig) {
  return invoke<MCPServerConfig>("SaveMCPServer", { server });
}

export function setMCPServerEnabled(serverId: string, enabled: boolean) {
  return invoke<MCPServerConfig>("SetMCPServerEnabled", { serverId, enabled });
}

export function probeMCPServer(serverId: string) {
  return invoke<MCPProbeResult>("ProbeMCPServer", { serverId });
}

export function getMCPPrompt(
  serverId: string,
  name: string,
  args: Record<string, string> = {},
) {
  return invoke<MCPPromptGetResult>("GetMCPPrompt", {
    serverId,
    name,
    arguments: args,
  });
}

export function readMCPResource(serverId: string, uri: string) {
  return invoke<MCPResourceReadResult>("ReadMCPResource", { serverId, uri });
}

export function insertMCPPromptIntoSession(
  sessionId: string,
  serverId: string,
  name: string,
  args: Record<string, string> = {},
) {
  return invoke<SessionEvent>("InsertMCPPromptIntoSession", {
    sessionId,
    serverId,
    name,
    arguments: args,
  });
}

export function insertMCPResourceIntoSession(
  sessionId: string,
  serverId: string,
  uri: string,
) {
  return invoke<SessionEvent>("InsertMCPResourceIntoSession", {
    sessionId,
    serverId,
    uri,
  });
}

export function readMCPServerLog(serverId: string, limit = 16_000) {
  return invoke<MCPServerLogResult>("ReadMCPServerLog", {
    serverId,
    limit,
    tail: true,
  });
}

export function discoverMCPOAuth(serverId: string) {
  return invoke<MCPOAuthDiscoveryResult>("DiscoverMCPOAuth", { serverId });
}

export function startMCPOAuth(serverId: string) {
  return invoke<MCPOAuthStartResult>("StartMCPOAuth", { serverId });
}

export function getMCPOAuthStatus(serverId: string) {
  return invoke<MCPOAuthStatus>("GetMCPOAuthStatus", { serverId });
}

export function listToolCatalog(workspaceRoot = "") {
  return invoke<ToolCatalogEntry[]>("ListToolCatalog", { workspaceRoot });
}

export function getSessionActiveTools(sessionId: string) {
  return invoke<SessionActiveToolsResult>("GetSessionActiveTools", sessionId);
}

export function setSessionActiveTools(sessionId: string, toolNames: string[]) {
  return invoke<SessionActiveToolsResult>("SetSessionActiveTools", {
    sessionId,
    toolNames,
  });
}
