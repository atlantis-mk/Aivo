import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { EventsOn } from "../../../../bridge/runtime/runtime";
import { useTerminalDock } from "@/features/projects/terminal/terminal-dock-store";
import {
  applyTerminalCursorUpdates,
  isTerminalNotFoundError,
  nextTerminalTitleNumber,
  removeTerminalTab,
  renameTerminalTab,
  resizeTerminalTab,
  selectTerminalTab,
  setTerminalTabStatus,
  shouldRetryWithPOSIXShell,
  syncTerminalTabsWithLiveTerminals,
  upsertCreatedTerminalTab,
  upsertTerminalInfo,
} from "@/features/projects/terminal/terminal-panel-model";
import type { TerminalPanelViewProps } from "@/features/projects/terminal/terminal-panel-view";
import {
  clampTerminalHeight,
  defaultTerminalState,
  useTerminalWorkspaceStateStore,
  type WorkspaceTerminalState,
} from "@/features/projects/terminal/terminal-state";
import type { TerminalRuntimeStatus } from "@/features/projects/terminal/terminal-types";
import {
  createTerminal,
  listTerminals,
  removeTerminal,
  updateTerminal,
  type TerminalEventPayload,
  type TerminalInfo,
} from "@/services/terminal";

const EMPTY_TERMINAL_STATE = defaultTerminalState();

export type TerminalPanelControllerProps = {
  enabled: boolean;
  height?: number;
  terminalEnabled?: boolean;
  workspaceRoot: string;
};

export function useTerminalPanelController({
  enabled,
  height,
  terminalEnabled = true,
  workspaceRoot,
}: TerminalPanelControllerProps): {
  canOpenPanel: boolean;
  viewProps: TerminalPanelViewProps;
} {
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
  const dismissedTerminalIdsRef = useRef<Set<string>>(new Set());
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
    dismissedTerminalIdsRef.current.clear();
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
        (terminal) =>
          terminal.status !== "removed" &&
          !dismissedTerminalIdsRef.current.has(terminal.id),
      );
      setTerminals(liveTerminals);
      setTerminalsLoaded(true);
      setError("");
      setState((current) =>
        syncTerminalTabsWithLiveTerminals(current, liveTerminals),
      );
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
      setTerminals((current) => upsertTerminalInfo(current, terminal));
      setTerminalsLoaded(true);
      setOpen(true);
      setState((current) =>
        upsertCreatedTerminalTab(current, terminal, titleNumber),
      );
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
      const titleNumber = nextTerminalTitleNumber(stateRef.current);
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
    if (dismissedTerminalIdsRef.current.has(terminalId)) return;
    dismissedTerminalIdsRef.current.add(terminalId);

    // Closing a tab is an immediate UI action. Process termination continues
    // in the background and may take up to the graceful shutdown timeout.
    setTerminals((current) =>
      current.filter((terminal) => terminal.id !== terminalId),
    );
    const remainingTabs = stateRef.current.tabs.filter(
      (tab) => tab.id !== terminalId,
    );
    setState((current) => removeTerminalTab(current, terminalId));
    setError("");
    if (remainingTabs.length === 0) {
      setOpen(false);
    }

    try {
      await removeTerminal(workspaceRoot, terminalId);
    } catch (err) {
      if (!isTerminalNotFoundError(err)) {
        setError(err instanceof Error ? err.message : String(err));
      }
    }
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
    setState((current) => renameTerminalTab(current, next));
  }

  const selectTerminal = useCallback(
    (terminalId: string) => {
      setOpen(true);
      setState((current) => selectTerminalTab(current, terminalId));
    },
    [setOpen, setState],
  );

  const flushCursorUpdates = useCallback(() => {
    const updates = cursorUpdatesRef.current;
    cursorUpdatesRef.current = {};
    setState((current) => applyTerminalCursorUpdates(current, updates));
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

  const setTerminalSize = useCallback(
    (terminalId: string, rows: number, cols: number) => {
      setState((current) => resizeTerminalTab(current, terminalId, rows, cols));
    },
    [setState],
  );

  const setTerminalStatus = useCallback(
    (
      terminalId: string,
      status: TerminalRuntimeStatus,
    ) => {
      setState((current) => setTerminalTabStatus(current, terminalId, status));
    },
    [setState],
  );

  return {
    canOpenPanel,
    viewProps: {
      activeTerminal,
      canUseTerminal,
      creatingTerminal,
      error,
      onAddTerminal: () => void addTerminal(),
      onCloseTerminal: (terminalId) => void closeTerminal(terminalId),
      onCollapse: () => setOpen(false),
      onRenameTerminal: (terminal) => void renameTerminal(terminal),
      onSelectTerminal: selectTerminal,
      onTerminalCursor: setTerminalCursor,
      onTerminalResize: setTerminalSize,
      onTerminalStatus: setTerminalStatus,
      state,
      terminals,
      terminalsLoaded,
      workspaceRoot,
    },
  };
}
