import { useEffect, useMemo, useRef, useState } from "react";
import { Cancel01Icon, CommandLineIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ToolActivityCommandTab } from "@/features/projects/tool-activity-model";
import { useFollowScrollToEnd } from "@/features/projects/tool-activity-follow-scroll";
import {
  commandEntries,
  shellPrompt,
  terminalOutputSegment,
  tabTitle,
} from "@/features/projects/tool-activity-sidebar-model";
import { commandSessionId } from "@/features/projects/tool-activity-command-entry-model";
import { AgentTerminalView } from "@/features/projects/agent-terminal-view";
import { ToolActivityStatusIcon } from "@/features/projects/tool-activity-status-icon";
import {
  listSessionTerminals,
  removeSessionTerminal,
  terminateSessionTerminals,
  updateSessionTerminal,
  type AgentTerminalSnapshot,
} from "@/services/agent-terminal";

export function CommandActivityDetail({
  tab,
  workspaceRoot,
}: {
  tab: ToolActivityCommandTab;
  workspaceRoot: string;
}) {
  const entries = commandEntries(tab);
  const outputKey = entries
    .map(
      (entry) =>
        `${entry.id}:${entry.stdout.length}:${entry.stderr.length}:${entry.status}`,
    )
    .join("|");
  const { endRef, scrollAreaRef } = useFollowScrollToEnd(outputKey);
  const interactiveEntries = entries.filter((entry) =>
    entry.processRef?.startsWith("agent-pty:"),
  );
  if (interactiveEntries.length > 0) {
    return (
      <SessionTerminalSwitcher
        entries={interactiveEntries}
        sessionId={commandSessionId(tab)}
        workspaceRoot={workspaceRoot}
      />
    );
  }
  return (
    <div className="flex h-full min-h-0 flex-col bg-background p-2">
      <section className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-border/80 bg-card text-card-foreground shadow-sm shadow-foreground/[0.03]">
        <div className="flex min-h-11 shrink-0 items-center gap-2 px-4 pt-3 pb-2">
          <HugeiconsIcon
            aria-hidden="true"
            className="size-3.5 shrink-0 text-muted-foreground"
            icon={CommandLineIcon}
            strokeWidth={2}
          />
          <div className="min-w-0 flex-1 truncate text-sm font-semibold">
            {tabTitle(tab)}
          </div>
          <div className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
            <span>{entries.length} 条记录</span>
            <ToolActivityStatusIcon className="size-3.5" status={tab.status} />
          </div>
        </div>
        <ScrollArea
          className="min-h-0 flex-1 [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0 [&_[data-slot=scroll-area-viewport]>div]:!w-full"
          ref={scrollAreaRef}
        >
          <div className="flex min-h-full w-full max-w-full flex-col gap-4 px-4 pb-4 pt-1">
            {entries.map((entry) => (
              <div className="min-w-0" key={entry.id}>
                <pre className="m-0 w-full max-w-full whitespace-pre-wrap break-all font-mono text-[12px] leading-[1.55] text-foreground [overflow-wrap:anywhere]">
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
      </section>
    </div>
  );
}

function SessionTerminalSwitcher({
  entries,
  sessionId,
  workspaceRoot,
}: {
  entries: ReturnType<typeof commandEntries>;
  sessionId: string;
  workspaceRoot: string;
}) {
  const entryByRef = useMemo(
    () => new Map(entries.flatMap((entry) => entry.processRef ? [[entry.processRef, entry] as const] : [])),
    [entries],
  );
  const initialRefs = [...entryByRef.keys()];
  const [terminals, setTerminals] = useState<AgentTerminalSnapshot[]>([]);
  const [activeRef, setActiveRef] = useState(initialRefs.at(-1) ?? "");
  const [dismissedRefs, setDismissedRefs] = useState<Set<string>>(() => new Set());
  const [renaming, setRenaming] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [closingRefs, setClosingRefs] = useState<Set<string>>(() => new Set());
  const [terminalError, setTerminalError] = useState("");
  const initializedRef = useRef(false);

  useEffect(() => {
    let disposed = false;
    const refresh = async () => {
      try {
        const next = await listSessionTerminals(workspaceRoot, sessionId);
        if (disposed) return;
        setTerminals(next);
        if (!initializedRef.current) {
          initializedRef.current = true;
          setActiveRef((current) => current || next.at(-1)?.processRef || "");
        }
      } catch {
        // Tool activity remains the fallback when the live runtime is unavailable.
      }
    };
    void refresh();
    const timer = window.setInterval(refresh, 1500);
    return () => { disposed = true; window.clearInterval(timer); };
  }, [sessionId, workspaceRoot]);

  const refs = [...new Set([...initialRefs, ...terminals.flatMap((terminal) => terminal.processRef ? [terminal.processRef] : [])])]
    .filter((processRef) => !dismissedRefs.has(processRef));
  const activeEntry = entryByRef.get(activeRef);
  const activeTerminal = terminals.find((terminal) => terminal.processRef === activeRef);
  const closeTerminal = async (processRef: string) => {
    if (closingRefs.has(processRef)) return;
    setClosingRefs((current) => new Set(current).add(processRef));
    setTerminalError("");
    setTerminals((current) => current.filter((terminal) => terminal.processRef !== processRef));
    setDismissedRefs((current) => new Set(current).add(processRef));
    setActiveRef((current) => current === processRef ? (refs.find((ref) => ref !== processRef) ?? "") : current);
    try {
      await removeSessionTerminal(workspaceRoot, sessionId, processRef);
    } catch (cause) {
      setTerminalError(cause instanceof Error ? cause.message : "无法关闭终端");
    } finally {
      setClosingRefs((current) => {
        const next = new Set(current);
        next.delete(processRef);
        return next;
      });
    }
  };
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-9 shrink-0 items-center gap-1 overflow-x-auto border-b px-2">
        {refs.map((processRef, index) => {
          const terminal = terminals.find((item) => item.processRef === processRef);
          const label = terminal?.title || entryByRef.get(processRef)?.command || `Terminal ${index + 1}`;
          const attention = terminal?.status === "waiting_input" || terminal?.attention === "possibly_waiting" || terminal?.attention === "interactive";
          return (
            <div className="flex shrink-0 items-center" key={processRef}>
              <Button
                className="max-w-44 rounded-r-none pr-1"
                onClick={() => setActiveRef(processRef)}
                size="xs"
                variant={processRef === activeRef ? "secondary" : "ghost"}
              >
                <span className={attention ? "size-1.5 rounded-full bg-amber-500" : terminal?.status === "exited" ? "size-1.5 rounded-full bg-muted-foreground" : "size-1.5 rounded-full bg-emerald-500"} />
                <span className="truncate">{label}</span>
              </Button>
              <Button
                aria-label={`关闭 ${label}`}
                className="rounded-l-none px-1.5"
                disabled={closingRefs.has(processRef)}
                onClick={() => void closeTerminal(processRef)}
                size="icon-xs"
                title={terminal?.status === "exited" ? "关闭终端" : "终止并关闭终端"}
                variant={processRef === activeRef ? "secondary" : "ghost"}
              >
                <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
              </Button>
            </div>
          );
        })}
        {terminalError ? <span className="max-w-48 truncate text-xs text-destructive">{terminalError}</span> : null}
        <Button className="ml-auto shrink-0" onClick={() => void terminateSessionTerminals(workspaceRoot, sessionId)} size="xs" variant="destructive">
          终止全部
        </Button>
        {renaming ? (
          <form
            className="flex shrink-0 items-center gap-1"
            onSubmit={(event) => {
              event.preventDefault();
              const title = titleDraft.trim();
              if (!title) return;
              void updateSessionTerminal({ workspaceRoot, sessionId, processRef: activeRef, title }).then(() => {
                setTerminals((current) => current.map((terminal) => terminal.processRef === activeRef ? { ...terminal, title } : terminal));
                setRenaming(false);
              });
            }}
          >
            <Input autoFocus className="h-6 w-36 px-2 text-xs" onChange={(event) => setTitleDraft(event.target.value)} value={titleDraft} />
            <Button size="xs" type="submit">保存</Button>
          </form>
        ) : (
          <Button
            disabled={!activeRef}
            onClick={() => { setTitleDraft(activeTerminal?.title || activeEntry?.command || "Terminal"); setRenaming(true); }}
            size="xs"
            variant="outline"
          >
            重命名
          </Button>
        )}
      </div>
      {activeRef ? (
        <AgentTerminalView
          initialInputMode={activeEntry?.inputMode}
          initialInputRequest={activeEntry?.inputRequest}
          key={activeRef}
          processRef={activeRef}
          sessionId={sessionId}
          workspaceRoot={workspaceRoot}
        />
      ) : null}
      {activeTerminal?.status === "exited" ? <span className="sr-only">Terminal exited</span> : null}
    </div>
  );
}
