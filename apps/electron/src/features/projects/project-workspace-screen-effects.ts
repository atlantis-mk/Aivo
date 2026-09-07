import { useProjectWorkspaceEvents } from "@/features/projects/project-workspace-events";
import { useProjectWorkspaceLifecycleEffects } from "@/features/projects/project-workspace-lifecycle-effects";

type ProjectWorkspaceScreenEffectsProps = Parameters<
  typeof useProjectWorkspaceEvents
>[0] &
  Parameters<typeof useProjectWorkspaceLifecycleEffects>[0];

export function useProjectWorkspaceScreenEffects({
  cancelPendingAssistantDelta,
  refreshRecentProjects,
  stopComposerTransition,
  stopForceScrollToBottom,
  ...eventProps
}: ProjectWorkspaceScreenEffectsProps) {
  useProjectWorkspaceEvents(eventProps);
  useProjectWorkspaceLifecycleEffects({
    cancelPendingAssistantDelta,
    refreshRecentProjects,
    setSessions: eventProps.setSessions,
    stopComposerTransition,
    stopForceScrollToBottom,
  });
}
