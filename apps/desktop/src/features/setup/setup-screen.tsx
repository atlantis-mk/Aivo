import { useState } from "react";
import { Navigate } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";
import { useAppConfig } from "@/lib/app-config";
import type { CatalogState, ProviderAccountInfo, ProviderConnectInput } from "@/lib/provider-catalog";
import {
 ConnectedAccountModelDialog,
 ConnectedAccountsBar,
} from "@/features/setup/setup-connected-accounts";
import { SetupLoadingSkeleton } from "@/features/setup/setup-loading-skeleton";
import {
 ProviderConnectionDialogs,
} from "@/features/setup/provider-connection-dialogs";
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

type SetupStep = "welcome" | "provider" | "project";

export function SetupScreen() {
 const { catalog, config, loading, setCatalog, setConfig, setError, error } = useAppConfig();
 const [saving, setSaving] = useState(false);
 const [step, setStep] = useState<SetupStep>("welcome");
 const [providerValidated, setProviderValidated] = useState(false);

 const {
 completeProviderDialog,
 refreshProviderCatalog,
 removeProviderAccount,
 saveConnectedAccountModels,
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

 if (step === "project") {
 return <Navigate to="/projects/chat" replace />;
 }

 return (
 <main className="min-h-dvh overflow-x-hidden bg-background text-foreground">
 <div className="window-title-drag-region" />
 {step === "welcome" ? (
 <WelcomeStep onNext={() => setStep("provider")} />
 ) : step === "provider" ? (
 <ProviderStep
 config={config as AppConfigWithAuxiliary | null}
 error={error}
 onContinue={completeProviderDialog}
 onNextPage={() => setStep("project")}
 onResetValidation={() => setProviderValidated(false)}
 onValidate={validateProvider}
 catalog={catalog}
 connectedAccounts={catalog?.providers.flatMap((provider) => provider.accounts ?? []) ?? []}
 onRemoveAccount={removeProviderAccount}
 onSaveAccountModels={saveConnectedAccountModels}
 onRefreshModels={refreshProviderCatalog}
 providerValidated={providerValidated}
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
 onContinue,
 onNextPage,
 onResetValidation,
 onRefreshModels,
 onValidate,
 connectedAccounts,
 onRemoveAccount,
 onSaveAccountModels,
 providerValidated,
 saving,
}: {
 catalog: CatalogState | null;
 config: AppConfigWithAuxiliary | null;
 error: string;
 onContinue: (
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 apiKey?: string,
 customProvider?: CustomProviderForm,
 selectedModelId?: string,
 selectedAuxiliaryModelId?: string,
 ) => Promise<boolean>;
 onNextPage: () => void;
 onResetValidation: () => void;
 onRefreshModels: (input: ProviderConnectInput) => Promise<CatalogState | null>;
 onValidate: (
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 callbackInput?: string,
 apiKey?: string,
 ) => Promise<ProviderValidationResult>;
 connectedAccounts: ProviderAccountInfo[];
 onRemoveAccount: (accountId: string) => Promise<void>;
 onSaveAccountModels: (providerId: string, modelId: string, auxiliaryModelId: string) => Promise<void>;
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
 selectedAuxiliaryModelId,
 selectOpenAIAuthMode,
 selectOtherProvider,
 setApiKey,
 setCallbackInput,
 setCustomProviderForm,
 setOtherProviderPickerOpen,
 setOtherProviderSearch,
 setProviderDialogStep,
 setSelectedAuxiliaryModelId,
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
 <section className="relative flex min-h-dvh justify-center bg-background px-5 py-20 sm:pt-28">
 <div className="flex w-full max-w-[1032px] flex-col items-center gap-6 text-center">
 <h1 className="max-w-[680px] text-3xl font-extrabold leading-9 tracking-normal text-foreground sm:text-4xl sm:leading-10">
 连接你的执行能力
 </h1>

 <ProviderChoiceGrid
 activeProviderId={activeProvider?.id}
 onProviderClick={openProvider}
 />

 <ConnectedAccountsBar
 accounts={connectedAccounts}
 onAccountClick={setSettingsAccount}
 onRemoveAccount={onRemoveAccount}
 />
 {!activeProvider ? (
 <Button
 className="h-12 rounded-full px-8 text-base"
 onClick={onNextPage}
 size="lg"
 >
 继续
 </Button>
 ) : null}
 </div>

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
 onSelectedAuxiliaryModelIdChange={setSelectedAuxiliaryModelId}
 onSubmit={submitActiveProvider}
 providerDialogStep={providerDialogStep}
 saving={saving}
 selectedModelId={activeProviderModelValue}
 selectedAuxiliaryModelId={selectedAuxiliaryModelId}
 showModelSelect={showModelSelect}
 submitDisabled={submitDisabled}
 />
 <ConnectedAccountModelDialog
 account={settingsAccount}
 catalog={catalog}
 config={config}
 onClose={() => setSettingsAccount(null)}
 onSave={onSaveAccountModels}
 />
 </section>
 );
}
