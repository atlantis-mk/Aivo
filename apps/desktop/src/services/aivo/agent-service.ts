import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

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
