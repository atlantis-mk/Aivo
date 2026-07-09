import { create } from "zustand";
import type { PersistedTerminalStatus } from "@/features/projects/terminal/terminal-types";

export type WorkspaceTerminalState = {
  opened: boolean;
  height: number;
  activeId?: string;
  tabs: Array<{
    id: string;
    title: string;
    titleNumber: number;
    rows?: number;
    cols?: number;
    cursor?: number;
    scrollY?: number;
    bufferSnapshot?: string;
    status?: PersistedTerminalStatus;
  }>;
};

export const DEFAULT_TERMINAL_HEIGHT = 300;
export const MIN_TERMINAL_HEIGHT = 180;
export const MAX_TERMINAL_HEIGHT = 420;

type TerminalStateUpdater =
  | WorkspaceTerminalState
  | ((current: WorkspaceTerminalState) => WorkspaceTerminalState);

type TerminalWorkspaceStateStore = {
  workspaceStates: Record<string, WorkspaceTerminalState>;
  loadWorkspaceState: (workspaceRoot: string) => WorkspaceTerminalState;
  setWorkspaceState: (
    workspaceRoot: string,
    updater: TerminalStateUpdater,
  ) => void;
};

function stateKey(workspaceRoot: string) {
  return `aivo:terminal-state:${workspaceRoot}`;
}

export const useTerminalWorkspaceStateStore =
  create<TerminalWorkspaceStateStore>((set) => ({
    workspaceStates: {},
    loadWorkspaceState: (workspaceRoot) => {
      const nextState = workspaceRoot
        ? readWorkspaceTerminalState(workspaceRoot)
        : defaultTerminalState();
      if (workspaceRoot) {
        set((state) => ({
          workspaceStates: {
            ...state.workspaceStates,
            [workspaceRoot]: nextState,
          },
        }));
      }
      return nextState;
    },
    setWorkspaceState: (workspaceRoot, updater) => {
      if (!workspaceRoot) return;
      set((store) => {
        const current =
          store.workspaceStates[workspaceRoot] ??
          readWorkspaceTerminalState(workspaceRoot);
        const nextState =
          typeof updater === "function"
            ? updater(current)
            : updater;
        if (Object.is(nextState, current)) return store;
        writeWorkspaceTerminalState(workspaceRoot, nextState);
        return {
          workspaceStates: {
            ...store.workspaceStates,
            [workspaceRoot]: nextState,
          },
        };
      });
    },
  }));

export function clampTerminalHeight(height: number) {
  if (!Number.isFinite(height)) return DEFAULT_TERMINAL_HEIGHT;
  return Math.min(MAX_TERMINAL_HEIGHT, Math.max(MIN_TERMINAL_HEIGHT, height));
}

export function readWorkspaceTerminalState(
  workspaceRoot: string,
): WorkspaceTerminalState {
  if (typeof window === "undefined" || !workspaceRoot) {
    return defaultTerminalState();
  }
  try {
    const parsed = JSON.parse(
      window.localStorage.getItem(stateKey(workspaceRoot)) ?? "null",
    ) as WorkspaceTerminalState | null;
    if (!parsed || typeof parsed !== "object") return defaultTerminalState();
    return {
      opened: parsed.opened === true,
      height: clampTerminalHeight(parsed.height),
      activeId: typeof parsed.activeId === "string" ? parsed.activeId : undefined,
      tabs: Array.isArray(parsed.tabs)
        ? parsed.tabs.flatMap((tab) =>
            tab && typeof tab.id === "string"
              ? [
                  {
                    id: tab.id,
                    title: typeof tab.title === "string" ? tab.title : "Shell",
                    titleNumber:
                      typeof tab.titleNumber === "number" ? tab.titleNumber : 1,
                    rows: typeof tab.rows === "number" ? tab.rows : undefined,
                    cols: typeof tab.cols === "number" ? tab.cols : undefined,
                    cursor:
                      typeof tab.cursor === "number" ? tab.cursor : undefined,
                    scrollY:
                      typeof tab.scrollY === "number" ? tab.scrollY : undefined,
                    bufferSnapshot:
                      typeof tab.bufferSnapshot === "string"
                        ? tab.bufferSnapshot.slice(-32_000)
                        : undefined,
                    status:
                      tab.status === "connecting" ||
                      tab.status === "running" ||
                      tab.status === "exited" ||
                      tab.status === "failed"
                        ? tab.status
                        : undefined,
                  },
                ]
              : [],
          )
        : [],
    };
  } catch {
    return defaultTerminalState();
  }
}

export function writeWorkspaceTerminalState(
  workspaceRoot: string,
  state: WorkspaceTerminalState,
) {
  if (typeof window === "undefined" || !workspaceRoot) return;
  const safeState: WorkspaceTerminalState = {
    ...state,
    height: clampTerminalHeight(state.height),
    tabs: state.tabs.map((tab) => ({
      ...tab,
      bufferSnapshot: tab.bufferSnapshot?.slice(-32_000),
    })),
  };
  window.localStorage.setItem(stateKey(workspaceRoot), JSON.stringify(safeState));
}

export function defaultTerminalState(): WorkspaceTerminalState {
  return {
    opened: false,
    height: DEFAULT_TERMINAL_HEIGHT,
    tabs: [],
  };
}
