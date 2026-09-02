import { useEffect, useMemo, useState } from "react";
import { AlertCircle, CheckCircle2, Download, RefreshCw, ShieldCheck } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import {
  desktopUpdateAction,
  desktopUpdateStatusLabel,
  initialDesktopUpdateState,
  type DesktopUpdateAction,
  type DesktopUpdateState,
} from "@/features/updates/desktop-update-model";

export function DesktopUpdateSettings() {
  const [state, setState] = useState<DesktopUpdateState>(initialDesktopUpdateState);
  const [bridgeError, setBridgeError] = useState("");

  useEffect(() => {
    const unsubscribe = window.aivo.updates.onState(setState);
    void window.aivo.updates.getState().then(setState).catch(() => {
      setBridgeError("无法读取桌面更新服务。请重新启动 Aivo 后再试。");
    });
    return unsubscribe;
  }, []);

  const primary = useMemo(
    () => desktopUpdateAction(state, window.aivo.platform),
    [state],
  );

  async function runAction(action: DesktopUpdateAction) {
    setBridgeError("");
    try {
      const next = await window.aivo.updates[action]();
      setState(next);
    } catch {
      setBridgeError("更新操作没有完成，请稍后重试。");
    }
  }

  return (
    <div className="h-full overflow-x-hidden overflow-y-auto">
      <div className="mx-auto flex min-w-0 w-full max-w-3xl flex-col gap-5 px-5 py-6 sm:px-8 sm:py-8">
        <div className="space-y-1">
          <h1 className="text-lg font-semibold tracking-tight">软件更新</h1>
          <p className="text-sm text-muted-foreground">
            自动检查稳定版本，并在打开安装包前核对 R2 与 GitHub Release 的 SHA-256。
          </p>
        </div>

        {(bridgeError || state.phase === "error") && (
          <Alert variant="destructive">
            <AlertCircle />
            <AlertTitle>更新暂不可用</AlertTitle>
            <AlertDescription>{bridgeError || state.message}</AlertDescription>
          </Alert>
        )}

        <Card className="min-w-0">
          <CardHeader>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0 space-y-1">
                <CardTitle>Aivo {state.currentVersion ? `v${state.currentVersion}` : ""}</CardTitle>
                <CardDescription className="break-words">{state.message}</CardDescription>
              </div>
              <Badge variant={state.phase === "error" ? "destructive" : "secondary"}>
                {desktopUpdateStatusLabel(state.phase)}
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {state.phase === "downloading" && (
              <div className="space-y-2" aria-live="polite">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>下载并验证 v{state.availableVersion}</span>
                  <span>{state.progress}%</span>
                </div>
                <Progress aria-label="更新下载进度" value={state.progress} />
              </div>
            )}

            <div className="grid gap-3 rounded-lg border bg-muted/30 p-4 text-sm sm:grid-cols-2">
              <div className="flex min-w-0 gap-2.5">
                <ShieldCheck className="mt-0.5 size-4 shrink-0 text-primary" />
                <div>
                  <p className="font-medium">双源完整性校验</p>
                  <p className="mt-1 break-words text-xs leading-relaxed text-muted-foreground">
                    名称、大小和 SHA-256 必须同时匹配固定 R2 通道与 GitHub Release。
                  </p>
                </div>
              </div>
              <div className="flex min-w-0 gap-2.5">
                <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-primary" />
                <div>
                  <p className="font-medium">由你确认安装</p>
                  <p className="mt-1 break-words text-xs leading-relaxed text-muted-foreground">
                    Aivo 不会静默运行安装包，也不会绕过操作系统安全提示。
                  </p>
                </div>
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex flex-col items-stretch gap-3 border-t sm:flex-row sm:items-center sm:justify-between">
            <p className="min-w-0 text-xs text-muted-foreground">
              {state.automaticChecksEnabled
                ? "已启用：应用启动后自动检查一次稳定通道。"
                : "开发版本不会在启动时自动联网检查。"}
            </p>
            {primary && (
              <Button
                className="w-full sm:w-auto"
                onClick={() => void runAction(primary.action)}
                variant={primary.action === "cancel" ? "outline" : "default"}
              >
                {primary.action === "check" && <RefreshCw />}
                {primary.action === "download" && <Download />}
                {primary.label}
              </Button>
            )}
          </CardFooter>
        </Card>

        <p className="text-xs leading-relaxed text-muted-foreground">
          当前发布包可能尚未完成平台签名。SHA-256 验证只能确认下载内容与发布记录一致，不能替代 macOS 公证、Windows 代码签名或 Linux 发行签名。
        </p>
      </div>
    </div>
  );
}
