import { create } from "zustand";

import { DEFAULT_TERMINAL_HEIGHT } from "@/features/projects/terminal/terminal-state";
import { discardLegacyDefaultActiveToolNames } from "@/features/projects/project-tool-activation-scope";

const PINNED_CONVERSATION_IDS_STORAGE_KEY = "aivo:pinned-conversation-ids";
const ARCHIVED_CONVERSATION_IDS_STORAGE_KEY = "aivo:archived-conversation-ids";
const PROJECT_PANEL_LAYOUT_STORAGE_KEY = "aivo:project-panel-layout:v1";
const HIDDEN_TODO_PLAN_KEYS_STORAGE_KEY = "aivo:hidden-todo-plan-keys:v1";
const PROJECT_LEFT_SIDEBAR_WIDTH = 260;
const PROJECT_LEFT_SIDEBAR_MIN_WIDTH = 210;
const PROJECT_RIGHT_SIDEBAR_WIDTH = 360;
const PROJECT_RIGHT_SIDEBAR_MIN_WIDTH = 240;
const PROJECT_BOTTOM_PANEL_MIN_HEIGHT = 180;
const PROJECT_MAIN_MIN_WIDTH = 360;
const PROJECT_UPPER_MIN_HEIGHT = 240;

type StoreUpdater<T> = T | ((current: T) => T);

export type ProjectPanelLayout = {
  bottomHeight: number;
  leftWidth: number;
  rightWidth: number;
};

type ProjectPreferencesState = {
  archivedConversationIds: string[];
  pendingActiveToolNames: string[];
  hiddenTodoPlanKeys: Record<string, string>;
  panelLayout: ProjectPanelLayout;
  pinnedConversationIds: string[];
  setArchivedConversationIds: (updater: StoreUpdater<string[]>) => void;
  setPendingActiveToolNames: (updater: StoreUpdater<string[]>) => void;
  setHiddenTodoPlanKey: (sessionId: string, planKey: string) => void;
  setPanelLayout: (layout: ProjectPanelLayout) => void;
  setPinnedConversationIds: (updater: StoreUpdater<string[]>) => void;
};

export const DEFAULT_PROJECT_PANEL_LAYOUT: ProjectPanelLayout = {
  bottomHeight: DEFAULT_TERMINAL_HEIGHT,
  leftWidth: PROJECT_LEFT_SIDEBAR_WIDTH,
  rightWidth: PROJECT_RIGHT_SIDEBAR_WIDTH,
};

export const useProjectPreferencesStore = create<ProjectPreferencesState>(
  (set) => ({
    archivedConversationIds: readStoredStringArray(
      ARCHIVED_CONVERSATION_IDS_STORAGE_KEY,
    ),
    pendingActiveToolNames: discardLegacyDefaultActiveToolNames(
      typeof window === "undefined" ? undefined : window.localStorage,
    ),
    hiddenTodoPlanKeys: readHiddenTodoPlanKeys(),
    panelLayout: readProjectPanelLayout(),
    pinnedConversationIds: readStoredStringArray(
      PINNED_CONVERSATION_IDS_STORAGE_KEY,
    ),
    setArchivedConversationIds: (updater) =>
      set((state) => {
        const archivedConversationIds = uniqueStrings(
          resolveUpdater(updater, state.archivedConversationIds),
        );
        writeStoredStringArray(
          ARCHIVED_CONVERSATION_IDS_STORAGE_KEY,
          archivedConversationIds,
        );
        return { archivedConversationIds };
      }),
    setPendingActiveToolNames: (updater) =>
      set((state) => {
        const pendingActiveToolNames = uniqueStrings(
          resolveUpdater(updater, state.pendingActiveToolNames),
        );
        return { pendingActiveToolNames };
      }),
    setHiddenTodoPlanKey: (sessionId, planKey) =>
      set((state) => {
        if (!sessionId) return state;
        const hiddenTodoPlanKeys = { ...state.hiddenTodoPlanKeys };
        if (planKey) {
          hiddenTodoPlanKeys[sessionId] = planKey;
        } else {
          delete hiddenTodoPlanKeys[sessionId];
        }
        writeHiddenTodoPlanKeys(hiddenTodoPlanKeys);
        return { hiddenTodoPlanKeys };
      }),
    setPanelLayout: (layout) =>
      set(() => {
        const panelLayout = clampProjectPanelLayout(layout);
        writeProjectPanelLayout(panelLayout);
        return { panelLayout };
      }),
    setPinnedConversationIds: (updater) =>
      set((state) => {
        const pinnedConversationIds = uniqueStrings(
          resolveUpdater(updater, state.pinnedConversationIds),
        );
        writeStoredStringArray(
          PINNED_CONVERSATION_IDS_STORAGE_KEY,
          pinnedConversationIds,
        );
        return { pinnedConversationIds };
      }),
  }),
);

