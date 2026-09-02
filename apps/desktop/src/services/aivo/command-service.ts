import { invoke } from "@/services/aivo/invoke";
import type { domain } from "../../../bridge/go/models";

export type CommandArgument = {
  name: string;
  description?: string;
  required?: boolean;
  default?: string;
};

export type CommandCatalogEntry = {
  id: string;
  name: string;
  description?: string;
  source: "builtin" | "config" | "skill" | "mcp";
  sourceId?: string;
  arguments?: CommandArgument[];
  agent?: string;
  model?: domain.ModelRef;
  toolsets?: string[];
  subtask?: boolean;
};

export type InvokeCommandResult = {
  commandId: string;
  source: string;
  sourceId?: string;
  prompt: string;
  agent?: string;
  model?: domain.ModelRef;
  toolsets?: string[];
  subtask?: boolean;
  childSessionId?: string;
  agentRunId?: string;
  response?: string;
};

export function listCommandCatalog(projectPath: string) {
  return invoke<CommandCatalogEntry[]>("ListCommandCatalog", { projectPath });
}

export function invokeCommand(input: {
  sessionId?: string;
  projectPath?: string;
  commandId: string;
  arguments?: Record<string, string>;
}) {
  return invoke<InvokeCommandResult>("InvokeCommand", input);
}

export function getEffectiveRuntimeConfig(projectPath: string) {
  return invoke<{
    projectPath?: string;
    config: Record<string, unknown>;
    sources?: Array<{ path: string; scope: string }>;
    diagnostics?: Array<{ path: string; level: string; message: string }>;
  }>("GetEffectiveRuntimeConfig", projectPath);
}

export function parseCommandArgumentLine(line: string) {
  const tokens: string[] = [];
  let current = "";
  let quote: "'" | '"' | "" = "";
  let escaping = false;
  const push = () => {
    if (current) tokens.push(current);
    current = "";
  };
  for (const char of line.trim()) {
    if (escaping) {
      current += char;
      escaping = false;
      continue;
    }
    if (char === "\\") {
      escaping = true;
      continue;
    }
    if (quote) {
      if (char === quote) quote = "";
      else current += char;
      continue;
    }
    if (char === "'" || char === '"') {
      quote = char;
      continue;
    }
    if (/\s/.test(char)) {
      push();
      continue;
    }
    current += char;
  }
  if (escaping) current += "\\";
  if (quote) throw new Error("命令参数存在未闭合的引号");
  push();
  return tokens;
}
