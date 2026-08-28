import { toast } from "sonner";
import type { Dispatch, SetStateAction } from "react";

import {
  normalizePermissionMode,
  normalizeReasoningEffort,
  normalizeServiceTier,
  providerSupportsServiceTier,
  type ModelOption,
} from "@/features/projects/project-model-options";
import { hasAppBridge } from "@/lib/app-config";
import {
  cancelAgentRun,
  setPermissionMode,
  setSessionAgentMode,
  updateModelPreferences,
  type AgentModeId,
  type AgentRun,
  type PermissionMode,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

type PermissionModePreferenceInput = domain.ModelPreferencesInput & {
  defaultPermissionMode: PermissionMode;
};

export function useProjectWorkspacePreferenceActions({
  activeModelRef,
  activeRunningSubagentRun,
  activeSessionId,
  activeSessionIdRef,
  permissionModeRef,
  reasoningEffort,
  refreshAgentRuntimeState,
  serviceTier,
  setAgentMode,
  setLocalPermissionMode,
  setReasoningEffort,
  setSelectedModelId,
  setSelectedProviderId,
  setServiceTier,
  setSessions,
}: {
  activeModelRef: domain.ModelRef | undefined;
  activeRunningSubagentRun: AgentRun | undefined;
  activeSessionId: string;
  activeSessionIdRef: { current: string };
  permissionModeRef: { current: PermissionMode };
  reasoningEffort: string;
  refreshAgentRuntimeState: (sessionId?: string) => Promise<void>;
  serviceTier: string;
  setAgentMode: Dispatch<SetStateAction<AgentModeId>>;
  setLocalPermissionMode: Dispatch<SetStateAction<PermissionMode>>;
  setReasoningEffort: Dispatch<SetStateAction<string>>;
  setSelectedModelId: Dispatch<SetStateAction<string>>;
  setSelectedProviderId: Dispatch<SetStateAction<string>>;
  setServiceTier: Dispatch<SetStateAction<string>>;
  setSessions: Dispatch<SetStateAction<domain.Session[]>>;
}) {
  async function rememberModelPreferences(
    model: domain.ModelRef | undefined,
    nextReasoningEffort: string,
    nextServiceTier: string,
  ) {
    if (!model && !nextReasoningEffort && !nextServiceTier) return;
    if (!hasAppBridge()) return;
    try {
      await updateModelPreferences({
        model,
        reasoningEffort: normalizeReasoningEffort(nextReasoningEffort),
        serviceTier: normalizeServiceTier(nextServiceTier),
      } as domain.ModelPreferencesInput);
    } catch {
      // Preference persistence should not block composing or sending a message.
    }
  }

  function selectModel(option: ModelOption) {
    setSelectedProviderId(option.providerId);
    setSelectedModelId(option.id);
    const nextServiceTier = providerSupportsServiceTier(option.providerId)
      ? serviceTier
      : "default";
    setServiceTier(nextServiceTier);
    void rememberModelPreferences(
      { providerId: option.providerId, modelId: option.id },
      reasoningEffort,
      nextServiceTier,
    );
  }

  function selectReasoningEffort(nextReasoningEffort: string) {
    const normalized = normalizeReasoningEffort(nextReasoningEffort);
    setReasoningEffort(normalized);
    void rememberModelPreferences(activeModelRef, normalized, serviceTier);
  }

  function selectServiceTier(nextServiceTier: string) {
    const normalized = normalizeServiceTier(nextServiceTier);
    setServiceTier(normalized);
    void rememberModelPreferences(activeModelRef, reasoningEffort, normalized);
  }

  function selectPermissionMode(nextMode: PermissionMode) {
    const normalized = normalizePermissionMode(nextMode);
    permissionModeRef.current = normalized;
    setLocalPermissionMode(normalized);
    const sessionId = activeSessionIdRef.current;
    if (!hasAppBridge()) return;
    void updateModelPreferences({
      defaultPermissionMode: normalized,
    } as PermissionModePreferenceInput).catch(() => {
      toast.error("默认权限模式保存失败");
    });
    if (sessionId) {
      void setPermissionMode(sessionId, normalized).catch(() => {
        toast.error("当前对话权限模式保存失败");
      });
    }
  }

  function selectAgentMode(nextMode: AgentModeId) {
    setAgentMode(nextMode);
    const sessionId = activeSessionIdRef.current;
    if (!sessionId || !hasAppBridge()) return;
    void setSessionAgentMode(sessionId, nextMode)
      .then((session) => {
        setSessions((currentSessions) =>
          currentSessions.map((currentSession) =>
            currentSession.id === session.id ? session : currentSession,
          ),
        );
      })
      .catch(() => {
        toast.error("Agent 模式保存失败");
      });
  }

  async function cancelActiveSubagentRun() {
    if (!activeRunningSubagentRun?.id || !hasAppBridge()) return;
    try {
      await cancelAgentRun(activeRunningSubagentRun.id);
      await refreshAgentRuntimeState(activeSessionId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "取消子代理失败");
    }
  }

  return {
    cancelActiveSubagentRun,
    selectAgentMode,
    selectModel,
    selectPermissionMode,
    selectReasoningEffort,
    selectServiceTier,
  };
}
