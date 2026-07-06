import { ArrowLeft, Plus, Trash2, X } from "lucide-react";

import type { domain } from "../../../bridge/go/models";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
 Dialog,
 DialogClose,
 DialogContent,
 DialogDescription,
 DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
 Select,
 SelectContent,
 SelectItem,
 SelectTrigger,
 SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { hasAppBridge } from "@/lib/app-config";
import type { ModelInfo } from "@/lib/provider-catalog";

export type ProviderAuthMode = "oauth-browser" | "oauth-headless" | "api-key";
export type ProviderDialogStep = "options" | "details";
export type CustomProviderProtocol = "openai" | "openai-compatible" | "anthropic" | "google" | "openrouter";
export type CustomProviderRow = { id: string; name: string; value: string };
export type CustomProviderForm = {
 providerId: string;
 displayName: string;
 protocol: CustomProviderProtocol;
 baseUrl: string;
 apiKey: string;
 models: CustomProviderRow[];
 headers: CustomProviderRow[];
};

export type ProviderChoice = {
 id: string;
 name: string;
 iconClassName?: string;
 iconSrc?: string;
 custom?: boolean;
 opensProviderPicker?: boolean;
};

const openAIAuthModes: Array<{ id: ProviderAuthMode; label: string; description: string }> = [
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

const customProviderProtocols: Array<{ id: CustomProviderProtocol; label: string }> = [
 { id: "openai", label: "OpenAI Responses" },
 { id: "openai-compatible", label: "OpenAI Compatible" },
 { id: "anthropic", label: "Anthropic Messages" },
 { id: "google", label: "Google Gemini" },
 { id: "openrouter", label: "OpenRouter" },
];

export function ProviderConnectionDialogs({
 activeProvider,
 apiKey,
 authMode,
 authSuccessMessage,
 callbackInput,
 customProviderForm,
 error,
 models,
 oauthReady,
 oauthStarted,
 oauthStartResult,
 oauthStatus,
 onApiKeyChange,
 onCallbackInputChange,
 onClose,
 onCustomProviderFormChange,
 onProviderDialogStepChange,
 onResetAuthMode,
 onSelectOpenAIAuthMode,
 onSelectedAuxiliaryModelIdChange,
 onSelectedModelIdChange,
 onSubmit,
 providerDialogStep,
 saving,
 selectedAuxiliaryModelId,
 selectedModelId,
 showModelSelect,
 submitDisabled,
}: {
 activeProvider: ProviderChoice | null;
 apiKey: string;
 authMode: ProviderAuthMode;
 authSuccessMessage: string;
 callbackInput: string;
 customProviderForm: CustomProviderForm;
 error: string;
 models: ModelInfo[];
 oauthReady: boolean;
 oauthStarted: boolean;
 oauthStartResult: domain.ProviderAuthStartResult | null;
 oauthStatus: domain.ProviderAuthStatus | null;
 onApiKeyChange: (value: string) => void;
 onCallbackInputChange: (value: string) => void;
 onClose: () => void;
 onCustomProviderFormChange: (form: CustomProviderForm) => void;
 onProviderDialogStepChange: (step: ProviderDialogStep) => void;
 onResetAuthMode: (mode: ProviderAuthMode) => void;
 onSelectOpenAIAuthMode: (mode: ProviderAuthMode) => void;
 onSelectedAuxiliaryModelIdChange?: (value: string) => void;
 onSelectedModelIdChange: (value: string) => void;
 onSubmit: () => void;
 providerDialogStep: ProviderDialogStep;
 saving: boolean;
 selectedAuxiliaryModelId?: string;
 selectedModelId: string;
 showModelSelect: boolean;
 submitDisabled: boolean;
}) {
 return (
 <>
 <Dialog
 open={Boolean(activeProvider && activeProvider.id === "openai" && providerDialogStep === "options")}
 onOpenChange={(open: boolean) => {
 if (!open) onClose();
 }}
 >
 {activeProvider && activeProvider.id === "openai" ? (
 <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-hidden sm:max-w-md" showCloseButton={false}>
 <div className="flex flex-col gap-4">
 <div className="flex items-center gap-3">
 <ProviderIcon provider={activeProvider} size="sm" />
 <DialogTitle>选择 OpenAI 登录方式</DialogTitle>
 </div>
 <DialogDescription>选择一种方式连接你的 OpenAI 账号。</DialogDescription>

 <div className="flex  flex-col gap-2">
 {openAIAuthModes.map((mode) => (
 <button
 className="rounded-lg border px-3 py-2 text-left text-sm transition-colors hover:bg-muted"
 key={mode.id}
 onClick={() => onSelectOpenAIAuthMode(mode.id)}
 type="button"
 >
 <span className="block ">{mode.label}</span>
 <span className="mt-1 block text-muted-foreground">{mode.description}</span>
 </button>
 ))}
 </div>

 <div className="flex justify-end">
 <DialogClose asChild>
 <Button variant="secondary">取消</Button>
 </DialogClose>
 </div>
 </div>
 </DialogContent>
 ) : null}
 </Dialog>

 <Dialog
 open={Boolean(activeProvider && (activeProvider.id !== "openai" || providerDialogStep === "details"))}
 onOpenChange={(open: boolean) => {
 if (!open) onClose();
 }}
 >
 {activeProvider ? (
 <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-hidden sm:max-w-lg" showCloseButton={false}>
 <div className="flex min-w-0 items-center justify-between gap-3">
 <div className="flex min-w-0 items-center gap-3">
 {activeProvider.id === "openai" ? (
 <Button
 aria-label="返回登录方式选择"
 onClick={() => onProviderDialogStepChange("options")}
 size="icon"
 variant="ghost"
 >
 <ArrowLeft className="size-5" />
 </Button>
 ) : null}
 <ProviderIcon provider={activeProvider} size="sm" />
 <DialogTitle className="min-w-0 truncate">连接 {activeProvider.name}</DialogTitle>
 </div>
 <DialogClose asChild>
 <Button aria-label="关闭" size="icon" variant="ghost">
 <X className="size-5" />
 </Button>
 </DialogClose>
 </div>

 <div className="flex flex-col gap-4">
 <ScrollArea className="max-h-[min(60vh,520px)] pr-3">
 <div className="flex  flex-col gap-4">
 {authSuccessMessage ? (
 <Alert className=" text-left">
 <AlertTitle>{authSuccessMessage}</AlertTitle>
 <AlertDescription>选择要默认使用的模型。</AlertDescription>
 </Alert>
 ) : null}
 {activeProvider.id !== "openai" ? (
 <ProviderAuthModePicker
 authMode={authMode}
 onChange={onResetAuthMode}
 provider={activeProvider}
 />
 ) : null}
 {isCustomProviderChoice(activeProvider) ? (
 <CustomProviderFormFields
 form={customProviderForm}
 onChange={onCustomProviderFormChange}
 />
 ) : authMode === "api-key" ? (
 <div className="flex  flex-col gap-2 text-left">
 <label className="text-sm ">
 {providerNameForPrompt(activeProvider.id)} API 密钥
 </label>
 <Input
 aria-label={`${activeProvider.name} API key`}
 onChange={(event) => onApiKeyChange(event.target.value)}
 placeholder="API 密钥"
 type="password"
 value={apiKey}
 />
 </div>
 ) : oauthReady ? null : authMode === "oauth-browser" && activeProvider.id === "openai" && !oauthStarted ? (
 <ProviderField label="操作" value="点击下方按钮打开浏览器授权" />
 ) : authMode === "oauth-browser" && activeProvider.id === "openai" && oauthStarted && !hasAppBridge() ? (
 <div className="flex  flex-col gap-2 text-left">
 <label className="text-sm ">授权回调</label>
 <Input
 aria-label="OpenAI OAuth callback"
 onChange={(event) => onCallbackInputChange(event.target.value)}
 placeholder="粘贴回调 URL 或 authorization code"
 value={callbackInput}
 />
 <ProviderField label="状态" value={oauthStatusLabel(oauthStatus?.status, oauthReady)} />
 </div>
 ) : authMode === "oauth-headless" && activeProvider.id === "openai" && !oauthStarted ? (
 <ProviderField label="操作" value="点击下方按钮生成确认码" />
 ) : authMode === "oauth-headless" && activeProvider.id === "openai" && oauthStarted ? (
 <div className="flex  flex-col gap-2 text-left">
 <ProviderField label="打开链接" value={oauthStartResult?.url || "https://auth.openai.com/codex/device"} />
 <ProviderField label="确认码" value={oauthStartResult?.userCode || "等待生成"} />
 <ProviderField label="状态" value={oauthStatusLabel(oauthStatus?.status, oauthReady)} />
 </div>
 ) : authMode === "oauth-browser" && activeProvider.id === "openai" && oauthStarted ? (
 <div className="flex  flex-col gap-2 text-left">
 <ProviderField label="授权链接" value={oauthStartResult?.url || "等待生成"} />
 <ProviderField label="回调验证" value={oauthReady ? "已完成" : "本地回调自动检测中"} />
 <ProviderField label="状态" value={oauthStatusLabel(oauthStatus?.status, oauthReady)} />
 </div>
 ) : null}
 {showModelSelect && onSelectedAuxiliaryModelIdChange ? (
 <div className="grid gap-3 sm:grid-cols-2">
 <ProviderModelSelect
 label="默认模型"
 models={models}
 onValueChange={onSelectedModelIdChange}
 value={selectedModelId}
 />
 <ProviderModelSelect
 label="辅助模型"
 models={models}
 onValueChange={onSelectedAuxiliaryModelIdChange}
 value={selectedAuxiliaryModelId || selectedModelId}
 />
 </div>
 ) : showModelSelect ? (
 <ProviderModelSelect
 label="模型"
 models={models}
 onValueChange={onSelectedModelIdChange}
 value={selectedModelId}
 />
 ) : null}
 </div>
 </ScrollArea>

 <div className="flex items-center justify-end gap-2">
 <Button disabled={submitDisabled} onClick={onSubmit}>
 {saving ? "连接中" : providerSubmitLabel(activeProvider, authMode, oauthStarted, oauthReady)}
 </Button>
 </div>

 {error ? (
 <Alert className=" text-left" variant="destructive">
 <AlertTitle>无法连接提供方</AlertTitle>
 <AlertDescription>{error}</AlertDescription>
 </Alert>
 ) : null}
 </div>
 </DialogContent>
 ) : null}
 </Dialog>
 </>
 );
}

export function ProviderIcon({ provider, size }: { provider: ProviderChoice; size: "sm" | "lg" }) {
 const wrapperClassName = size === "lg" ? "size-11" : "size-5 shrink-0";
 const imageClassName = size === "lg" ? "size-7" : "size-4";

 if (provider.iconSrc) {
 return (
 <span className={cn("grid place-items-center rounded-full bg-card text-foreground", wrapperClassName)}>
 <img alt="" className={imageClassName} src={provider.iconSrc} />
 </span>
 );
 }

 return <span className={cn("rounded-full bg-primary", wrapperClassName, provider.iconClassName)} />;
}

function isCustomProviderChoice(provider: ProviderChoice) {
 return provider.id === "custom-api";
}

function CustomProviderFormFields({
 form,
 onChange,
}: {
 form: CustomProviderForm;
 onChange: (form: CustomProviderForm) => void;
}) {
 function updateField(field: keyof Omit<CustomProviderForm, "models" | "headers">, value: string) {
 onChange({ ...form, [field]: value });
 }

 function updateRow(section: "models" | "headers", id: string, field: "name" | "value", value: string) {
 onChange({
 ...form,
 [section]: form[section].map((row) => (row.id === id ? { ...row, [field]: value } : row)),
 });
 }

 function addRow(section: "models" | "headers") {
 onChange({ ...form, [section]: [...form[section], emptyCustomRow()] });
 }

 function removeRow(section: "models" | "headers", id: string) {
 const nextRows = form[section].filter((row) => row.id !== id);
 onChange({ ...form, [section]: nextRows.length > 0 ? nextRows : [emptyCustomRow()] });
 }

 return (
 <div className="flex flex-col gap-4 text-left">
 <CustomProviderInput
 description="使用小写字母、数字、连字符或下划线"
 label="提供商 ID"
 onChange={(value) => updateField("providerId", value)}
 placeholder="myprovider"
 value={form.providerId}
 />
 <CustomProviderInput
 label="显示名称"
 onChange={(value) => updateField("displayName", value)}
 placeholder="我的 AI 提供商"
 value={form.displayName}
 />
 <div className="flex flex-col gap-1.5">
 <label className="text-sm ">协议</label>
 <Select
 onValueChange={(value: string) => updateField("protocol", value as CustomProviderProtocol)}
 value={form.protocol}
 >
 <SelectTrigger className="h-9  px-3 text-sm">
 <SelectValue />
 </SelectTrigger>
 <SelectContent>
 {customProviderProtocols.map((protocol) => (
 <SelectItem key={protocol.id} value={protocol.id}>
 {protocol.label}
 </SelectItem>
 ))}
 </SelectContent>
 </Select>
 </div>
 <CustomProviderInput
 label="基础 URL"
 onChange={(value) => updateField("baseUrl", value)}
 placeholder={customProviderBaseURLPlaceholder(form.protocol)}
 value={form.baseUrl}
 />
 <CustomProviderInput
 description="可选。如果你通过请求头管理认证，可留空。"
 label="API 密钥"
 onChange={(value) => updateField("apiKey", value)}
 placeholder="API 密钥"
 value={form.apiKey}
 />

 <CustomRows
 addLabel="添加模型"
 leftPlaceholder="model-id"
 onAdd={() => addRow("models")}
 onRemove={(id) => removeRow("models", id)}
 onUpdate={(id, field, value) => updateRow("models", id, field, value)}
 removeLabel="移除模型"
 rightPlaceholder="显示名称"
 rows={form.models}
 title="模型"
 />

 <CustomRows
 addLabel="添加请求头"
 leftPlaceholder="Header-Name"
 onAdd={() => addRow("headers")}
 onRemove={(id) => removeRow("headers", id)}
 onUpdate={(id, field, value) => updateRow("headers", id, field, value)}
 removeLabel="移除请求头"
 rightPlaceholder="value"
 rows={form.headers}
 title="请求头（可选）"
 />
 </div>
 );
}

function ProviderModelSelect({
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

function CustomProviderInput({
 description,
 label,
 onChange,
 placeholder,
 value,
}: {
 description?: string;
 label: string;
 onChange: (value: string) => void;
 placeholder: string;
 value: string;
}) {
 return (
 <label className="flex flex-col gap-1.5 text-sm ">
 {label}
 <Input
 onChange={(event) => onChange(event.target.value)}
 placeholder={placeholder}
 value={value}
 />
 {description ? <span className="text-xs font-normal text-muted-foreground">{description}</span> : null}
 </label>
 );
}

function CustomRows({
 addLabel,
 leftPlaceholder,
 onAdd,
 onRemove,
 onUpdate,
 removeLabel,
 rightPlaceholder,
 rows,
 title,
}: {
 addLabel: string;
 leftPlaceholder: string;
 onAdd: () => void;
 onRemove: (id: string) => void;
 onUpdate: (id: string, field: "name" | "value", value: string) => void;
 removeLabel: string;
 rightPlaceholder: string;
 rows: CustomProviderRow[];
 title: string;
}) {
 return (
 <div className="flex flex-col gap-2">
 <div className="text-sm  text-muted-foreground">{title}</div>
 {rows.map((row) => (
 <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-2 max-[420px]:grid-cols-[minmax(0,1fr)_auto]" key={row.id}>
 <Input
 aria-label={`${title} ID`}
 onChange={(event) => onUpdate(row.id, "name", event.target.value)}
 placeholder={leftPlaceholder}
 value={row.name}
 />
 <Input
 aria-label={`${title} 值`}
 className="max-[420px]:col-span-2 max-[420px]:col-start-1"
 onChange={(event) => onUpdate(row.id, "value", event.target.value)}
 placeholder={rightPlaceholder}
 value={row.value}
 />
 <Button
 aria-label={removeLabel}
 className="max-[420px]:col-start-2 max-[420px]:row-start-1"
 disabled={rows.length === 1}
 onClick={() => onRemove(row.id)}
 size="icon"
 type="button"
 variant="ghost"
 >
 <Trash2 className="size-4" />
 </Button>
 </div>
 ))}
 <Button className="self-start" onClick={onAdd} type="button" variant="ghost">
 <Plus className="size-4" />
 {addLabel}
 </Button>
 </div>
 );
}

function emptyCustomRow(): CustomProviderRow {
 return { id: crypto.randomUUID(), name: "", value: "" };
}

function ProviderAuthModePicker({
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

function providerSubmitLabel(
 provider: ProviderChoice,
 authMode: ProviderAuthMode,
 oauthStarted: boolean,
 providerValidated: boolean,
) {
 if (provider.id === "openai" && authMode === "oauth-browser") {
 if (providerValidated) return "完成";
 return oauthStarted ? "检查状态" : "打开浏览器";
 }
 if (provider.id === "openai" && authMode === "oauth-headless") {
 if (providerValidated) return "完成";
 return oauthStarted ? "检查状态" : "生成确认码";
 }
 return "提交";
}

function providerNameForPrompt(providerId: string) {
 if (providerId === "claude-code") return "Anthropic";
 if (providerId === "gemini") return "Google";
 if (providerId === "custom-api") return "Custom API";
 return providerDisplayName(providerId);
}

function providerDisplayName(providerId: string) {
 const knownNames: Record<string, string> = {
 anthropic: "Anthropic",
 "claude-code": "Claude Code",
 gemini: "Gemini",
 google: "Google",
 openai: "OpenAI",
 openrouter: "OpenRouter",
 };
 if (knownNames[providerId]) return knownNames[providerId];
 return providerId
 .split("-")
 .filter(Boolean)
 .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
 .join(" ");
}

function oauthStatusLabel(status?: string, validated?: boolean) {
 if (validated || status === "success") return "已连接";
 if (status === "failed") return "授权失败";
 if (status === "cancelled") return "已取消";
 if (status === "pending") return "自动检查中";
 return "等待开始";
}

function customProviderBaseURLPlaceholder(protocol: CustomProviderProtocol) {
 if (protocol === "openai") return "https://api.openai.com/v1";
 if (protocol === "anthropic") return "https://api.anthropic.com/v1";
 if (protocol === "google") return "https://generativelanguage.googleapis.com/v1beta";
 if (protocol === "openrouter") return "https://openrouter.ai/api/v1";
 return "https://api.myprovider.com/v1";
}

function ProviderField({ label, value }: { label: string; value: string }) {
 return (
 <div className="flex flex-col gap-1 rounded-lg border bg-muted/50 px-3 py-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
 <span className="shrink-0 text-sm text-muted-foreground">{label}</span>
 <span className="min-w-0 break-all text-left text-sm leading-5 sm:text-right">{value}</span>
 </div>
 );
}
