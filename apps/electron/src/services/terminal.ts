import type { domain } from "@/types/codex-domain";
import { invoke } from "@/services/aivo/invoke";

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
  return invoke<TerminalInfo[]>("ListTerminals", workspaceRoot);
}

export function createTerminal(input: TerminalCreateInput) {
  return invoke<TerminalInfo>("CreateTerminal", input);
}

export function updateTerminal(input: TerminalUpdateInput) {
  return invoke<TerminalInfo>("UpdateTerminal", input);
}

export function removeTerminal(workspaceRoot: string, terminalId: string) {
  return invoke<null>("RemoveTerminal", workspaceRoot, terminalId);
}

export async function createTerminalConnectTicket(
  workspaceRoot: string,
  terminalId: string,
) {
  void workspaceRoot;
  void terminalId;
  return "";
}

export function terminalWebSocketURL(
  workspaceRoot: string,
  terminalId: string,
  ticket: string,
  cursor: number,
) {
  void workspaceRoot;
  void terminalId;
  void ticket;
  void cursor;
  return "";
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
  return invoke<ShellProcessInfo>("PollShellProcess", id);
}

export function waitShellProcess(id: string) {
  return invoke<ShellProcessInfo>("WaitShellProcess", id);
}

export function killShellProcess(id: string) {
  return invoke<ShellProcessInfo>("KillShellProcess", id);
}

export function readShellProcessOutput(id: string) {
  return invoke<ShellProcessInfo>("ReadShellProcessOutput", id);
}

export type TerminalEventPayload = {
  workspaceRoot: string;
  terminal: TerminalInfo;
};

export type ToolCall = domain.ToolCall;
