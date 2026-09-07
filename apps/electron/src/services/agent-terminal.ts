import { invoke } from "@/services/aivo/invoke";
import type {
  AgentTerminalInputMode,
  AgentTerminalInputRequest,
} from "@/features/projects/tool-activity-types";

export type AgentTerminalSnapshot = {
  type: "snapshot" | "status" | "input_request" | "input_granted" | "input_rejected" | "attention" | "exit" | "error";
  processRef?: string;
  status?: "running" | "waiting_input" | "exited";
  cursor?: number;
  baseCursor?: number;
  rows?: number;
  cols?: number;
  inputMode?: AgentTerminalInputMode;
  inputRequest?: AgentTerminalInputRequest | null;
  exitCode?: number | null;
  truncated?: boolean;
  message?: string;
  attention?: "none" | "possibly_waiting" | "interactive";
  inputOwner?: "none" | "user" | "agent";
  leaseMode?: "none" | "once" | "always";
  leaseVersion?: number;
  title?: string;
  command?: string;
  origin?: "agent" | "user";
};

export type ResolveAgentTerminalInputRequest = {
  workspaceRoot: string;
  sessionId: string;
  processRef: string;
  requestId: string;
  mode: Exclude<AgentTerminalInputMode, "ask">;
};

export async function createAgentTerminalConnectTicket(
  workspaceRoot: string,
  sessionId: string,
  processRef: string,
) {
  void workspaceRoot;
  void sessionId;
  void processRef;
  return "";
}

export function agentTerminalWebSocketURL(
  workspaceRoot: string,
  sessionId: string,
  processRef: string,
  ticket: string,
  cursor: number,
) {
  void workspaceRoot;
  void sessionId;
  void processRef;
  void ticket;
  void cursor;
  return "";
}

export function resolveAgentTerminalInput(input: ResolveAgentTerminalInputRequest) {
  return invoke<AgentTerminalSnapshot>("ResolveAgentTerminalInput", input);
}

export function listSessionTerminals(workspaceRoot: string, sessionId: string) {
  return invoke<AgentTerminalSnapshot[]>("ListSessionTerminals", workspaceRoot, sessionId);
}

export function releaseAgentTerminalInput(input: { workspaceRoot: string; sessionId: string; processRef: string; leaseVersion: number }) {
  return invoke<AgentTerminalSnapshot>("ReleaseAgentTerminalInput", input);
}

export function terminateSessionTerminals(workspaceRoot: string, sessionId: string) {
  return invoke<null>("TerminateSessionTerminals", workspaceRoot, sessionId);
}

export function updateSessionTerminal(input: { workspaceRoot: string; sessionId: string; processRef: string; title: string }) {
  return invoke<AgentTerminalSnapshot>("UpdateSessionTerminal", input);
}

export function removeSessionTerminal(workspaceRoot: string, sessionId: string, processRef: string) {
  return invoke<null>("RemoveSessionTerminal", workspaceRoot, sessionId, processRef);
}
