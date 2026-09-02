import { useMemo, useState } from "react";
import {
  Add01Icon,
  Alert02Icon,
  Delete02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { RefreshCw } from "lucide-react";
import { toast } from "sonner";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ProviderIcon } from "@/features/providers/provider-icon";
import {
  configuredProviderRefreshInput,
  configuredProviders,
  providerCanRefreshModels,
  providerConnectionMethodLabel,
  providerModelLabel,
  providerRefreshUnavailableMessage,
  providerReadinessLabel,
} from "@/features/providers/provider-settings-model";
import type { ProviderChoice } from "@/features/providers/provider-types";
import {
  normalizeWebSearchMode,
  webSearchModeLabel,
  webSearchModeOptions,
  type WebSearchMode,
} from "@/features/projects/project-model-options";
import { ProviderConnectionDialogs } from "@/features/setup/provider-connection-dialogs";
import { useSetupProviderActions } from "@/features/setup/setup-provider-actions";
import {
  otherProviderChoices,
  providerChoices,
} from "@/features/setup/setup-provider-options";
import { useSetupProviderStepState } from "@/features/setup/setup-provider-step-state";
import {
  OtherProviderPickerDialog,
  ProviderChoiceGrid,
} from "@/features/setup/setup-step-components";
import { hasAppBridge, useAppConfig } from "@/lib/app-config";
import { deletePreviewProvider } from "@/lib/preview-state";
import type { ProviderInfo } from "@/lib/provider-catalog";
import {
  deleteProvider,
  getAppConfig,
  getProviderCatalog,
  refreshProviderEcosystemCatalog,
  refreshProviderModels,
  updateModelPreferences,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

type AppConfigWithWebSearch = domain.AppConfig & {
  webSearch?: {
    mode?: WebSearchMode;
    route?: string;
  };
};

type WebSearchPreferenceInput = domain.ModelPreferencesInput & {
  webSearch: {
    mode: WebSearchMode;
    route: "auto";
  };
};

export function ProviderSettingsScreen() {
  const {
    catalog,
    config,
    error,
    loading,
    setCatalog,
    setConfig,
    setError,
  } = useAppConfig();
  const [saving, setSaving] = useState(false);
  const [deletingProviderId, setDeletingProviderId] = useState("");
  const [refreshingCatalog, setRefreshingCatalog] = useState(false);
  const [refreshingProviderId, setRefreshingProviderId] = useState("");
  const [savingWebSearch, setSavingWebSearch] = useState(false);
  const [providerValidated, setProviderValidated] = useState(false);
  const webSearchMode = normalizeWebSearchMode(
    (config as AppConfigWithWebSearch | null)?.webSearch?.mode,
  );

  const actions = useSetupProviderActions({
    catalog,
    config,
    setCatalog,
    setConfig,
    setError,
    setProviderValidated,
    setSaving,
  });
  const connection = useSetupProviderStepState({
    catalog,
    onContinue: actions.completeProviderDialog,
    onRefreshModels: actions.refreshProviderCatalog,
    onResetValidation: () => setProviderValidated(false),
    onValidate: actions.validateProvider,
    providerValidated,
    saving,
  });
  const providers = useMemo(
    () =>
      configuredProviders(
        catalog?.providers ?? [],
        catalog?.defaultModel?.providerId,
      ),
    [catalog],
  );

  async function removeProvider(provider: ProviderInfo) {
    if (
      deletingProviderId ||
      refreshingCatalog ||
      refreshingProviderId ||
      savingWebSearch
    ) {
      return;
    }
    setDeletingProviderId(provider.id);
    setError("");
    try {
      if (hasAppBridge()) {
        const nextCatalog = await deleteProvider(provider.id);
        const nextConfig = await getAppConfig();
        setCatalog(nextCatalog);
        setConfig(nextConfig);
      } else {
        const next = deletePreviewProvider(provider.id);
        setCatalog(next.catalog);
        setConfig(next.config);
      }
      toast.success(`已删除 ${provider.name}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setDeletingProviderId("");
    }
  }

  async function refreshEcosystemCatalog() {
    if (
      refreshingCatalog ||
      refreshingProviderId ||
      deletingProviderId ||
      saving ||
      savingWebSearch
    ) {
      return;
    }
    if (!hasAppBridge()) {
      setError("更新模型目录需要连接本地 Aivo Core。");
      return;
    }
    setRefreshingCatalog(true);
    setError("");
    try {
      const result = await refreshProviderEcosystemCatalog();
      const nextCatalog = await getProviderCatalog();
      setCatalog(nextCatalog);
      toast.success(
        `已更新 ${result.providerCount} 个 Provider、${result.modelCount} 个模型`,
      );
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`更新模型目录失败：${message}`);
    } finally {
      setRefreshingCatalog(false);
    }
  }

  async function refreshConfiguredProvider(provider: ProviderInfo) {
    if (
      refreshingProviderId ||
      refreshingCatalog ||
      deletingProviderId ||
      saving ||
      savingWebSearch
    ) {
      return;
    }
    const unavailableMessage = providerRefreshUnavailableMessage(provider);
    if (unavailableMessage) {
      setError(unavailableMessage);
      toast.error(unavailableMessage);
      return;
    }
    if (!hasAppBridge()) {
      const message = "刷新模型需要连接本地 Aivo Core。";
      setError(message);
      toast.error(message);
      return;
    }
    setRefreshingProviderId(provider.id);
    setError("");
    try {
      const nextCatalog = await refreshProviderModels(
        configuredProviderRefreshInput(provider),
      );
      setCatalog(nextCatalog);
      const refreshed = nextCatalog.providers.find(
        (candidate) => candidate.id === provider.id,
      );
      toast.success(
        `已刷新 ${provider.name}，获取 ${refreshed?.models.length ?? 0} 个模型`,
      );
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const failureMessage = `刷新 ${provider.name} 模型失败，已保留原模型列表：${message}`;
      setError(failureMessage);
      toast.error(failureMessage);
    } finally {
      setRefreshingProviderId("");
    }
  }

  async function saveWebSearchMode(nextMode: string) {
    if (savingWebSearch) return;
    const normalized = normalizeWebSearchMode(nextMode);
    const existing = (config as AppConfigWithWebSearch | null)?.webSearch ?? {};
    const webSearch = {
      ...existing,
      mode: normalized,
      route: "auto" as const,
    };
    if (!hasAppBridge()) {
      if (config) {
        setConfig({
          ...(config as AppConfigWithWebSearch),
          webSearch,
        } as unknown as domain.AppConfig);
      }
      return;
    }
    setSavingWebSearch(true);
    setError("");
    try {
      const nextConfig = await updateModelPreferences({
        webSearch: {
          mode: normalized,
          route: "auto",
        },
      } as WebSearchPreferenceInput);
      setConfig(nextConfig);
      toast.success(`Web Search 已设置为${webSearchModeLabel(normalized)}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(`Web Search 模式保存失败：${message}`);
      toast.error("Web Search 模式保存失败");
    } finally {
      setSavingWebSearch(false);
    }
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-aivo-8 px-aivo-4 py-aivo-6 sm:px-aivo-8 sm:py-aivo-8">
        <header className="flex flex-col gap-aivo-2">
          <h1 className="aivo-type-title-1 font-semibold text-foreground">
            模型提供商
          </h1>
          <p className="aivo-type-body max-w-2xl text-muted-foreground">
            查看已经配置的 Provider，或使用与初始化相同的流程连接新的服务。
          </p>
        </header>

        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Provider 操作失败</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        <section
          aria-labelledby="model-runtime-preferences-heading"
          className="flex flex-col gap-aivo-4"
        >
          <div className="flex flex-col gap-aivo-1">
            <h2
              className="aivo-type-title-3 font-semibold text-foreground"
              id="model-runtime-preferences-heading"
            >
              模型运行偏好
            </h2>
            <p className="aivo-type-footnote text-muted-foreground">
              这些默认值会随模型请求一起发送，Provider 或模型不支持时由 Core 跳过。
            </p>
          </div>
          <Card>
            <CardHeader>
              <CardTitle>Web Search</CardTitle>
              <CardDescription>
                默认使用实时搜索，可按需要限制为索引、缓存或关闭。
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-aivo-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-foreground">
                    搜索模式
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {
                      webSearchModeOptions.find(
                        (option) => option.mode === webSearchMode,
                      )?.description
                    }
                  </p>
                </div>
                <Select
                  disabled={loading || savingWebSearch}
                  onValueChange={(value) => void saveWebSearchMode(value)}
                  value={webSearchMode}
                >
                  <SelectTrigger className="h-9 w-full px-3 text-sm sm:w-44">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {webSearchModeOptions.map((option) => (
                      <SelectItem key={option.mode} value={option.mode}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>
        </section>

        <section aria-labelledby="configured-providers-heading" className="flex flex-col gap-aivo-4">
          <div className="flex items-end justify-between gap-aivo-3">
            <div className="flex flex-col gap-aivo-1">
              <h2
                className="aivo-type-title-3 font-semibold text-foreground"
                id="configured-providers-heading"
              >
                已配置
              </h2>
              <p className="aivo-type-footnote text-muted-foreground">
                配置与凭据由本地 Core 管理。
              </p>
            </div>
            {!loading ? (
              <Badge variant="secondary">{providers.length} 个</Badge>
            ) : null}
          </div>

          {loading ? (
            <ProviderSettingsSkeleton />
          ) : providers.length > 0 ? (
            <div className="grid grid-cols-1 gap-aivo-3 lg:grid-cols-2">
              {providers.map((provider) => (
                <ProviderSettingsCard
                  deleting={deletingProviderId === provider.id}
                  disabled={
                    Boolean(deletingProviderId) ||
                    refreshingCatalog ||
                    Boolean(refreshingProviderId) ||
                    saving ||
                    savingWebSearch
                  }
                  key={provider.id}
                  onDelete={() => void removeProvider(provider)}
                  onRefresh={() => void refreshConfiguredProvider(provider)}
                  provider={provider}
                  refreshing={refreshingProviderId === provider.id}
                />
              ))}
            </div>
          ) : (
            <Empty className="border">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <HugeiconsIcon icon={Add01Icon} strokeWidth={2} />
                </EmptyMedia>
                <EmptyTitle>还没有配置 Provider</EmptyTitle>
                <EmptyDescription>
                  从下方选择一个服务并完成连接，它会立即出现在这里。
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent />
            </Empty>
          )}
        </section>

        <section aria-labelledby="add-provider-heading" className="flex flex-col gap-aivo-4 pb-aivo-6">
          <div className="flex flex-col gap-aivo-3 sm:flex-row sm:items-end sm:justify-between">
            <div className="flex flex-col gap-aivo-1">
              <h2
                className="aivo-type-title-3 font-semibold text-foreground"
                id="add-provider-heading"
              >
                添加 Provider
              </h2>
              <p className="aivo-type-footnote text-muted-foreground">
                API Key 只在连接请求中临时使用，不会写入渲染器存储。
              </p>
            </div>
            <Button
              className="self-start sm:self-auto"
              disabled={
                loading ||
                refreshingCatalog ||
                Boolean(refreshingProviderId) ||
                Boolean(deletingProviderId) ||
                saving ||
                savingWebSearch
              }
              onClick={() => void refreshEcosystemCatalog()}
              variant="outline"
            >
              {refreshingCatalog ? (
                <Spinner data-icon="inline-start" />
              ) : (
                <RefreshCw data-icon="inline-start" />
              )}
              更新模型目录
            </Button>
          </div>
          <ProviderChoiceGrid
            activeProviderId={connection.activeProvider?.id}
            fluid
            onProviderClick={connection.openProvider}
          />
        </section>
      </div>

      <OtherProviderPickerDialog
        onOpenChange={connection.setOtherProviderPickerOpen}
        onSearchChange={connection.setOtherProviderSearch}
        onSelect={connection.selectOtherProvider}
        open={connection.otherProviderPickerOpen}
        search={connection.otherProviderSearch}
      />
      <ProviderConnectionDialogs
        activeProvider={connection.activeProvider}
        apiKey={connection.apiKey}
        authMode={connection.authMode}
        authSuccessMessage={connection.authSuccessMessage}
        callbackInput={connection.callbackInput}
        customProviderForm={connection.customProviderForm}
        error={error}
        models={connection.activeProviderModels}
        oauthReady={connection.oauthReady}
        oauthStarted={connection.oauthStarted}
        oauthStartResult={connection.oauthStartResult}
        oauthStatus={connection.oauthStatus}
        onApiKeyChange={connection.setApiKey}
        onCallbackInputChange={connection.setCallbackInput}
        onClose={connection.closeProvider}
        onCustomProviderFormChange={connection.setCustomProviderForm}
        onProviderDialogStepChange={connection.setProviderDialogStep}
        onResetAuthMode={connection.resetAuthMode}
        onSelectOpenAIAuthMode={connection.selectOpenAIAuthMode}
        onSelectedModelIdChange={connection.setSelectedModelId}
        onSubmit={connection.submitActiveProvider}
        providerDialogStep={connection.providerDialogStep}
        saving={saving}
        selectedModelId={connection.activeProviderModelValue}
        showModelSelect={connection.showModelSelect}
        submitDisabled={connection.submitDisabled}
      />
    </div>
  );
}

function ProviderSettingsCard({
  deleting,
  disabled,
  onDelete,
  onRefresh,
  provider,
  refreshing,
}: {
  deleting: boolean;
  disabled: boolean;
  onDelete: () => void;
  onRefresh: () => void;
  provider: ProviderInfo;
  refreshing: boolean;
}) {
  const accountCount = provider.accounts?.length ?? 0;
  const refreshable = providerCanRefreshModels(provider);
  return (
    <Card>
      <CardHeader>
        <div className="flex min-w-0 items-center gap-aivo-3">
          <ProviderIcon provider={providerChoiceFor(provider)} size="sm" />
          <div className="min-w-0">
            <CardTitle className="truncate">{provider.name}</CardTitle>
            <CardDescription className="truncate">{provider.id}</CardDescription>
          </div>
        </div>
        <CardAction>
          <Badge variant={provider.readiness?.ready ? "default" : "outline"}>
            {providerReadinessLabel(provider)}
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-aivo-4 gap-y-aivo-2">
          <dt className="text-muted-foreground">默认模型</dt>
          <dd className="max-w-52 truncate text-right text-foreground">
            {providerModelLabel(provider)}
          </dd>
          <dt className="text-muted-foreground">连接方式</dt>
          <dd className="max-w-52 truncate text-right text-foreground">
            {providerConnectionMethodLabel(provider)}
          </dd>
          <dt className="text-muted-foreground">账号</dt>
          <dd className="text-right text-foreground">
            {accountCount > 0 ? `${accountCount} 个` : "默认连接"}
          </dd>
        </dl>
      </CardContent>
      <CardFooter className="justify-end gap-aivo-2 border-t">
        <Button
          disabled={disabled}
          onClick={onRefresh}
          title={refreshable ? undefined : "点击查看该 Provider 的模型更新方式"}
          variant="ghost"
        >
          {refreshing ? (
            <Spinner data-icon="inline-start" />
          ) : (
            <RefreshCw data-icon="inline-start" />
          )}
          刷新模型
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button disabled={disabled} variant="ghost">
              {deleting ? (
                <Spinner data-icon="inline-start" />
              ) : (
                <HugeiconsIcon
                  data-icon="inline-start"
                  icon={Delete02Icon}
                  strokeWidth={2}
                />
              )}
              删除
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogMedia>
                <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />
              </AlertDialogMedia>
              <AlertDialogTitle>删除“{provider.name}”？</AlertDialogTitle>
              <AlertDialogDescription>
                这会移除该 Provider 的配置、认证信息、模型缓存和健康状态。调用记录仍会保留用于本地审计。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction onClick={onDelete} variant="destructive">
                删除 Provider
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardFooter>
    </Card>
  );
}

function ProviderSettingsSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-aivo-3 lg:grid-cols-2">
      {[0, 1].map((index) => (
        <Card aria-hidden="true" key={index}>
          <CardHeader>
            <Skeleton className="h-5 w-36" />
            <CardDescription>
              <Skeleton className="h-4 w-24" />
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-aivo-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-4/5" />
          </CardContent>
          <CardFooter className="justify-end border-t">
            <Skeleton className="h-8 w-24" />
          </CardFooter>
        </Card>
      ))}
    </div>
  );
}

function providerChoiceFor(provider: ProviderInfo): ProviderChoice {
  return (
    [...providerChoices, ...otherProviderChoices].find(
      (candidate) => candidate.id === provider.id,
    ) ?? {
      id: provider.id,
      name: provider.name,
      custom: provider.custom,
    }
  );
}
