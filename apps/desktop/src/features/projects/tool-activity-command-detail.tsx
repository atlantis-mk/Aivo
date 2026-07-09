import { ScrollArea } from "@/components/ui/scroll-area";
import type { ToolActivityCommandTab } from "@/features/projects/tool-activity-model";
import { useFollowScrollToEnd } from "@/features/projects/tool-activity-follow-scroll";
import {
  commandEntries,
  shellPrompt,
  terminalOutputSegment,
} from "@/features/projects/tool-activity-sidebar-model";

export function CommandActivityDetail({ tab }: { tab: ToolActivityCommandTab }) {
  const entries = commandEntries(tab);
  const outputKey = entries
    .map(
      (entry) =>
        `${entry.id}:${entry.stdout.length}:${entry.stderr.length}:${entry.status}`,
    )
    .join("|");
  const { endRef, scrollAreaRef } = useFollowScrollToEnd(outputKey);

  return (
    <ScrollArea
      className="min-h-0 flex-1 bg-background [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0 [&_[data-slot=scroll-area-viewport]>div]:!w-full"
      ref={scrollAreaRef}
    >
      <div className="flex min-h-full w-full max-w-full flex-col p-3">
        {entries.map((entry) => (
          <div className="min-w-0" key={entry.id}>
            <pre className="m-0 w-full max-w-full whitespace-pre-wrap break-all font-mono text-[12px] leading-[1.45] text-foreground [overflow-wrap:anywhere]">
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
            </pre>
          </div>
        ))}
        <span ref={endRef} />
      </div>
    </ScrollArea>
  );
}
