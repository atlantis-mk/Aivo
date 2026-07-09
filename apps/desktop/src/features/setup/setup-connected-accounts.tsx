import { useEffect, useState } from "react";
import { X } from "lucide-react";

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
  currentAuxiliaryModelForProvider,
  currentDefaultModelForProvider,
  modelOptionsForConnectedProvider,
  type AppConfigWithAuxiliary,
} from "@/features/setup/setup-provider-models";
import type {
  CatalogState,
  ModelInfo,
  ProviderAccountInfo,
} from "@/lib/provider-catalog";

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
    <div className="flex w-full max-w-[880px] flex-wrap items-center justify-center gap-2">
      {accounts.map((account) => (
        <div
          key={account.id}
          className="inline-flex max-w-full items-center gap-1.5"
        >
          <Button
            className="max-w-full rounded-full px-3"
            onClick={() => onAccountClick(account)}
            size="sm"
            type="button"
            variant="secondary"
          >
            <span className="min-w-0 truncate">
              {accountTypeLabel(account)} {accountDisplayName(account)} 已连接
            </span>
          </Button>
          <Button
            aria-label={`删除 ${accountDisplayName(account)}`}
            className="rounded-full"
            onClick={(event) => {
              event.stopPropagation();
              void onRemoveAccount(account.id);
            }}
            size="icon"
            type="button"
            variant="ghost"
          >
            <X />
          </Button>
        </div>
      ))}
    </div>
  );
}

export function ConnectedAccountModelDialog({
  account,
  catalog,
  config,
  onClose,
  onSave,
}: {
  account: ProviderAccountInfo | null;
  catalog: CatalogState | null;
  config: AppConfigWithAuxiliary | null;
  onClose: () => void;
  onSave: (
    providerId: string,
    modelId: string,
    auxiliaryModelId: string,
  ) => Promise<void>;
}) {
  const provider = account
    ? catalog?.providers.find((item) => item.id === account.providerId)
    : undefined;
  const modelOptions = provider
    ? modelOptionsForConnectedProvider(provider, catalog)
    : [];
  const defaultModelId = provider
    ? currentDefaultModelForProvider(config, provider, modelOptions)
    : "";
  const defaultAuxiliaryModelId = provider
    ? currentAuxiliaryModelForProvider(
        config,
        provider,
        modelOptions,
        defaultModelId,
      )
    : "";
  const [modelId, setModelId] = useState(defaultModelId);
  const [auxiliaryModelId, setAuxiliaryModelId] = useState(
    defaultAuxiliaryModelId,
  );
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setModelId(defaultModelId);
    setAuxiliaryModelId(defaultAuxiliaryModelId);
  }, [defaultModelId, defaultAuxiliaryModelId, account?.id]);

  async function handleSave() {
    if (!account || !modelId || !auxiliaryModelId) return;
    setSaving(true);
    try {
      await onSave(account.providerId, modelId, auxiliaryModelId);
      onClose();
    } finally {
      setSaving(false);
    }
  }

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
            <DialogTitle className="min-w-0 truncate">
              {provider?.name || account?.providerId || "Provider"} 模型设置
            </DialogTitle>
            <DialogClose asChild>
              <Button aria-label="关闭" size="icon" type="button" variant="ghost">
                <X />
              </Button>
            </DialogClose>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <ConnectedModelSelect
              label="主模型"
              models={modelOptions}
              onValueChange={setModelId}
              value={modelId}
            />
            <ConnectedModelSelect
              label="辅助模型"
              models={modelOptions}
              onValueChange={setAuxiliaryModelId}
              value={auxiliaryModelId}
            />
          </div>

          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button type="button" variant="secondary">
                取消
              </Button>
            </DialogClose>
            <Button
              disabled={!modelId || !auxiliaryModelId || saving}
              onClick={handleSave}
              type="button"
            >
              {saving ? "保存中" : "保存"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ConnectedModelSelect({
  label,
  models,
  onValueChange,
  value,
}: {
  label: string;
  models: ModelInfo[];
  onValueChange: (value: string) => void;
  value: string;
}) {
  return (
    <div className="flex flex-col gap-1.5 text-left">
      <label className="text-sm">{label}</label>
      <Select onValueChange={onValueChange} value={value}>
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {models.map((model) => (
            <SelectItem key={model.id} value={model.id}>
              {model.name || model.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
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

function accountDisplayName(account: ProviderAccountInfo) {
  const displayName = account.displayName?.trim();
  const accountId = account.accountId?.trim();
  if (displayName) return displayName;
  return displayName || accountId || "默认账号";
}
