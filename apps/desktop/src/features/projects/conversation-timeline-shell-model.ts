import {
  arrayRecords,
  objectRecord,
  optionalNumberValue,
  stringArg,
  stringValue,
} from "@/features/projects/conversation-timeline-value-model";
import type { domain } from "../../../bridge/go/models";

const INLINE_SHELL_OUTPUT_CHARS = 4_000;

export type ShellPreviewEntry = {
  command: string;
  cwd?: string;
  error?: string;
  exitCode?: number;
  id: string;
  stderr: string;
  stdout: string;
  toolName: string;
};

export function shellPreviewEntries(
  toolCall: domain.ToolCall,
  resultText: string,
): ShellPreviewEntry[] {
  const structured = objectRecord(toolCall.result?.structured);
  if (toolCall.name === "run_tests") {
    const commands = arrayRecords(structured?.commands);
    if (commands.length > 0) {
      return commands.map((command, index) =>
        shellPreviewEntryFromStructured(toolCall, command, index),
      );
    }
  }

  if (structured) {
    return [shellPreviewEntryFromStructured(toolCall, structured, 0)];
  }

  return [shellPreviewEntryFromResultText(toolCall, resultText)];
}

export function terminalOutputSegment(content: string) {
  return content.endsWith("\n") ? content : `${content}\n`;
}

export function shellPrompt(cwd?: string) {
  return `agent@aivo ${shellCwdLabel(cwd)} % `;
}

function shellPreviewEntryFromStructured(
  toolCall: domain.ToolCall,
  structured: Record<string, unknown>,
  index: number,
): ShellPreviewEntry {
  const args = toolCall.arguments ?? {};
  return {
    command: stringValue(structured.command) || shellCommandFromToolArgs(toolCall),
    cwd: stringValue(structured.cwd) || stringArg(args, "workdir"),
    error: toolCall.error || stringValue(toolCall.result?.error),
    exitCode: optionalNumberValue(structured.exitCode),
    id: `${toolCall.id}:${index}`,
    stderr: previewInlineShellOutput(stringValue(structured.stderr)),
    stdout: previewInlineShellOutput(stringValue(structured.stdout)),
    toolName: toolCall.name,
  };
}

function shellPreviewEntryFromResultText(
  toolCall: domain.ToolCall,
  resultText: string,
): ShellPreviewEntry {
  const parsed = parseCommandResultText(resultText);
  return {
    command: parsed.command || shellCommandFromToolArgs(toolCall),
    cwd: parsed.cwd || stringArg(toolCall.arguments ?? {}, "workdir"),
    error: toolCall.error || parsed.error || stringValue(toolCall.result?.error),
    exitCode: parsed.exitCode,
    id: `${toolCall.id}:0`,
    stderr: previewInlineShellOutput(parsed.stderr),
    stdout: previewInlineShellOutput(parsed.stdout),
    toolName: toolCall.name,
  };
}

function shellCommandFromToolArgs(toolCall: domain.ToolCall) {
  const args = toolCall.arguments ?? {};
  if (toolCall.name === "exec_command") {
    return stringArg(args, "cmd") || "exec_command";
  }
  if (toolCall.name === "run_tests") {
    return [stringArg(args, "target") || "all", stringArg(args, "kind") || "auto"]
      .filter(Boolean)
      .join(":");
  }
  return toolCall.name || "tool";
}

function parseCommandResultText(text: string) {
  const lines = text.split("\n");
  const parsed: {
    command?: string;
    cwd?: string;
    error?: string;
    exitCode?: number;
    stderr: string;
    stdout: string;
  } = {
    stderr: "",
    stdout: "",
  };
  let stream: "stdout" | "stderr" | null = null;

  for (const line of lines) {
    if (line.startsWith("STDOUT:")) {
      stream = "stdout";
      continue;
    }
    if (line.startsWith("STDERR:")) {
      stream = "stderr";
      continue;
    }
    if (stream === "stdout") {
      parsed.stdout += `${line}\n`;
      continue;
    }
    if (stream === "stderr") {
      parsed.stderr += `${line}\n`;
      continue;
    }
    if (line.startsWith("Command:")) {
      parsed.command = line.replace(/^Command:\s*/, "").trim();
    } else if (line.startsWith("CWD:")) {
      parsed.cwd = line.replace(/^CWD:\s*/, "").trim();
    } else if (line.startsWith("Exit code:")) {
      parsed.exitCode = optionalNumberValue(
        Number(line.replace(/^Exit code:\s*/, "").trim()),
      );
    } else if (line.startsWith("Error:")) {
      parsed.error = line.replace(/^Error:\s*/, "").trim();
    }
  }

  parsed.stdout = parsed.stdout.trimEnd();
  parsed.stderr = parsed.stderr.trimEnd();
  return parsed;
}

function previewInlineShellOutput(text: string) {
  if (text.length <= INLINE_SHELL_OUTPUT_CHARS) return text;
  const omitted = text.length - INLINE_SHELL_OUTPUT_CHARS;
  return `${text.slice(0, INLINE_SHELL_OUTPUT_CHARS).trimEnd()}\n... 已省略 ${omitted.toLocaleString()} 个字符 ...`;
}

function shellCwdLabel(cwd?: string) {
  const value = cwd?.trim();
  if (!value) return "~";
  const parts = value.split("/").filter(Boolean);
  return parts.at(-1) || "/";
}