function resolveUpdater<T>(updater: StoreUpdater<T>, current: T) {
  return typeof updater === "function"
    ? (updater as (current: T) => T)(current)
    : updater;
}

function uniqueStrings(value: string[]) {
  return [...new Set(value.filter((item) => typeof item === "string"))];
}

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(Math.max(value, min), Math.max(min, max));
}

function clampProjectPanelLayout(layout: Partial<ProjectPanelLayout>) {
  if (typeof window === "undefined") return DEFAULT_PROJECT_PANEL_LAYOUT;
  return {
    bottomHeight: clampNumber(
      Number(layout.bottomHeight),
      PROJECT_BOTTOM_PANEL_MIN_HEIGHT,
      window.innerHeight - PROJECT_UPPER_MIN_HEIGHT,
    ),
    leftWidth: clampNumber(
      Number(layout.leftWidth),
      PROJECT_LEFT_SIDEBAR_MIN_WIDTH,
      window.innerWidth - PROJECT_MAIN_MIN_WIDTH,
    ),
    rightWidth: clampNumber(
      Number(layout.rightWidth),
      PROJECT_RIGHT_SIDEBAR_MIN_WIDTH,
      window.innerWidth - PROJECT_MAIN_MIN_WIDTH,
    ),
  };
}

function readProjectPanelLayout(): ProjectPanelLayout {
  if (typeof window === "undefined") return DEFAULT_PROJECT_PANEL_LAYOUT;
  try {
    const raw = window.localStorage.getItem(PROJECT_PANEL_LAYOUT_STORAGE_KEY);
    if (!raw) return DEFAULT_PROJECT_PANEL_LAYOUT;
    return clampProjectPanelLayout(JSON.parse(raw) as Partial<ProjectPanelLayout>);
  } catch {
    return DEFAULT_PROJECT_PANEL_LAYOUT;
  }
}

function writeProjectPanelLayout(layout: ProjectPanelLayout) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(
    PROJECT_PANEL_LAYOUT_STORAGE_KEY,
    JSON.stringify(layout),
  );
}

function readStoredStringArray(key: string) {
  if (typeof window === "undefined") return [];
  try {
    const parsed = JSON.parse(window.localStorage.getItem(key) ?? "[]");
    if (!Array.isArray(parsed)) return [];
    return uniqueStrings(parsed.filter((item): item is string => typeof item === "string"));
  } catch {
    return [];
  }
}

function writeStoredStringArray(key: string, value: string[]) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(key, JSON.stringify(uniqueStrings(value)));
}

function readHiddenTodoPlanKeys() {
  if (typeof window === "undefined") return {};
  try {
    const parsed = JSON.parse(
      window.localStorage.getItem(HIDDEN_TODO_PLAN_KEYS_STORAGE_KEY) ?? "{}",
    );
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return Object.fromEntries(
      Object.entries(parsed).filter(
        (entry): entry is [string, string] =>
          typeof entry[0] === "string" && typeof entry[1] === "string",
      ),
    );
  } catch {
    return {};
  }
}

function writeHiddenTodoPlanKeys(keys: Record<string, string>) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(
    HIDDEN_TODO_PLAN_KEYS_STORAGE_KEY,
    JSON.stringify(keys),
  );
}
