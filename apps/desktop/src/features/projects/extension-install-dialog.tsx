import { useEffect, useState } from "react";
import {
  Alert02Icon,
  Folder01Icon,
  InformationCircleIcon,
  PackageAddIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldTitle,
} from "@/components/ui/field";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  extensionCapabilityBadges,
  extensionRuntimeLabel,
} from "@/features/projects/extension-install-model";
import {
  installExtension,
  previewExtensionInstall,
  type ExtensionInstallPreview,
} from "@/services/aivo";

export function ExtensionInstallDialog({
  onInstalled,
  onOpenChange,
  open,
}: {
  onInstalled: () => Promise<void>;
  onOpenChange: (open: boolean) => void;
  open: boolean;
}) {
  const [preview, setPreview] = useState<ExtensionInstallPreview>();
  const [enable, setEnable] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setPreview(undefined);
    setEnable(true);
    setLoading(false);
    setError("");
  }, [open]);

  async function chooseDirectory() {
    setError("");
    const path = await window.aivo.selectExtensionDirectory();
    if (!path) return;
    setLoading(true);
    try {
      setPreview(await previewExtensionInstall(path));
    } catch (err) {
      setPreview(undefined);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  async function install() {
    if (!preview) return;
    setLoading(true);
    setError("");
    try {
      await installExtension(preview.path, preview.integrity, enable);
      await onInstalled();
      toast.success(preview.update ? "扩展已更新" : "扩展已安装");
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  const summary = preview?.summary;
  const riskTitle = summary?.executable
    ? "此扩展可执行本地代码"
    : summary?.runtimeType === "external"
      ? "此扩展会连接外部服务"
      : "此扩展不启动本地进程";
  const riskDescription = summary?.executable
    ? "启用后，它会以当前系统用户权限运行。上方权限声明用于告知和授权能力，不等同于操作系统沙箱。"
    : summary?.runtimeType === "external"
      ? "启用后，Aivo 会连接 Manifest 声明的 HTTPS 服务；凭据仍由 Host 安全存储并按槽位绑定。"
      : "Aivo 将直接加载声明的静态界面和资源，不需要独立 Web 服务或固定端口。";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{preview?.update ? "更新本地扩展" : "安装本地扩展"}</DialogTitle>
          <DialogDescription>
            选择包含 Manifest v2 的源文件夹。确认后，Aivo 会把校验通过的完整扩展复制到内部托管目录。
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1 pr-3">
          <div className="flex flex-col gap-4">
            {!preview ? (
              <Empty className="border">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    {loading ? (
                      <Spinner />
                    ) : (
                      <HugeiconsIcon icon={PackageAddIcon} />
                    )}
                  </EmptyMedia>
                  <EmptyTitle>选择扩展文件夹</EmptyTitle>
                  <EmptyDescription>
                    Aivo 只读取源文件夹并创建独立托管副本，不会修改或移动原文件。
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button disabled={loading} onClick={() => void chooseDirectory()} type="button">
                    {loading ? <Spinner /> : <HugeiconsIcon data-icon="inline-start" icon={Folder01Icon} />}
                    浏览文件夹
                  </Button>
                </EmptyContent>
              </Empty>
            ) : null}

            {summary ? (
              <>
                <Card>
                  <CardHeader>
                    <CardTitle>{summary.name}</CardTitle>
                    <CardDescription>{summary.description || summary.id}</CardDescription>
                    <CardAction>
                      <Badge variant="outline">v{summary.version}</Badge>
                    </CardAction>
                  </CardHeader>
                  <CardContent className="flex flex-col gap-3">
                    <div className="flex flex-wrap gap-2">
                      <Badge variant="secondary">Manifest {summary.apiVersion}</Badge>
                      <Badge variant="outline">{extensionRuntimeLabel(summary.runtimeType)}</Badge>
                      {extensionCapabilityBadges(summary).map((label) => (
                        <Badge key={label} variant="outline">{label}</Badge>
                      ))}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {summary.permissions?.length ? summary.permissions.map((permission) => (
                        <Badge key={permission} variant="secondary">权限：{permission}</Badge>
                      )) : <Badge variant="outline">无额外 Aivo 权限</Badge>}
                      {summary.network ? <Badge variant="secondary">需要网络</Badge> : null}
                      {summary.credentialIds?.map((credential) => (
                        <Badge key={credential} variant="secondary">凭据槽：{credential}</Badge>
                      ))}
                      {summary.platforms?.map((platform) => (
                        <Badge key={platform} variant="outline">平台：{platform}</Badge>
                      ))}
                    </div>
                    <p className="break-all text-xs text-muted-foreground">导入来源：{preview.path}</p>
                  </CardContent>
                </Card>

                <Alert>
                  <HugeiconsIcon icon={InformationCircleIcon} />
                  <AlertTitle>安装后由 Aivo 托管</AlertTitle>
                  <AlertDescription>
                    运行时只加载内部只读副本，并在启动和启用前校验完整性。安装完成后，移动、删除或修改源文件夹都不会影响已安装版本。
                  </AlertDescription>
                </Alert>

                <Alert variant={summary.executable ? "destructive" : "default"}>
                  <HugeiconsIcon icon={summary.executable ? Alert02Icon : InformationCircleIcon} />
                  <AlertTitle>
                    {riskTitle}
                  </AlertTitle>
                  <AlertDescription>
                    {riskDescription}
                    {summary.command ? (
                      <p className="mt-2 break-all font-mono">启动命令：{summary.command}</p>
                    ) : null}
                  </AlertDescription>
                </Alert>

                <FieldGroup>
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldTitle>安装后立即启用</FieldTitle>
                      <FieldDescription>
                        关闭后仅保存安装记录，之后可以在扩展列表中启用。
                      </FieldDescription>
                    </FieldContent>
                    <Switch checked={enable} onCheckedChange={setEnable} />
                  </Field>
                </FieldGroup>

                <Button onClick={() => void chooseDirectory()} type="button" variant="outline">
                  <HugeiconsIcon data-icon="inline-start" icon={Folder01Icon} />
                  重新选择
                </Button>
              </>
            ) : null}

            {error ? (
              <Alert variant="destructive">
                <HugeiconsIcon icon={Alert02Icon} />
                <AlertTitle>无法安装扩展</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}
          </div>
        </ScrollArea>

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline">取消</Button>
          </DialogClose>
          <Button disabled={!preview || loading} onClick={() => void install()} type="button">
            {loading ? <Spinner /> : <HugeiconsIcon data-icon="inline-start" icon={PackageAddIcon} />}
            {preview?.update ? "确认更新" : "确认安装"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
