import { useProjectPreferencesStore } from "@/features/projects/project-preferences-store";

export function useProjectWorkspacePreferencesState() {
  const pinnedConversationIds = useProjectPreferencesStore(
    (state) => state.pinnedConversationIds,
  );
  const setPinnedConversationIds = useProjectPreferencesStore(
    (state) => state.setPinnedConversationIds,
  );
  const archivedConversationIds = useProjectPreferencesStore(
    (state) => state.archivedConversationIds,
  );
  const setArchivedConversationIds = useProjectPreferencesStore(
    (state) => state.setArchivedConversationIds,
  );
  const pendingActiveToolNames = useProjectPreferencesStore(
    (state) => state.pendingActiveToolNames,
  );
  const setPendingActiveToolNames = useProjectPreferencesStore(
    (state) => state.setPendingActiveToolNames,
  );
  const hiddenTodoPlanKeys = useProjectPreferencesStore(
    (state) => state.hiddenTodoPlanKeys,
  );
  const setHiddenTodoPlanKeyForSession = useProjectPreferencesStore(
    (state) => state.setHiddenTodoPlanKey,
  );

  return {
    archivedConversationIds,
    pendingActiveToolNames,
    hiddenTodoPlanKeys,
    pinnedConversationIds,
    setArchivedConversationIds,
    setPendingActiveToolNames,
    setHiddenTodoPlanKeyForSession,
    setPinnedConversationIds,
  };
}
