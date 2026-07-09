import { useCallback, useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";
import { useNavigate, useRouterState } from "@tanstack/react-router";

import { getActiveProjectPage } from "@/features/projects/project-workspace-derived-state";

export function useProjectWorkspaceRouteState({
  activeSessionId,
  setPinnedSummaryOpen,
}: {
  activeSessionId: string;
  setPinnedSummaryOpen: Dispatch<SetStateAction<boolean>>;
}) {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const activeProjectPage = getActiveProjectPage(pathname);
  const navigateToProjectChat = useCallback(() => {
    void navigate({ to: "/projects/chat" });
  }, [navigate]);

  useEffect(() => {
    if (activeProjectPage === "chat" && activeSessionId) return;
    setPinnedSummaryOpen(false);
  }, [activeProjectPage, activeSessionId, setPinnedSummaryOpen]);

  useEffect(() => {
    if (pathname === "/projects") {
      void navigate({ to: "/projects/chat", replace: true });
    }
  }, [navigate, pathname]);

  return {
    activeProjectPage,
    navigateToProjectChat,
  };
}
