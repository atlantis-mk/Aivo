import { invokeBridge } from "@/services/bridge-invoke";
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
  const response = await fetch(
    `${window.aivo.coreUrl}/api/terminals/${encodeURIComponent(processRef)}/connect-token`,
    {
      method: "POST",
      headers: { "Aivo-Terminal-CSRF": "1", "Content-Type": "application/json" },
      body: JSON.stringify({ workspaceRoot, sessionId }),
    },
  );
  const payload = (await response.json().catch(() => null)) as { ticket?: string } | null;
  if (!response.ok || !payload?.ticket) throw new Error("Failed to create agent terminal ticket");
  return payload.ticket;
}

export function agentTerminalWebSocketURL(
  workspaceRoot: string,
  sessionId: string,
  processRef: string,
  ticket: string,
  cursor: number,
) {
  const url = new URL(`${window.aivo.coreUrl}/api/terminals/${encodeURIComponent(processRef)}/connect`);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("workspaceRoot", workspaceRoot);
  url.searchParams.set("sessionId", sessionId);
  url.searchParams.set("ticket", ticket);
  url.searchParams.set("cursor", String(cursor));
  return url.toString();
}

export function resolveAgentTerminalInput(input: ResolveAgentTerminalInputRequest) {
  return invokeBridge<AgentTerminalSnapshot>("ResolveAgentTerminalInput", input);
}

export function listSessionTerminals(workspaceRoot: string, sessionId: string) {
  return invokeBridge<AgentTerminalSnapshot[]>("ListSessionTerminals", workspaceRoot, sessionId);
}

export function releaseAgentTerminalInput(input: { workspaceRoot: string; sessionId: string; processRef: string; leaseVersion: number }) {
  return invokeBridge<AgentTerminalSnapshot>("ReleaseAgentTerminalInput", input);
}

export function terminateSessionTerminals(workspaceRoot: string, sessionId: string) {
  return invokeBridge<null>("TerminateSessionTerminals", workspaceRoot, sessionId);
}

export function updateSessionTerminal(input: { workspaceRoot: string; sessionId: string; processRef: string; title: string }) {
  return invokeBridge<AgentTerminalSnapshot>("UpdateSessionTerminal", input);
}

export function removeSessionTerminal(workspaceRoot: string, sessionId: string, processRef: string) {
  return invokeBridge<null>("RemoveSessionTerminal", workspaceRoot, sessionId, processRef);
}
