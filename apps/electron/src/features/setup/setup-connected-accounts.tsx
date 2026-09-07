import { useEffect, useState } from "react";
import {
  Cancel01Icon,
  CheckmarkCircle01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { providerNameForPrompt } from "@/features/providers/provider-defaults";
import {
  modelOptionsForConnectedProvider,
  type AppConfigWithAuxiliary,
} from "@/features/setup/setup-provider-models";
import type { CatalogState, ProviderAccountInfo } from "@/lib/provider-catalog";

export function ConnectedAccountsBar({
  accounts,
  onAccountClick,
  onRemoveAccount,
}: {
  accounts: ProviderAccountInfo[];
  onAccountClick: (account: ProviderAccountInfo) => void;
  onRemoveAccount: (accountId: string) => Promise<void>;
}) {
  if (accounts.length === 0) {
    return null;
  }

  return (
    <section className="mt-6 flex w-full max-w-[640px] flex-col gap-3">
      <div className="flex items-baseline justify-between gap-3 text-left">
        <h2 className="font-semibold text-foreground">已连接</h2>
        <span className="text-muted-foreground">{accounts.length} 个服务</span>
      </div>
      <ul className="grid w-full grid-cols-1 gap-2 sm:grid-cols-2">
        {accounts.map((account) => (
          <li
            className="flex min-w-0 items-center rounded-lg border border-border bg-muted/40 p-1"
            key={account.id}
          >
            <Button
              className="min-w-0 flex-1 justify-start gap-2 px-2 py-2"
              onClick={() => onAccountClick(account)}
              size="sm"
              type="button"
              variant="ghost"
            >
              <HugeiconsIcon
                aria-hidden="true"
                className="size-4 shrink-0"
                icon={CheckmarkCircle01Icon}
                strokeWidth={1.8}
              />
              <span className="min-w-0 truncate text-left">
                {accountTypeLabel(account)} {accountDisplayName(account)}
              </span>
            </Button>
            <Button
              aria-label={`删除 ${accountDisplayName(account)}`}
              onClick={(event) => {
                event.stopPropagation();
                void onRemoveAccount(account.id);
              }}
              size="icon-lg"
              type="button"
              variant="ghost"
            >
              <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
            </Button>
          </li>
        ))}
      </ul>
    </section>
  );
}

export function ConnectedAccountDetailsDialog({
  account,
  onClose,
}: {
  account: ProviderAccountInfo | null;
  onClose: () => void;
}) {
  return (
    <Dialog
      open={Boolean(account)}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="sm:max-w-md" showCloseButton={false}>
        <div className="flex flex-col gap-4">
          <div className="flex min-w-0 items-center justify-between gap-3">
            <DialogTitle className="min-w-0 truncate">连接详情</DialogTitle>
            <DialogClose asChild>
              <Button
                aria-label="关闭"
                size="icon"
                type="button"
                variant="ghost"
              >
                <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
              </Button>
            </DialogClose>
          </div>

          {account ? (
            <dl className="grid gap-3 text-left">
              <AccountDetail
                label="服务"
                value={providerNameForPrompt(account.providerId)}
              />
              <AccountDetail
                label="连接方式"
                value={connectionMethodLabel(account)}
              />
              <AccountDetail label="账号" value={accountDisplayName(account)} />
              <AccountDetail label="状态" value="已连接" />
              <AccountDetail
                label="连接时间"
                value={formatConnectedAt(account.connectedAt)}
              />
            </dl>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function AccountDetail({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-3 border-b border-border pb-3 last:border-b-0 last:pb-0">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 break-words text-foreground">{value}</dd>
    </div>
  );
}

export function AuxiliaryModelSelect({
  accounts,
  catalog,
  config,
  onSave,
}: {
  accounts: ProviderAccountInfo[];
  catalog: CatalogState | null;
  config: AppConfigWithAuxiliary | null;
  onSave: (providerId: string, modelId: string) => Promise<boolean>;
}) {
  const options = auxiliaryModelOptions(accounts, catalog);
  const configuredValue = modelRefValue(
    config?.auxiliaryModel?.providerId,
    config?.auxiliaryModel?.modelId,
  );
  const validConfiguredValue = options.some(
    (option) => option.value === configuredValue,
  )
    ? configuredValue
    : "";
  const [selectedValue, setSelectedValue] = useState(validConfiguredValue);
  const [saving, setSaving] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);

  useEffect(() => {
    if (!saving) setSelectedValue(validConfiguredValue);
  }, [saving, validConfiguredValue]);

  if (accounts.length === 0) return null;

  async function handleValueChange(value: string) {
    const option = options.find((item) => item.value === value);
    if (!option) return;
    setSaveFailed(false);
    setSelectedValue(value);
    setSaving(true);
    const saved = await onSave(option.providerId, option.modelId);
    if (!saved) {
      setSelectedValue(validConfiguredValue);
      setSaveFailed(true);
    }
    setSaving(false);
  }

  return (
    <section className="mt-4 flex w-full max-w-[640px] flex-col gap-3 text-left">
      <div className="flex items-baseline justify-between gap-3">
        <div>
          <h2 className="font-semibold text-foreground">辅助模型</h2>
          <p className="mt-1 text-muted-foreground">
            从已连接服务的模型中选择一个
          </p>
        </div>
        <span aria-live="polite" className="text-muted-foreground">
          {saving ? "保存中" : selectedValue ? "已配置" : "待选择"}
        </span>
      </div>
      <Select
        disabled={options.length === 0 || saving}
        onValueChange={(value) => void handleValueChange(value)}
        value={selectedValue}
      >
        <SelectTrigger
          aria-label="辅助模型"
          className="h-12 w-full bg-background"
        >
          <SelectValue placeholder="选择辅助模型" />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {saveFailed ? (
        <p className="text-destructive" role="alert">
          保存失败，请重试
        </p>
      ) : null}
    </section>
  );
}

function accountTypeLabel(account: ProviderAccountInfo) {
  if (account.providerId === "openai" && account.method === "oauth-browser") {
    return "OpenAI Browser";
  }
  if (account.providerId === "openai" && account.method === "oauth-headless") {
    return "OpenAI Headless";
  }
  if (account.method === "api-key") {
    return `${providerNameForPrompt(account.providerId)} API Key`;
  }
  return providerNameForPrompt(account.providerId);
}

function connectionMethodLabel(account: ProviderAccountInfo) {
  if (account.method === "oauth-browser") return "浏览器 OAuth";
  if (account.method === "oauth-headless") return "设备授权";
  if (account.method === "api-key") return "API Key";
  return account.method || "默认方式";
}

function accountDisplayName(account: ProviderAccountInfo) {
  const displayName = account.displayName?.trim();
  const accountId = account.accountId?.trim();
  if (displayName) return displayName;
  return displayName || accountId || "默认账号";
}

function formatConnectedAt(value?: string) {
  if (!value) return "未知";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function auxiliaryModelOptions(
  accounts: ProviderAccountInfo[],
  catalog: CatalogState | null,
) {
  const connectedProviderIds = new Set(
    accounts.map((account) => account.providerId),
  );
  return (catalog?.providers ?? []).flatMap((provider) => {
    if (!connectedProviderIds.has(provider.id)) return [];
    return modelOptionsForConnectedProvider(provider, catalog).map((model) => ({
      label: `${provider.name || provider.id} · ${model.name || model.id}`,
      modelId: model.id,
      providerId: provider.id,
      value: modelRefValue(provider.id, model.id),
    }));
  });
}

function modelRefValue(providerId?: string, modelId?: string) {
  if (!providerId || !modelId) return "";
  return `${encodeURIComponent(providerId)}:${encodeURIComponent(modelId)}`;
}
