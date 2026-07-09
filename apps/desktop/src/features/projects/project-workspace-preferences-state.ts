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
  const defaultActiveToolNames = useProjectPreferencesStore(
    (state) => state.defaultActiveToolNames,
  );
  const setDefaultActiveToolNames = useProjectPreferencesStore(
    (state) => state.setDefaultActiveToolNames,
  );
  const hiddenTodoPlanKeys = useProjectPreferencesStore(
    (state) => state.hiddenTodoPlanKeys,
  );
  const setHiddenTodoPlanKeyForSession = useProjectPreferencesStore(
    (state) => state.setHiddenTodoPlanKey,
  );

  return {
    archivedConversationIds,
    defaultActiveToolNames,
    hiddenTodoPlanKeys,
    pinnedConversationIds,
    setArchivedConversationIds,
    setDefaultActiveToolNames,
    setHiddenTodoPlanKeyForSession,
    setPinnedConversationIds,
  };
}
