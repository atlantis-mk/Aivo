import type { domain } from "../../bridge/go/models";
import { invokeBridge } from "@/services/bridge-invoke";

export type TerminalInfo = {
  id: string;
  workspaceRoot: string;
  sessionId?: string;
  origin: "user" | "agent";
  title: string;
  command: string;
  args?: string[];
  cwd: string;
  status: "running" | "waiting_input" | "exited" | "removed";
  attention: "none" | "possibly_waiting" | "interactive";
  inputOwner: "none" | "user" | "agent";
  leaseMode: "none" | "once" | "always";
  leaseVersion: number;
  pid: number;
  exitCode?: number;
  rows: number;
  cols: number;
  cursor: number;
  truncated?: boolean;
  timeCreated: string;
  timeUpdated: string;
};

export type TerminalCreateInput = {
  workspaceRoot: string;
  cwd?: string;
  title?: string;
  shell?: string;
  env?: Record<string, string>;
  rows?: number;
  cols?: number;
};

export type TerminalUpdateInput = {
  workspaceRoot: string;
  terminalId: string;
  title?: string;
  rows?: number;
  cols?: number;
};

export function listTerminals(workspaceRoot: string) {
  return invokeBridge<TerminalInfo[]>("ListTerminals", workspaceRoot);
}

export function createTerminal(input: TerminalCreateInput) {
  return invokeBridge<TerminalInfo>("CreateTerminal", input);
}

export function updateTerminal(input: TerminalUpdateInput) {
  return invokeBridge<TerminalInfo>("UpdateTerminal", input);
}

export function removeTerminal(workspaceRoot: string, terminalId: string) {
  return invokeBridge<null>("RemoveTerminal", workspaceRoot, terminalId);
}

export async function createTerminalConnectTicket(
  workspaceRoot: string,
  terminalId: string,
) {
  const response = await fetch(
    `${window.aivo.coreUrl}/api/terminals/${encodeURIComponent(
      terminalId,
    )}/connect-token`,
    {
      method: "POST",
      headers: {
        "Aivo-Terminal-CSRF": "1",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ workspaceRoot }),
    },
  );
  const payload = (await response.json().catch(() => null)) as
    | { ticket?: string }
    | null;
  if (!response.ok || !payload?.ticket) {
    throw new Error("Failed to create terminal connect ticket");
  }
  return payload.ticket;
}

export function terminalWebSocketURL(
  workspaceRoot: string,
  terminalId: string,
  ticket: string,
  cursor: number,
) {
  const url = new URL(
    `${window.aivo.coreUrl}/api/terminals/${encodeURIComponent(
      terminalId,
    )}/connect`,
  );
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("workspaceRoot", workspaceRoot);
  url.searchParams.set("ticket", ticket);
  url.searchParams.set("cursor", String(cursor));
  return url.toString();
}

export type ShellProcessInfo = {
  id: string;
  status: string;
  command: string;
  cwd: string;
  pid: number;
  exitCode?: number;
  stdout?: string;
  stderr?: string;
};

export function pollShellProcess(id: string) {
  return invokeBridge<ShellProcessInfo>("PollShellProcess", id);
}

export function waitShellProcess(id: string) {
  return invokeBridge<ShellProcessInfo>("WaitShellProcess", id);
}

export function killShellProcess(id: string) {
  return invokeBridge<ShellProcessInfo>("KillShellProcess", id);
}

export function readShellProcessOutput(id: string) {
  return invokeBridge<ShellProcessInfo>("ReadShellProcessOutput", id);
}

export type TerminalEventPayload = {
  workspaceRoot: string;
  terminal: TerminalInfo;
};

export type ToolCall = domain.ToolCall;
