import { useState } from "react";

import type { domain } from "../../../bridge/go/models";
import { ProviderConnectionDialogs } from "@/features/setup/provider-connection-dialogs";
import type { CatalogState, ProviderInfo } from "@/lib/provider-catalog";
import type { ModelOption } from "@/features/projects/project-model-options";
import { useProviderConnectState } from "@/features/projects/project-provider-connect-state";
import { ProviderPickerDialog } from "@/features/projects/project-provider-connect-picker";
import {
  getProviderCatalog,
  getProviderCatalogForProject,
  refreshProviderEcosystemCatalog,
} from "@/services/aivo";

export function ProviderConnectDialog({
  catalogProviders,
  onConnected,
  onOpenChange,
  open,
  projectPath,
  setCatalog,
  setConfig,
  setError,
}: {
  catalogProviders: ProviderInfo[];
  onConnected: (option: ModelOption | null) => Promise<void>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  projectPath: string;
  setCatalog: (catalog: CatalogState) => void;
  setConfig: (config: domain.AppConfig) => void;
  setError: (error: string) => void;
}) {
  const [refreshing, setRefreshing] = useState(false);
  const [refreshMessage, setRefreshMessage] = useState("");
  const providerConnectState = useProviderConnectState({
    catalogProviders,
    onConnected,
    onOpenChange,
    setCatalog,
    setConfig,
    setError,
  });

  const refreshCatalog = async () => {
    setRefreshing(true);
    setRefreshMessage("");
    try {
      const result = await refreshProviderEcosystemCatalog();
      const nextCatalog = projectPath
        ? await getProviderCatalogForProject(projectPath)
        : await getProviderCatalog();
      setCatalog(nextCatalog);
      setRefreshMessage(
        `已刷新 ${result.providerCount} 个 Provider、${result.modelCount} 个模型`,
      );
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      setRefreshMessage(`刷新失败：${message}`);
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <>
      <ProviderPickerDialog
        filteredProviders={providerConnectState.filteredProviders}
        onCatalogRefresh={() => void refreshCatalog()}
        onOpenChange={providerConnectState.handlePickerOpenChange}
        onProviderSelect={providerConnectState.selectProvider}
        onQueryChange={providerConnectState.setQuery}
        open={open}
        query={providerConnectState.query}
        refreshMessage={refreshMessage}
        refreshing={refreshing}
      />
      <ProviderConnectionDialogs
        activeProvider={providerConnectState.selectedProvider}
        apiKey={providerConnectState.apiKey}
        authMode={providerConnectState.authMode}
        authSuccessMessage={providerConnectState.authSuccessMessage}
        callbackInput={providerConnectState.callbackInput}
        customProviderForm={providerConnectState.customProviderForm}
        error={providerConnectState.localError}
        models={providerConnectState.activeProviderModels}
        oauthReady={providerConnectState.oauthReady}
        oauthStarted={providerConnectState.oauthStarted}
        oauthStartResult={providerConnectState.oauthStartResult}
        oauthStatus={providerConnectState.oauthStatus}
        onApiKeyChange={providerConnectState.setApiKey}
        onCallbackInputChange={providerConnectState.setCallbackInput}
        onClose={providerConnectState.closeProviderDetails}
        onCustomProviderFormChange={providerConnectState.setCustomProviderForm}
        onProviderDialogStepChange={providerConnectState.setProviderDialogStep}
        onResetAuthMode={providerConnectState.resetAuthMode}
        onSelectOpenAIAuthMode={providerConnectState.selectOpenAIAuthMode}
        onSelectedModelIdChange={providerConnectState.setSelectedModelId}
        onSubmit={providerConnectState.submitProvider}
        providerDialogStep={providerConnectState.providerDialogStep}
        saving={providerConnectState.submitting}
        selectedModelId={providerConnectState.activeProviderModelValue}
        showModelSelect={providerConnectState.showModelSelect}
        submitDisabled={providerConnectState.submitDisabled}
      />
    </>
  );
}
