import { useEffect, useState } from "react";
import { Navigate } from "@tanstack/react-router";

import {
  hasAppBridge,
  hasCodexDesktopBridge,
  useAppConfig,
} from "@/lib/app-config";
import type {
  CatalogState,
  ProviderAccountInfo,
  ProviderConnectInput,
} from "@/lib/provider-catalog";
import {
  AuxiliaryModelSelect,
  ConnectedAccountDetailsDialog,
  ConnectedAccountsBar,
} from "@/features/setup/setup-connected-accounts";
import { SetupLoadingSkeleton } from "@/features/setup/setup-loading-skeleton";
import { SetupNameStep } from "@/features/setup/setup-name-step";
import { hasCompletedInitialization } from "@/features/setup/setup-routing";
import { ProviderConnectionDialogs } from "@/features/setup/provider-connection-dialogs";
import type {
  CustomProviderForm,
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  OtherProviderPickerDialog,
  ProviderChoiceGrid,
  WelcomeStep,
} from "@/features/setup/setup-step-components";
import { useSetupProviderActions } from "@/features/setup/setup-provider-actions";
import {
  type ProviderValidationResult,
  useSetupProviderStepState,
} from "@/features/setup/setup-provider-step-state";
import { type AppConfigWithAuxiliary } from "@/features/setup/setup-provider-models";
import { SetupStepNavigation } from "@/features/setup/setup-step-navigation";
import { SetupWorkspaceStep } from "@/features/setup/setup-workspace-step";
import { resolveSetupWorkspacePath } from "@/features/setup/setup-workspace-path";
import { completePreviewInitialization } from "@/lib/preview-state";
import {
  appNameFromConfig,
  canSubmitAppName,
  limitAppNameInput,
} from "@/lib/app-identity";
import {
  completeInitialization,
  selectProjectDirectory,
} from "@/services/aivo";

type SetupStep = "welcome" | "name" | "provider" | "workspace" | "complete";

