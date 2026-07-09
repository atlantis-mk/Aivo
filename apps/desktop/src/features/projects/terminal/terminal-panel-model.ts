import type { WorkspaceTerminalState } from "@/features/projects/terminal/terminal-state";
import type {
  PersistedTerminalStatus,
  TerminalRuntimeStatus,
} from "@/features/projects/terminal/terminal-types";
import type { TerminalInfo } from "@/services/terminal";

export function nextTerminalTitleNumber(state: WorkspaceTerminalState) {
  return (
    Math.max(0, ...state.tabs.map((tab) => tab.titleNumber).filter(Boolean)) + 1
  );
}

export function syncTerminalTabsWithLiveTerminals(
  current: WorkspaceTerminalState,
  liveTerminals: TerminalInfo[],
): WorkspaceTerminalState {
  const liveIds = new Set(liveTerminals.map((terminal) => terminal.id));
  const tabs = current.tabs.filter((tab) => liveIds.has(tab.id));
  const tabIds = new Set(tabs.map((tab) => tab.id));
  let titleNumber = nextTerminalTitleNumber({ ...current, tabs });
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
}

export function upsertCreatedTerminalTab(
  current: WorkspaceTerminalState,
  terminal: TerminalInfo,
  titleNumber: number,
): WorkspaceTerminalState {
  return {
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
  };
}

export function removeTerminalTab(
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

export function renameTerminalTab(
  current: WorkspaceTerminalState,
  terminal: TerminalInfo,
): WorkspaceTerminalState {
  return {
    ...current,
    tabs: current.tabs.map((tab) =>
      tab.id === terminal.id ? { ...tab, title: terminal.title } : tab,
    ),
  };
}

export function selectTerminalTab(
  current: WorkspaceTerminalState,
  terminalId: string,
): WorkspaceTerminalState {
  return {
    ...current,
    opened: true,
    activeId: terminalId,
  };
}

export function applyTerminalCursorUpdates(
  current: WorkspaceTerminalState,
  updates: Record<string, number>,
): WorkspaceTerminalState {
  let changed = false;
  const tabs = current.tabs.map((tab) => {
    const cursor = updates[tab.id];
    if (typeof cursor !== "number" || tab.cursor === cursor) return tab;
    changed = true;
    return { ...tab, cursor };
  });
  return changed ? { ...current, tabs } : current;
}

export function resizeTerminalTab(
  current: WorkspaceTerminalState,
  terminalId: string,
  rows: number,
  cols: number,
): WorkspaceTerminalState {
  let changed = false;
  const tabs = current.tabs.map((tab) => {
    if (tab.id !== terminalId) return tab;
    if (tab.rows === rows && tab.cols === cols) return tab;
    changed = true;
    return { ...tab, rows, cols };
  });
  return changed ? { ...current, tabs } : current;
}

export function setTerminalTabStatus(
  current: WorkspaceTerminalState,
  terminalId: string,
  status: TerminalRuntimeStatus,
): WorkspaceTerminalState {
  const nextStatus: PersistedTerminalStatus =
    status === "reconnecting" ? "connecting" : status;
  let changed = false;
  const tabs = current.tabs.map((tab) => {
    if (tab.id !== terminalId) return tab;
    if (tab.status === nextStatus) return tab;
    changed = true;
    return { ...tab, status: nextStatus };
  });
  return changed ? { ...current, tabs } : current;
}

export function upsertTerminalInfo(
  terminals: TerminalInfo[],
  terminal: TerminalInfo,
) {
  if (terminals.some((item) => item.id === terminal.id)) {
    return terminals.map((item) => (item.id === terminal.id ? terminal : item));
  }
  return [...terminals, terminal];
}

export function shouldRetryWithPOSIXShell(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  return (
    message.includes("/bin/zsh") &&
    message.toLowerCase().includes("operation not permitted")
  );
}

export function isTerminalNotFoundError(err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  return message.includes("terminal ") && message.includes(" not found");
}
