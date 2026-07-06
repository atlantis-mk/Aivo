import {
  ChevronDown,
  Plus,
  TerminalSquare,
  X,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { EventsOn } from "../../../../bridge/runtime/runtime";
import {
  createTerminal,
  listTerminals,
  removeTerminal,
  updateTerminal,
  type TerminalEventPayload,
  type TerminalInfo,
} from "@/services/terminal";
import {
  clampTerminalHeight,
  defaultTerminalState,
  useTerminalWorkspaceStateStore,
  type WorkspaceTerminalState,
} from "@/features/projects/terminal/terminal-state";
import { TerminalView } from "@/features/projects/terminal/terminal-view";
import { useTerminalDock } from "@/features/projects/terminal/terminal-dock-store";

const EMPTY_TERMINAL_STATE = defaultTerminalState();

type TerminalPanelProps = {
  enabled: boolean;
  height?: number;
  terminalEnabled?: boolean;
  workspaceRoot: string;
};

export function TerminalPanelContent({
  enabled,
  height,
  terminalEnabled = true,
  workspaceRoot,
}: TerminalPanelProps) {
  const { open, setOpen } = useTerminalDock();
  const state = useTerminalWorkspaceStateStore((store) =>
    workspaceRoot
      ? store.workspaceStates[workspaceRoot] ?? EMPTY_TERMINAL_STATE
      : EMPTY_TERMINAL_STATE,
  );
  const loadWorkspaceState = useTerminalWorkspaceStateStore(
    (store) => store.loadWorkspaceState,
  );
  const setWorkspaceState = useTerminalWorkspaceStateStore(
    (store) => store.setWorkspaceState,
  );
  const setState = useCallback(
    (
      updater:
        | WorkspaceTerminalState
        | ((current: WorkspaceTerminalState) => WorkspaceTerminalState),
    ) => {
      setWorkspaceState(workspaceRoot, updater);
    },
    [setWorkspaceState, workspaceRoot],
  );
  const [terminals, setTerminals] = useState<TerminalInfo[]>([]);
  const [terminalsLoaded, setTerminalsLoaded] = useState(false);
  const [creatingTerminal, setCreatingTerminal] = useState(false);
  const [error, setError] = useState("");
  const stateRef = useRef(state);
  const cursorUpdatesRef = useRef<Record<string, number>>({});
  const cursorUpdateTimerRef = useRef<number | null>(null);
  const creatingTerminalRef = useRef(false);
  const canOpenPanel = enabled;
  const canUseTerminal = canOpenPanel && terminalEnabled;

  const activeTerminal = useMemo(
    () =>
      terminals.find((terminal) => terminal.id === state.activeId) ??
      terminals[0],
    [state.activeId, terminals],
  );

  useEffect(() => {
    if (!workspaceRoot) {
      setState(defaultTerminalState());
      setTerminals([]);
      setTerminalsLoaded(false);
      setCreatingTerminal(false);
      setError("");
      creatingTerminalRef.current = false;
      return;
    }
    const nextState = loadWorkspaceState(workspaceRoot);
    setOpen(nextState.opened);
    setTerminals([]);
    setTerminalsLoaded(false);
    setCreatingTerminal(false);
    setError("");
    creatingTerminalRef.current = false;
  }, [loadWorkspaceState, setOpen, setState, workspaceRoot]);

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    if (!canOpenPanel || !open || typeof height !== "number") return;
    const clampedHeight = clampTerminalHeight(height);
    setState((current) =>
      current.height === clampedHeight
        ? current
        : { ...current, height: clampedHeight },
    );
  }, [canOpenPanel, height, open, setState]);

  useEffect(() => {
    setState((current) => {
      if (current.opened === open) return current;
      return { ...current, opened: open };
    });
  }, [open, setState]);

  useEffect(
    () => () => {
      if (cursorUpdateTimerRef.current !== null) {
        window.clearTimeout(cursorUpdateTimerRef.current);
      }
    },
    [],
  );

  const reload = useCallback(async () => {
    if (!canUseTerminal) return;
    try {
      const next = await listTerminals(workspaceRoot);
      const liveTerminals = next.filter(
        (terminal) => terminal.status !== "removed",
      );
      setTerminals(liveTerminals);
      setTerminalsLoaded(true);
      setError("");
      setState((current) => {
        const liveIds = new Set(liveTerminals.map((terminal) => terminal.id));
        const tabs = current.tabs.filter((tab) => liveIds.has(tab.id));
        const tabIds = new Set(tabs.map((tab) => tab.id));
        let titleNumber = nextTitleNumber({ ...current, tabs });
        for (const terminal of liveTerminals) {
          if (tabIds.has(terminal.id)) continue;
          tabs.push({
            id: terminal.id,
            title: terminal.title,
            titleNumber,
            status: terminal.status === "running" ? "running" : "exited",
          });
          titleNumber += 1;
        }
        const nextTabIds = new Set(tabs.map((tab) => tab.id));
        const activeId =
          current.activeId && nextTabIds.has(current.activeId)
            ? current.activeId
            : tabs[0]?.id;
        if (
          current.activeId === activeId &&
          tabs.length === current.tabs.length &&
          tabs.every((tab, index) => tab.id === current.tabs[index]?.id)
        ) {
          return current;
        }
        return { ...current, activeId, tabs };
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setTerminalsLoaded(true);
    }
  }, [canUseTerminal, setState, workspaceRoot]);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    if (!canUseTerminal) return;
    const names = [
      "terminal.created",
      "terminal.updated",
      "terminal.exited",
      "terminal.removed",
    ];
    const off = names.map((name) =>
      EventsOn(name, (payload: TerminalEventPayload) => {
        if (payload.workspaceRoot !== workspaceRoot) return;
        void reload();
      }),
    );
    return () => off.forEach((dispose) => dispose());
  }, [canUseTerminal, reload, workspaceRoot]);

  const setCreatedTerminal = useCallback(
    (terminal: TerminalInfo, titleNumber: number) => {
      setTerminals((current) => upsertTerminal(current, terminal));
      setTerminalsLoaded(true);
      setOpen(true);
      setState((current) => ({
        ...current,
        opened: true,
        activeId: terminal.id,
        tabs: current.tabs.some((tab) => tab.id === terminal.id)
          ? current.tabs.map((tab) =>
              tab.id === terminal.id
                ? { ...tab, title: terminal.title, status: "connecting" }
                : tab,
            )
          : [
              ...current.tabs,
              {
                id: terminal.id,
                title: terminal.title,
                titleNumber,
                status: "connecting",
              },
            ],
      }));
      setError("");
    },
    [setOpen, setState],
  );

  const addTerminal = useCallback(async () => {
    if (!canUseTerminal) return;
    if (creatingTerminalRef.current) return;
    creatingTerminalRef.current = true;
    setCreatingTerminal(true);
    try {
      const titleNumber = nextTitleNumber(stateRef.current);
      const input = {
        workspaceRoot,
        title: `Shell ${titleNumber}`,
        rows: 24,
        cols: 80,
      };
      const terminal = await createTerminal(input).catch((err: unknown) => {
        if (!shouldRetryWithPOSIXShell(err)) {
          throw err;
        }
        return createTerminal({ ...input, shell: "/bin/sh" });
      });
      setCreatedTerminal(terminal, titleNumber);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      creatingTerminalRef.current = false;
      setCreatingTerminal(false);
    }
  }, [canUseTerminal, setCreatedTerminal, workspaceRoot]);

  useEffect(() => {
    if (
      !canUseTerminal ||
      !open ||
      !terminalsLoaded ||
      creatingTerminal ||
      error ||
      state.tabs.length > 0 ||
      terminals.length > 0
    ) {
      return;
    }
    void addTerminal();
  }, [
    addTerminal,
    canUseTerminal,
    creatingTerminal,
    error,
    open,
    state.tabs.length,
    terminals.length,
    terminalsLoaded,
  ]);

  async function closeTerminal(terminalId: string) {
    try {
      await removeTerminal(workspaceRoot, terminalId);
    } catch (err) {
      if (!isTerminalNotFoundError(err)) {
        setError(err instanceof Error ? err.message : String(err));
        return;
      }
    }
    setTerminals((current) =>
      current.filter((terminal) => terminal.id !== terminalId),
    );
    const remainingTabs = stateRef.current.tabs.filter(
      (tab) => tab.id !== terminalId,
    );
    if (remainingTabs.length === 0) {
      setOpen(false);
    }
    setState((current) => removeTerminalTab(current, terminalId));
    setError("");
  }

  async function renameTerminal(terminal: TerminalInfo) {
    const title = window.prompt("终端名称", terminal.title)?.trim();
    if (!title) return;
    const next = await updateTerminal({
      workspaceRoot,
      terminalId: terminal.id,
      title,
    });
    setTerminals((current) =>
      current.map((item) => (item.id === next.id ? next : item)),
    );
    setState((current) => ({
      ...current,
      tabs: current.tabs.map((tab) =>
        tab.id === next.id ? { ...tab, title: next.title } : tab,
      ),
    }));
  }

  const flushCursorUpdates = useCallback(() => {
    const updates = cursorUpdatesRef.current;
    cursorUpdatesRef.current = {};
    setState((current) => {
      let changed = false;
      const tabs = current.tabs.map((tab) => {
        const cursor = updates[tab.id];
        if (typeof cursor !== "number" || tab.cursor === cursor) return tab;
        changed = true;
        return { ...tab, cursor };
      });
      return changed ? { ...current, tabs } : current;
    });
  }, [setState]);

  const setTerminalCursor = useCallback(
    (terminalId: string, cursor: number) => {
      cursorUpdatesRef.current[terminalId] = cursor;
      if (cursorUpdateTimerRef.current !== null) return;
      cursorUpdateTimerRef.current = window.setTimeout(() => {
        cursorUpdateTimerRef.current = null;
        flushCursorUpdates();
      }, 200);
    },
    [flushCursorUpdates],
  );

  const setTerminalSize = useCallback((terminalId: string, rows: number, cols: number) => {
    setState((current) => {
      let changed = false;
      const tabs = current.tabs.map((tab) => {
        if (tab.id !== terminalId) return tab;
        if (tab.rows === rows && tab.cols === cols) return tab;
        changed = true;
        return { ...tab, rows, cols };
      });
      return changed ? { ...current, tabs } : current;
    });
  }, [setState]);

  const setTerminalStatus = useCallback((
    terminalId: string,
    status: "connecting" | "running" | "reconnecting" | "exited" | "failed",
  ) => {
    const nextStatus = status === "reconnecting" ? "connecting" : status;
    setState((current) => {
      let changed = false;
      const tabs = current.tabs.map((tab) => {
        if (tab.id !== terminalId) return tab;
        if (tab.status === nextStatus) return tab;
        changed = true;
        return { ...tab, status: nextStatus };
      });
      return changed ? { ...current, tabs } : current;
    });
  }, [setState]);

  if (!canOpenPanel) return null;

  return (
          <section className="flex h-full min-h-0 flex-col overflow-hidden bg-card shadow-xl shadow-foreground/10">
            <div className="flex shrink-0 items-center gap-2 border-t border-border/70 px-4 py-1">
              <Button
                aria-label="收起终端"
                className="size-7"
                onClick={() => setOpen(false)}
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <ChevronDown />
              </Button>
              <div className="flex min-w-0 flex-1 items-center gap-1.5 overflow-x-auto">
                {state.tabs.map((tab) => {
                  const terminal = terminals.find((item) => item.id === tab.id);
                  const active = tab.id === activeTerminal?.id;
                  return (
                    <div
                      className={cn(
                        "group/tab relative flex min-w-0 shrink-0 items-center gap-1.5 rounded-lg py-1 text-xs font-medium transition-colors",
                        active
                          ? "bg-muted pl-2 pr-4 text-foreground"
                          : "pl-1.5 pr-3.5 text-muted-foreground hover:bg-muted/60 hover:text-foreground",
                      )}
                      key={tab.id}
                    >
                      <button
                        className="flex min-w-0 items-center gap-1.5"
                        onClick={() => {
                          setOpen(true);
                          setState((current) => ({
                            ...current,
                            opened: true,
                            activeId: tab.id,
                          }));
                        }}
                        onDoubleClick={() =>
                          terminal && void renameTerminal(terminal)
                        }
                        type="button"
                      >
                        <TerminalSquare className="size-3.5 shrink-0" />
                        <span className="max-w-28 truncate">
                          {terminal?.title || tab.title}
                        </span>
                      </button>
                      <Button
                        aria-label="关闭终端"
                        className={cn(
                          "absolute right-1 top-1/2 size-4 shrink-0 !-translate-y-1/2 rounded-full active:!-translate-y-1/2 [&_svg:not([class*='size-'])]:size-2",
                          active
                            ? "invisible inline-flex group-hover/tab:visible focus-visible:visible"
                            : "hidden",
                        )}
                        onClick={() => void closeTerminal(tab.id)}
                        size="icon-xs"
                        type="button"
                      >
                        <X />
                      </Button>
                    </div>
                  );
                })}
                <Button
                  aria-label="新建终端"
                  className="shrink-0 text-muted-foreground hover:text-foreground"
                  onClick={() => void addTerminal()}
                  size="icon-sm"
                  type="button"
                  variant="ghost"
                >
                  <Plus />
                </Button>
              </div>
            </div>
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {!canUseTerminal ? (
                <div className="flex h-full items-center justify-center px-4 text-xs text-muted-foreground">
                  终端暂不可用
                </div>
              ) : error ? (
                <div className="flex h-full items-center justify-center px-4 text-xs text-destructive">
                  {error}
                </div>
              ) : activeTerminal ? (
                <TerminalView
                  key={activeTerminal.id}
                  onCursor={setTerminalCursor}
                  onResize={setTerminalSize}
                  onStatus={setTerminalStatus}
                  terminal={activeTerminal}
                  workspaceRoot={workspaceRoot}
                />
              ) : (
                <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
                  {creatingTerminal || !terminalsLoaded
                    ? "正在创建终端"
                    : "没有终端"}
                </div>
              )}
            </div>
          </section>
  );
}
function nextTitleNumber(state: WorkspaceTerminalState) {
  return (
    Math.max(0, ...state.tabs.map((tab) => tab.titleNumber).filter(Boolean)) + 1
  );
}

function removeTerminalTab(
  state: WorkspaceTerminalState,
  terminalId: string,
): WorkspaceTerminalState {
  const tabs = state.tabs.filter((tab) => tab.id !== terminalId);
  return {
    ...state,
    activeId:
      state.activeId === terminalId || tabs.length === 0
        ? tabs[0]?.id
        : state.activeId,
    opened: tabs.length > 0 ? state.opened : false,
    tabs,
  };
}

function upsertTerminal(terminals: TerminalInfo[], terminal: TerminalInfo) {
  if (terminals.some((item) => item.id === terminal.id)) {
    return terminals.map((item) => (item.id === terminal.id ? terminal : item));
  }
  return [...terminals, terminal];
}

function shouldRetryWithPOSIXShell(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  return (
    message.includes("/bin/zsh") &&
    message.toLowerCase().includes("operation not permitted")
  );
}

function isTerminalNotFoundError(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  return message.includes("terminal ") && message.includes(" not found");
}
