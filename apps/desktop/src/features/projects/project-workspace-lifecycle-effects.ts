import { useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";

import { hasAppBridge, hasCodexDesktopBridge } from "@/lib/app-config";
import { listCodexSessions } from "@/services/codex-thread-service";
import { listSessions } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceLifecycleEffects({
  cancelPendingAssistantDelta,
  refreshRecentProjects,
  setSessions,
  stopComposerTransition,
  stopForceScrollToBottom,
}: {
  cancelPendingAssistantDelta: () => void;
  refreshRecentProjects: () => Promise<void>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
  stopComposerTransition: () => void;
  stopForceScrollToBottom: () => void;
}) {
  useEffect(() => {
    if (!hasAppBridge() && !hasCodexDesktopBridge()) return;
    const loadSessions = hasCodexDesktopBridge() ? listCodexSessions : listSessions;
    void loadSessions(50)
      .then((nextSessions) => setSessions(nextSessions ?? []))
      .catch(() => setSessions([]));
  }, [setSessions]);

  useEffect(() => {
    void refreshRecentProjects();
  }, [refreshRecentProjects]);

  useEffect(() => {
    return () => {
      cancelPendingAssistantDelta();
      stopForceScrollToBottom();
      stopComposerTransition();
    };
  }, [
    cancelPendingAssistantDelta,
    stopComposerTransition,
    stopForceScrollToBottom,
  ]);
}
