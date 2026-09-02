export type DesktopUpdatePhase =
  | "idle"
  | "checking"
  | "up-to-date"
  | "available"
  | "downloading"
  | "ready"
  | "unsupported"
  | "error";

export type DesktopUpdateState = {
  phase: DesktopUpdatePhase;
  currentVersion: string;
  availableVersion: string;
  progress: number;
  message: string;
  errorCode: string;
  automaticChecksEnabled: boolean;
};

export type DesktopUpdateAction = "check" | "download" | "install" | "cancel";

export const initialDesktopUpdateState: DesktopUpdateState = {
  phase: "idle",
  currentVersion: "",
  availableVersion: "",
  progress: 0,
  message: "正在读取更新状态…",
  errorCode: "",
  automaticChecksEnabled: false,
};

export function desktopUpdateAction(
  state: DesktopUpdateState,
  platform: string,
): { action: DesktopUpdateAction; label: string } | null {
  switch (state.phase) {
    case "available":
      return { action: "download", label: "下载更新" };
    case "downloading":
      return { action: "cancel", label: "取消下载" };
    case "ready":
      return {
        action: "install",
        label: platform === "linux" ? "显示更新包" : "打开安装包",
      };
    case "checking":
    case "unsupported":
      return null;
    default:
      return { action: "check", label: "检查更新" };
  }
}

export function desktopUpdateStatusLabel(phase: DesktopUpdatePhase) {
  switch (phase) {
    case "checking": return "正在检查";
    case "up-to-date": return "已是最新";
    case "available": return "有可用更新";
    case "downloading": return "正在下载";
    case "ready": return "等待安装";
    case "unsupported": return "不受支持";
    case "error": return "检查失败";
    default: return "尚未检查";
  }
}
