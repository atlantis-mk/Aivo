import { useState } from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { domain } from "../../../bridge/go/models";

export function useProjectWorkspaceScreenState() {
  const [prompt, setPrompt] = useState("");
  const [turns, setTurns] = useState<ConversationTurn[]>([]);
  const [sessions, setSessions] = useState<domain.Session[]>([]);
  const [activeSessionId, setActiveSessionId] = useState("");
  const [toolActivationDialogOpen, setToolActivationDialogOpen] =
    useState(false);
  const [extensionSettingsDrawerOpen, setExtensionSettingsDrawerOpen] =
    useState(false);
  const [isOpeningConversationFromEmpty, setOpeningConversationFromEmpty] =
    useState(false);
  const [isRevealingHistoryConversation, setRevealingHistoryConversation] =
    useState(false);
  const [recentProjects, setRecentProjects] = useState<
    domain.AssistantProject[]
  >([]);
  const [selectedProjectPath, setSelectedProjectPath] = useState("");

  return {
    activeSessionId,
    isOpeningConversationFromEmpty,
    isRevealingHistoryConversation,
    prompt,
    extensionSettingsDrawerOpen,
    recentProjects,
    selectedProjectPath,
    sessions,
    setActiveSessionId,
    setOpeningConversationFromEmpty,
    setExtensionSettingsDrawerOpen,
    setPrompt,
    setRecentProjects,
    setRevealingHistoryConversation,
    setSelectedProjectPath,
    setSessions,
    setToolActivationDialogOpen,
    setTurns,
    toolActivationDialogOpen,
    turns,
  };
}
