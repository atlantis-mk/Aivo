import { useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";

import { hasCodexDesktopBridge } from "@/lib/app-config";
import { listCodexSessions } from "@/services/codex-thread-service";
import type { domain } from "@/types/codex-domain";

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
    if (!hasCodexDesktopBridge()) return;
    void listCodexSessions(50)
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
