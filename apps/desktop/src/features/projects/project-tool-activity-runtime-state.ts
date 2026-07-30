import { useEffect, useRef, useState } from "react";

import {
  buildToolActivitySessionState,
  type ToolActivitySessionState,
} from "@/features/projects/project-tool-activity-session-model";
import type { ToolActivityTab } from "@/features/projects/tool-activity-model";

export function useProjectToolActivityRuntimeState() {
  const [isRightSidebarOpen, setRightSidebarOpen] = useState(false);
  const [toolActivityTabs, setToolActivityTabs] = useState<ToolActivityTab[]>(
    [],
  );
  const [activeToolActivityTabId, setActiveToolActivityTabId] = useState("");
  const toolActivitySessionStatesRef = useRef(
    new Map<string, ToolActivitySessionState>(),
  );
  const toolActivityTabsRef = useRef<ToolActivityTab[]>([]);
  const activeToolActivityTabIdRef = useRef("");
  const isRightSidebarOpenRef = useRef(false);
  const closedToolActivityItemIdsRef = useRef(new Set<string>());

  return {
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    closedToolActivityItemIdsRef,
    isRightSidebarOpen,
    isRightSidebarOpenRef,
    setActiveToolActivityTabId,
    setRightSidebarOpen,
    setToolActivityTabs,
    toolActivitySessionStatesRef,
    toolActivityTabs,
    toolActivityTabsRef,
  };
}

export function useProjectToolActivitySessionPersistence({
  activeSessionId,
  activeToolActivityTabId,
  activeToolActivityTabIdRef,
  closedToolActivityItemIdsRef,
  isRightSidebarOpen,
  isRightSidebarOpenRef,
  toolActivitySessionStatesRef,
  toolActivityTabs,
  toolActivityTabsRef,
}: {
  activeSessionId: string;
  activeToolActivityTabId: string;
  activeToolActivityTabIdRef: { current: string };
  closedToolActivityItemIdsRef: { current: Set<string> };
  isRightSidebarOpen: boolean;
  isRightSidebarOpenRef: { current: boolean };
  toolActivitySessionStatesRef: {
    current: Map<string, ToolActivitySessionState>;
  };
  toolActivityTabs: ToolActivityTab[];
  toolActivityTabsRef: { current: ToolActivityTab[] };
}) {
  useEffect(() => {
    toolActivityTabsRef.current = toolActivityTabs;
    activeToolActivityTabIdRef.current = activeToolActivityTabId;
    isRightSidebarOpenRef.current = isRightSidebarOpen;
    if (!activeSessionId) return;
    toolActivitySessionStatesRef.current.set(
      activeSessionId,
      buildToolActivitySessionState({
        activeTabId: activeToolActivityTabId,
        closedItemIds: closedToolActivityItemIdsRef.current,
        isOpen: isRightSidebarOpen,
        tabs: toolActivityTabs,
      }),
    );
  }, [
    activeSessionId,
    activeToolActivityTabId,
    activeToolActivityTabIdRef,
    closedToolActivityItemIdsRef,
    isRightSidebarOpen,
    isRightSidebarOpenRef,
    toolActivitySessionStatesRef,
    toolActivityTabs,
    toolActivityTabsRef,
  ]);
}
