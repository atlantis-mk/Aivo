import type {
  ProviderAuthMode,
  ProviderChoice,
} from "@/features/providers/provider-types";

export function providerSubmitLabel(
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

export function oauthStatusLabel(status?: string, validated?: boolean) {
  if (validated || status === "success") return "已连接";
  if (status === "failed") return "授权失败";
  if (status === "cancelled") return "已取消";
  if (status === "pending") return "自动检查中";
  return "等待开始";
}