export function SetupScreen() {
  const { catalog, config, loading, setCatalog, setConfig, setError, error } =
    useAppConfig();
  const [saving, setSaving] = useState(false);
  const [step, setStep] = useState<SetupStep>("welcome");
  const [providerValidated, setProviderValidated] = useState(false);
  const [appName, setAppName] = useState("Aivo");
  const [initialWorkspacePath, setInitialWorkspacePath] = useState("");

  useEffect(() => {
    const workspaceConfig = config as AppConfigWithAuxiliary | null;
    setAppName((current) =>
      current === "Aivo" ? appNameFromConfig(workspaceConfig) : current,
    );
    const savedPath = workspaceConfig?.initialWorkspacePath?.trim() ?? "";
    const suggestedPath = resolveSetupWorkspacePath(workspaceConfig);
    if (suggestedPath) {
      setInitialWorkspacePath((current) =>
        !current || current === savedPath ? suggestedPath : current,
      );
    }
  }, [config]);

  const {
    completeProviderDialog,
    refreshProviderCatalog,
    removeProviderAccount,
    saveAuxiliaryModel,
    validateProvider,
  } = useSetupProviderActions({
    catalog,
    config,
    setCatalog,
    setConfig,
    setError,
    setProviderValidated,
    setSaving,
  });

  if (loading) return <SetupLoadingSkeleton />;

  if (hasCompletedInitialization(config)) {
    return <Navigate to="/projects/chat" replace />;
  }

  if (step === "complete") {
    return <Navigate to="/projects/chat" replace />;
  }

  async function chooseInitialWorkspace() {
    setError("");
    try {
      const selected = hasCodexDesktopBridge()
        ? await window.aivoDesktop.workspace.choose()
        : await selectProjectDirectory();
      if (selected) setInitialWorkspacePath(selected);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function finishInitialization() {
    if (!initialWorkspacePath || !canSubmitAppName(appName)) return;
    setSaving(true);
    setError("");
    try {
      const nextConfig = hasAppBridge()
        ? await completeInitialization({
            appName: appName.trim(),
            initialWorkspacePath,
          })
        : completePreviewInitialization(initialWorkspacePath, appName.trim());
      setConfig(nextConfig);
      setStep("complete");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  function navigateToStep(nextStep: SetupStep) {
    setError("");
    setStep(nextStep);
  }

  return (
    <main className="min-h-dvh overflow-x-hidden bg-background text-foreground">
      <div className="window-title-drag-region" />
      {step === "welcome" ? (
        <WelcomeStep onNext={() => navigateToStep("name")} />
      ) : step === "name" ? (
        <SetupNameStep
          name={appName}
          onBack={() => navigateToStep("welcome")}
          onChange={(value) => setAppName(limitAppNameInput(value))}
          onNext={() => navigateToStep("provider")}
        />
      ) : step === "provider" ? (
        <ProviderStep
          config={config as AppConfigWithAuxiliary | null}
          error={error}
          onBack={() => navigateToStep("name")}
          onContinue={completeProviderDialog}
          onNextPage={() => navigateToStep("workspace")}
          onResetValidation={() => setProviderValidated(false)}
          onValidate={validateProvider}
          catalog={catalog}
          connectedAccounts={
            catalog?.providers.flatMap((provider) => provider.accounts ?? []) ??
            []
          }
          onRemoveAccount={removeProviderAccount}
          onSaveAuxiliaryModel={saveAuxiliaryModel}
          onRefreshModels={refreshProviderCatalog}
          providerValidated={providerValidated}
          saving={saving}
        />
      ) : step === "workspace" ? (
        <SetupWorkspaceStep
          error={error}
          onBack={() => navigateToStep("provider")}
          onChoose={chooseInitialWorkspace}
          onComplete={finishInitialization}
          path={initialWorkspacePath}
          saving={saving}
        />
      ) : null}
    </main>
  );
}

function ProviderStep({
  catalog,
  config,
  error,
  onBack,
  onContinue,
  onNextPage,
  onResetValidation,
  onRefreshModels,
  onValidate,
  connectedAccounts,
  onRemoveAccount,
  onSaveAuxiliaryModel,
  providerValidated,
  saving,
}: {
  catalog: CatalogState | null;
  config: AppConfigWithAuxiliary | null;
  error: string;
  onBack: () => void;
  onContinue: (
    provider: ProviderChoice,
    authMode: ProviderAuthMode,
    apiKey?: string,
    customProvider?: CustomProviderForm,
    selectedModelId?: string,
  ) => Promise<boolean>;
  onNextPage: () => void;
  onResetValidation: () => void;
  onRefreshModels: (
    input: ProviderConnectInput,
  ) => Promise<CatalogState | null>;
  onValidate: (
    provider: ProviderChoice,
    authMode: ProviderAuthMode,
    callbackInput?: string,
    apiKey?: string,
  ) => Promise<ProviderValidationResult>;
  connectedAccounts: ProviderAccountInfo[];
  onRemoveAccount: (accountId: string) => Promise<void>;
  onSaveAuxiliaryModel: (
    providerId: string,
    modelId: string,
  ) => Promise<boolean>;
  providerValidated: boolean;
  saving: boolean;
}) {
  const {
    activeProvider,
    activeProviderModels,
    activeProviderModelValue,
    apiKey,
    authMode,
    authSuccessMessage,
    callbackInput,
    closeProvider,
    customProviderForm,
    oauthReady,
    oauthStarted,
    oauthStartResult,
    oauthStatus,
    openProvider,
    otherProviderPickerOpen,
    otherProviderSearch,
    providerDialogStep,
    resetAuthMode,
    selectOpenAIAuthMode,
    selectOtherProvider,
    setApiKey,
    setCallbackInput,
    setCustomProviderForm,
    setOtherProviderPickerOpen,
    setOtherProviderSearch,
    setProviderDialogStep,
    setSelectedModelId,
    setSettingsAccount,
    settingsAccount,
    showModelSelect,
    submitActiveProvider,
    submitDisabled,
  } = useSetupProviderStepState({
    catalog,
    onContinue,
    onRefreshModels,
    onResetValidation,
    onValidate,
    providerValidated,
    saving,
  });

  return (
    <section className="relative flex min-h-dvh flex-col bg-background">
      <div className="flex flex-1 items-center justify-center px-aivo-4 py-aivo-8 sm:px-aivo-8">
        <div className="flex w-full max-w-[800px] flex-col items-center text-center">
          <h1 className="aivo-type-large-title font-bold tracking-tight text-foreground">
            连接你的执行能力
          </h1>
          <p className="aivo-type-title-3 mt-aivo-3 text-muted-foreground">
            选择一个服务完成连接，之后也可以随时添加
          </p>

          <div className="mt-aivo-8 flex w-full flex-col items-center gap-aivo-4">
            <h2 className="aivo-type-headline font-semibold text-foreground">
              选择服务
            </h2>
            <ProviderChoiceGrid
              activeProviderId={activeProvider?.id}
              onProviderClick={openProvider}
            />
          </div>

          <ConnectedAccountsBar
            accounts={connectedAccounts}
            onAccountClick={setSettingsAccount}
            onRemoveAccount={onRemoveAccount}
          />
          <AuxiliaryModelSelect
            accounts={connectedAccounts}
            catalog={catalog}
            config={config}
            onSave={onSaveAuxiliaryModel}
          />
        </div>
      </div>

      {!activeProvider ? (
        <SetupStepNavigation
          currentStep={3}
          helperText="之后可在设置中继续添加或管理服务"
          onBack={onBack}
          onPrimary={onNextPage}
          primaryContent="继续"
          totalSteps={4}
        />
      ) : null}

      <OtherProviderPickerDialog
        onOpenChange={setOtherProviderPickerOpen}
        onSelect={selectOtherProvider}
        open={otherProviderPickerOpen}
        search={otherProviderSearch}
        onSearchChange={setOtherProviderSearch}
      />

      <ProviderConnectionDialogs
        activeProvider={activeProvider}
        apiKey={apiKey}
        authMode={authMode}
        authSuccessMessage={authSuccessMessage}
        callbackInput={callbackInput}
        customProviderForm={customProviderForm}
        error={error}
        models={activeProviderModels}
        oauthReady={oauthReady}
        oauthStarted={oauthStarted}
        oauthStartResult={oauthStartResult}
        oauthStatus={oauthStatus}
        onApiKeyChange={setApiKey}
        onCallbackInputChange={setCallbackInput}
        onClose={closeProvider}
        onCustomProviderFormChange={setCustomProviderForm}
        onProviderDialogStepChange={setProviderDialogStep}
        onResetAuthMode={resetAuthMode}
        onSelectOpenAIAuthMode={selectOpenAIAuthMode}
        onSelectedModelIdChange={setSelectedModelId}
        onSubmit={submitActiveProvider}
        providerDialogStep={providerDialogStep}
        saving={saving}
        selectedModelId={activeProviderModelValue}
        showModelSelect={showModelSelect}
        submitDisabled={submitDisabled}
      />
      <ConnectedAccountDetailsDialog
        account={settingsAccount}
        onClose={() => setSettingsAccount(null)}
      />
    </section>
  );
}
