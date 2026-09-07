import { toast } from "sonner";
import type { Dispatch, SetStateAction } from "react";

import {
  normalizePermissionMode,
  normalizeReasoningEffort,
  normalizeServiceTier,
  providerSupportsServiceTier,
  type ModelOption,
} from "@/features/projects/project-model-options";
import type { PermissionMode } from "@/codex-app-server";
import type { AgentModeId, AgentRun } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import {
  updatePreviewModelPreferences,
  updatePreviewPermissionMode,
} from "@/lib/preview-state";

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
  setConfig,
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
  setConfig: (config: domain.AppConfig) => void;
}) {
  async function rememberModelPreferences(
    model: domain.ModelRef | undefined,
    nextReasoningEffort: string,
    nextServiceTier: string,
  ) {
    const nextConfig = updatePreviewModelPreferences(
      model,
      nextReasoningEffort,
      nextServiceTier,
    );
    setConfig(nextConfig);
    if (hasCodexDesktopBridge() && model) {
      try {
        await window.aivoDesktop.codex.saveModelPreferences({
          model: model.modelId,
          modelProvider: model.providerId,
          reasoningEffort: nextReasoningEffort,
        });
      } catch (error) {
        toast.error(
          error instanceof Error
            ? error.message
            : "保存模型配置失败，请重试。",
        );
      }
    }
    return nextConfig;
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
    updatePreviewPermissionMode(normalized);
  }

  function selectAgentMode(nextMode: AgentModeId) {
    setAgentMode(nextMode);
    const sessionId = activeSessionIdRef.current;
    void sessionId;
  }

  async function cancelActiveSubagentRun() {
    void activeRunningSubagentRun;
    void refreshAgentRuntimeState;
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
