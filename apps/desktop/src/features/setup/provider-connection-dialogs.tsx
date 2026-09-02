import { ArrowLeft, X } from "lucide-react";

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
 hasAppBridge,
} from "@/lib/app-config";
import type { ModelInfo } from "@/lib/provider-catalog";
import { CustomProviderFormFields } from "@/features/providers/custom-provider-form-fields";
import { isCustomProviderChoice } from "@/features/providers/custom-provider-form";
import { ProviderIcon } from "@/features/providers/provider-icon";
import { providerNameForPrompt } from "@/features/providers/provider-defaults";
import type {
 CustomProviderForm,
 ProviderAuthMode,
 ProviderChoice,
 ProviderDialogStep,
} from "@/features/providers/provider-types";
import {
 OpenAIAuthModeOptions,
 ProviderAuthModePicker,
 ProviderField,
 ProviderModelSelect,
} from "@/features/setup/provider-connection-parts";
import {
 oauthStatusLabel,
 providerSubmitLabel,
} from "@/features/setup/provider-connection-labels";

export type {
 CustomProviderForm,
 CustomProviderProtocol,
 CustomProviderRow,
 ProviderAuthMode,
 ProviderChoice,
 ProviderDialogStep,
} from "@/features/providers/provider-types";
export { ProviderIcon } from "@/features/providers/provider-icon";

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
 onSelectedModelIdChange,
 onSubmit,
 providerDialogStep,
 saving,
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
 onSelectedModelIdChange: (value: string) => void;
 onSubmit: () => void;
 providerDialogStep: ProviderDialogStep;
 saving: boolean;
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

 <OpenAIAuthModeOptions onSelect={onSelectOpenAIAuthMode} />

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
 <ArrowLeft />
 </Button>
 ) : null}
 <ProviderIcon provider={activeProvider} size="sm" />
 <DialogTitle className="min-w-0 truncate">连接 {activeProvider.name}</DialogTitle>
 </div>
 <DialogClose asChild>
 <Button aria-label="关闭" size="icon" variant="ghost">
 <X />
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
 {showModelSelect ? (
 <ProviderModelSelect
 label="默认模型"
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
