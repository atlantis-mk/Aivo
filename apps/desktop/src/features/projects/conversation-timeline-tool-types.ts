import type { domain } from "../../../bridge/go/models";

export type ToolCallGroup = {
  id: string;
  kind: string;
  calls: domain.ToolCall[];
  timeCreated?: string;
  title: string;
};

export type ToolFileChange = {
  path: string;
  movePath?: string;
  type: string;
  additions: number;
  deletions: number;
  diff?: string;
};
