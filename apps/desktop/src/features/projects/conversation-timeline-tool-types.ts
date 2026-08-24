import type { domain } from "../../../bridge/go/models";

export type ToolCallGroup = {
  description?: string;
  id: string;
  kind: string;
  calls: domain.ToolCall[];
  timeCreated?: string;
  title: string;
};

export type ToolCallActivity = {
  id: string;
  groups: ToolCallGroup[];
};

export type ToolFileChange = {
  path: string;
  movePath?: string;
  type: string;
  additions: number;
  deletions: number;
  diff?: string;
};
