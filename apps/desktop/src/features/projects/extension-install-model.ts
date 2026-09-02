import type { ExtensionInstallSummary } from "@/services/aivo";

export function extensionRuntimeLabel(runtimeType: string) {
  const labels: Record<string, string> = {
    builtin: "内置运行时",
    process: "本地进程",
    service: "本地服务",
    external: "外部服务",
    static: "静态界面",
  };
  return labels[runtimeType] ?? runtimeType;
}

export function extensionStatusLabel(status: string, enabled: boolean) {
  if (!enabled && status !== "error") return "已停用";
  const labels: Record<string, string> = {
    discovered: "待启用",
    validated: "已确认",
    untrusted: "待确认",
    enabled: "已启用",
    trusted: "已确认",
    starting: "启动中",
    ready: "运行中",
    active: "运行中",
    draining: "停止中",
    running: "运行中",
    stopped: "已停用",
    error: "需要处理",
  };
  return labels[status] ?? status;
}

export function extensionCapabilityBadges(summary: ExtensionInstallSummary) {
  return [
    summary.tools?.length ? `${summary.tools.length} 个工具` : "",
    summary.views?.length ? `${summary.views.length} 个界面` : "",
    summary.contexts?.length ? `${summary.contexts.length} 个上下文` : "",
  ].filter(Boolean);
}
