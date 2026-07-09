export const OUTPUT_PREVIEW_CHARS = 60_000;
export const FILE_PREVIEW_CHARS = 32_000;

export type ToolActivityStatus =
  | "running"
  | "success"
  | "failed"
  | "pending_approval";

export type ToolActivityFileTab = {
  id: string;
  kind: "file";
  draft?: boolean;
  toolCallId: string;
  turnId?: string;
  toolName: string;
  path: string;
  relativePath?: string;
  movePath?: string;
  relativeMovePath?: string;
  operation: string;
  status: ToolActivityStatus;
  contentPreview?: string;
  diff?: string;
  additions?: number;
  deletions?: number;
  baseHash?: string;
  currentHash?: string;
  currentFileHash?: string;
  revertible?: boolean;
  unrevertible?: boolean;
  revertReason?: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type ToolActivityFileState = {
  toolCallId: string;
  path: string;
  movePath?: string;
  revertible?: boolean;
  unrevertible?: boolean;
  reason?: string;
  currentFileHash?: string;
  timeUpdated?: string;
};

export type ToolActivityCommandEntry = {
  id: string;
  toolCallId: string;
  turnId?: string;
  toolName: string;
  command: string;
  cwd?: string;
  status: ToolActivityStatus;
  stdout: string;
  stderr: string;
  exitCode?: number;
  durationMs?: number;
  replayOfToolCallId?: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type ToolActivityCommandTab = ToolActivityCommandEntry & {
  kind: "command";
  entries: ToolActivityCommandEntry[];
};

export type ToolActivityTab = ToolActivityFileTab | ToolActivityCommandTab;

export type ShellOutputPayload = {
  sessionId?: string;
  turnId?: string;
  toolCallId?: string;
  processRef?: string;
  stream?: string;
  chunk?: string;
  sequence?: number;
  timeCreated?: string;
};

export type ToolFileChange = {
  path: string;
  fullPath?: string;
  movePath?: string;
  moveFullPath?: string;
  type: string;
  additions: number;
  deletions: number;
  diff?: string;
  baseHash?: string;
  currentHash?: string;
};
