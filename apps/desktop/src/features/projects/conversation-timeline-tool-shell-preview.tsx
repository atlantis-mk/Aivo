import {
  shellPreviewEntries,
  shellPrompt,
  terminalOutputSegment,
} from "@/features/projects/conversation-timeline-shell-model";
import type { domain } from "../../../bridge/go/models";

export function InlineShellPreview({
  resultText,
  toolCall,
}: {
  resultText: string;
  toolCall: domain.ToolCall;
}) {
  const entries = shellPreviewEntries(toolCall, resultText);
  if (entries.length === 0) return null;

  return (
    <div className="max-h-56 overflow-auto">
      <div className="flex min-w-0 flex-col gap-2">
        {entries.map((entry) => (
          <div
            className="min-w-0 border-b border-border/50 pb-2 last:border-b-0 last:pb-0"
            key={entry.id}
          >
            <div className="mb-1 min-w-0 truncate font-mono text-[11px] text-muted-foreground">
              {entry.toolName}
              {entry.exitCode !== undefined ? ` exit ${entry.exitCode}` : ""}
            </div>
            <pre className="m-0 whitespace-pre-wrap break-all font-mono text-[12px] leading-[1.45] text-foreground [overflow-wrap:anywhere]">
              <span>{shellPrompt(entry.cwd)}</span>
              <span>{entry.command}</span>
              {"\n"}
              {entry.stdout ? (
                <span>{terminalOutputSegment(entry.stdout)}</span>
              ) : null}
              {entry.stderr ? (
                <span className="text-destructive">
                  {terminalOutputSegment(entry.stderr)}
                </span>
              ) : null}
              {entry.error && !entry.stderr ? (
                <span className="text-destructive">
                  {terminalOutputSegment(entry.error)}
                </span>
              ) : null}
              {!entry.stdout && !entry.stderr && !entry.error ? (
                <span className="text-muted-foreground">暂无输出{"\n"}</span>
              ) : null}
            </pre>
          </div>
        ))}
      </div>
    </div>
  );
}
