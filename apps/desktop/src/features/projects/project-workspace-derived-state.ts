import { useMemo } from "react";

import {
  getActiveProvider,
  getAllModelOptions,
  getConnectedModelProviders,
  getDefaultModelId,
  getModelOptions,
  normalizeModelId,
} from "@/features/projects/project-model-options";
import {
  buildProjectConversationGroups,
  filterSessionsOutsideProjectGroups,
} from "@/features/projects/project-sidebar-model";
import {
  getTodoPlanKey,
  isTodoPlanComplete,
} from "@/features/projects/project-todo-status-model";
import type { CatalogState } from "@/lib/provider-catalog";
import type { TodoItem } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export type ProjectWorkspacePage = "chat" | "extensions";

export function getActiveProjectPage(pathname: string): ProjectWorkspacePage {
  return pathname.startsWith("/projects/extensions") ? "extensions" : "chat";
}

export function useProjectModelSelection({
  catalog,
  config,
  selectedModelId,
  selectedProviderId,
}: {
  catalog: CatalogState | null;
  config: domain.AppConfig | null;
  selectedModelId: string;
  selectedProviderId: string;
}) {
  const modelProviders = useMemo(
    () =>
      getConnectedModelProviders(
        config,
        catalog?.providers ?? [],
        catalog?.connectedProviders ?? [],
      ),
    [catalog?.connectedProviders, catalog?.providers, config],
  );
  const activeProvider = useMemo(
    () => getActiveProvider(config, modelProviders, selectedProviderId),
    [config, modelProviders, selectedProviderId],
  );
  const modelOptions = useMemo(
    () => getModelOptions(activeProvider, catalog?.models ?? []),
    [activeProvider, catalog?.models],
  );
  const allModelOptions = useMemo(
    () => getAllModelOptions(modelProviders, catalog?.models ?? []),
    [catalog?.models, modelProviders],
  );
  const defaultModelId = getDefaultModelId(
    config,
    activeProvider,
    modelOptions,
  );
  const modelOptionsKey = modelOptions.map((model) => model.id).join("|");
  const activeModelId = normalizeModelId(
    activeProvider?.id,
    selectedModelId || defaultModelId,
  );
  const activeModelRef =
    activeProvider && activeModelId
      ? { providerId: activeProvider.id, modelId: activeModelId }
      : undefined;

  return {
    activeModelId,
    activeModelRef,
    activeProvider,
    allModelOptions,
    defaultModelId,
    modelOptions,
    modelOptionsKey,
    modelProviders,
  };
}

export function useProjectSidebarConversationState({
  archivedConversationIds,
  recentProjects,
  selectedProjectPath,
  sessions,
}: {
  archivedConversationIds: string[];
  recentProjects: domain.AssistantProject[];
  selectedProjectPath: string;
  sessions: domain.Session[];
}) {
  const visibleSessions = useMemo(
    () =>
      sessions.filter(
        (session) =>
          !session.parentSessionId &&
          !archivedConversationIds.includes(session.id),
      ),
    [archivedConversationIds, sessions],
  );
  const projectConversationGroups = useMemo(
    () =>
      buildProjectConversationGroups(
        visibleSessions,
        recentProjects,
        selectedProjectPath,
      ),
    [recentProjects, selectedProjectPath, visibleSessions],
  );
  const visibleConversations = useMemo(
    () =>
      filterSessionsOutsideProjectGroups(
        visibleSessions,
        projectConversationGroups,
      ),
    [projectConversationGroups, visibleSessions],
  );

  return {
    projectConversationGroups,
    visibleConversations,
    visibleSessions,
  };
}

export function useProjectTodoFloatingState({
  hiddenTodoPlanKey,
  visibleTodoPlanItems,
}: {
  hiddenTodoPlanKey: string;
  visibleTodoPlanItems: TodoItem[];
}) {
  const visibleTodoPlanKey = useMemo(
    () => getTodoPlanKey(visibleTodoPlanItems),
    [visibleTodoPlanItems],
  );
  const isVisibleTodoPlanComplete = isTodoPlanComplete(visibleTodoPlanItems);
  const shouldShowTodoFloatingStatus =
    visibleTodoPlanItems.length > 0 &&
    visibleTodoPlanKey !== hiddenTodoPlanKey;

  return {
    isVisibleTodoPlanComplete,
    shouldShowTodoFloatingStatus,
    visibleTodoPlanKey,
  };
}
