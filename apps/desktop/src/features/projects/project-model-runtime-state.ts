import { useEffect, useState } from "react";

import { useProjectModelSelection } from "@/features/projects/project-workspace-derived-state";
import {
  normalizePermissionMode,
  normalizeReasoningEffort,
  normalizeServiceTier,
} from "@/features/projects/project-model-options";
import { hasAppBridge } from "@/lib/app-config";
import type { CatalogState } from "@/lib/provider-catalog";
import { getPermissionMode, type PermissionMode } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

type AppConfigWithPermissionPreference = domain.AppConfig & {
  defaultPermissionMode?: PermissionMode;
};

export function useProjectModelRuntimeState({
  activeSessionId,
  catalog,
  config,
}: {
  activeSessionId: string;
  catalog: CatalogState | null;
  config: domain.AppConfig | null;
}) {
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");
  const [reasoningEffort, setReasoningEffort] = useState("medium");
  const [serviceTier, setServiceTier] = useState("default");
  const [permissionMode, setLocalPermissionMode] =
    useState<PermissionMode>(() =>
      normalizePermissionMode(
        (config as AppConfigWithPermissionPreference | null)?.defaultPermissionMode,
      ),
    );

  const modelSelection = useProjectModelSelection({
    catalog,
    config,
    selectedModelId,
    selectedProviderId,
  });

  useEffect(() => {
    if (activeSessionId) return;
    setLocalPermissionMode(
      normalizePermissionMode(
        (config as AppConfigWithPermissionPreference | null)?.defaultPermissionMode,
      ),
    );
  }, [activeSessionId, config]);

  useEffect(() => {
    if (!hasAppBridge() || !activeSessionId) return;
    let cancelled = false;
    void getPermissionMode(activeSessionId)
      .then((state) => {
        if (!cancelled) {
          setLocalPermissionMode(normalizePermissionMode(state?.mode));
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [activeSessionId]);

  useEffect(() => {
    setReasoningEffort(normalizeReasoningEffort(config?.reasoningEffort));
  }, [config?.reasoningEffort]);

  useEffect(() => {
    setServiceTier(normalizeServiceTier(config?.serviceTier));
  }, [config?.serviceTier]);

  useEffect(() => {
    setSelectedModelId((currentModelId) => {
      if (
        currentModelId &&
        modelSelection.modelOptions.some((model) => model.id === currentModelId)
      ) {
        return currentModelId;
      }
      return modelSelection.defaultModelId;
    });
  }, [
    modelSelection.defaultModelId,
    modelSelection.modelOptions,
    modelSelection.modelOptionsKey,
  ]);

  return {
    ...modelSelection,
    permissionMode,
    reasoningEffort,
    serviceTier,
    setLocalPermissionMode,
    setReasoningEffort,
    setSelectedModelId,
    setSelectedProviderId,
    setServiceTier,
  };
}
