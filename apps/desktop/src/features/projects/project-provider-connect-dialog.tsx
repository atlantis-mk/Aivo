import type { domain } from "../../../bridge/go/models";
import { ProviderConnectionDialogs } from "@/features/setup/provider-connection-dialogs";
import type { CatalogState, ProviderInfo } from "@/lib/provider-catalog";
import type { ModelOption } from "@/features/projects/project-model-options";
import { useProviderConnectState } from "@/features/projects/project-provider-connect-state";
import { ProviderPickerDialog } from "@/features/projects/project-provider-connect-picker";

export function ProviderConnectDialog({
  catalogProviders,
  onConnected,
  onOpenChange,
  open,
  setCatalog,
  setConfig,
  setError,
}: {
  catalogProviders: ProviderInfo[];
  onConnected: (option: ModelOption | null) => Promise<void>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  setCatalog: (catalog: CatalogState) => void;
  setConfig: (config: domain.AppConfig) => void;
  setError: (error: string) => void;
}) {
  const providerConnectState = useProviderConnectState({
    catalogProviders,
    onConnected,
    onOpenChange,
    setCatalog,
    setConfig,
    setError,
  });

  return (
    <>
      <ProviderPickerDialog
        filteredProviders={providerConnectState.filteredProviders}
        onOpenChange={providerConnectState.handlePickerOpenChange}
        onProviderSelect={providerConnectState.selectProvider}
        onQueryChange={providerConnectState.setQuery}
        open={open}
        query={providerConnectState.query}
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
