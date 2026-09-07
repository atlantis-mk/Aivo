import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type {
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";
import type { ModelInfo } from "@/lib/provider-catalog";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import { cn } from "@/lib/utils";

const openAIAuthModes: Array<{
  id: ProviderAuthMode;
  label: string;
  description: string;
}> = [
  {
    id: "oauth-browser",
    label: "ChatGPT Pro/Plus (Browser)",
    description: "打开浏览器完成 OpenAI 授权。",
  },
  {
    id: "oauth-headless",
    label: "ChatGPT Pro/Plus (Headless)",
    description: "复制确认码，在授权页完成登录。",
  },
  {
    id: "api-key",
    label: "API Key",
    description: "直接输入 OpenAI API 密钥。",
  },
];

export function OpenAIAuthModeOptions({
  onSelect,
}: {
  onSelect: (mode: ProviderAuthMode) => void;
}) {
  const modes = hasCodexDesktopBridge()
    ? openAIAuthModes.filter((mode) => mode.id === "oauth-browser")
    : openAIAuthModes;

  return (
    <div className="flex  flex-col gap-2">
      {modes.map((mode) => (
        <button
          className="rounded-lg border px-3 py-2 text-left text-sm transition-colors hover:bg-muted"
          key={mode.id}
          onClick={() => onSelect(mode.id)}
          type="button"
        >
            <span className="block ">
              {hasCodexDesktopBridge() ? "使用 ChatGPT 登录" : mode.label}
            </span>
          <span className="mt-1 block text-muted-foreground">
            {mode.description}
          </span>
        </button>
      ))}
    </div>
  );
}

export function ProviderModelSelect({
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
      <label className="text-sm ">{label}</label>
      <Select onValueChange={onValueChange} value={value}>
        <SelectTrigger className="">
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

export function ProviderAuthModePicker({
  authMode,
  onChange,
  provider,
}: {
  authMode: ProviderAuthMode;
  onChange: (mode: ProviderAuthMode) => void;
  provider: ProviderChoice;
}) {
  const modes: Array<{ id: ProviderAuthMode; label: string }> =
    provider.id === "openai"
      ? [
          { id: "oauth-browser", label: "Browser" },
          { id: "oauth-headless", label: "Headless" },
          { id: "api-key", label: "API Key" },
        ]
      : [{ id: "api-key", label: "API Key" }];

  if (modes.length === 1) return null;

  return (
    <div className="flex gap-1 overflow-x-auto rounded-lg bg-muted p-1">
      {modes.map((mode) => (
        <button
          className={cn(
            "min-w-fit flex-1 rounded-md px-2 py-1.5 text-sm transition-colors",
            authMode === mode.id
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
          key={mode.id}
          onClick={() => onChange(mode.id)}
          type="button"
        >
          {mode.label}
        </button>
      ))}
    </div>
  );
}

export function ProviderField({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border bg-muted/50 px-3 py-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
      <span className="shrink-0 text-sm text-muted-foreground">{label}</span>
      <span className="min-w-0 break-all text-left text-sm leading-5 sm:text-right">
        {value}
      </span>
    </div>
  );
}
