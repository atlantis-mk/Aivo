import { useState } from "react";
import {
  Alert02Icon,
  Delete02Icon,
  PackageOpenIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  extensionCapabilityBadges,
  extensionRuntimeLabel,
  extensionStatusLabel,
} from "@/features/projects/extension-install-model";
import {
  setExtensionInstalledEnabled,
  uninstallExtension,
  type ExtensionInstall,
} from "@/services/aivo";

export function ExtensionInstallList({
  items,
  loading,
  onReload,
  query,
}: {
  items: ExtensionInstall[];
  loading: boolean;
  onReload: () => Promise<void>;
  query: string;
}) {
  const [pendingId, setPendingId] = useState("");

  async function setEnabled(item: ExtensionInstall, enabled: boolean) {
    setPendingId(item.id);
    try {
      await setExtensionInstalledEnabled(item.id, enabled);
      await onReload();
      toast.success(enabled ? "扩展已启用" : "扩展已停用");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
      await onReload();
    } finally {
      setPendingId("");
    }
  }

  async function uninstall(item: ExtensionInstall) {
    setPendingId(item.id);
    try {
      await uninstallExtension(item.id);
      await onReload();
      toast.success("扩展及 Aivo 托管副本已删除，原始源文件未修改");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setPendingId("");
    }
  }

  if (!items.length) {
    return (
      <Empty className="border">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <HugeiconsIcon icon={PackageOpenIcon} />
          </EmptyMedia>
          <EmptyTitle>{query ? "没有匹配的扩展" : "尚未安装扩展"}</EmptyTitle>
          <EmptyDescription>
            {query
              ? "试试其他名称、运行方式或权限关键词。"
              : "点击右上角的加号，从本地文件夹预览并安装 Manifest v2 扩展。"}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <ItemGroup className="gap-3">
      {items.map((item) => {
        const pending = loading || pendingId === item.id;
        const hasError = item.status === "error" || Boolean(item.error);
        return (
          <Item key={item.id} variant="outline">
            <ItemMedia variant="icon">
              {pendingId === item.id ? <Spinner /> : <HugeiconsIcon icon={PackageOpenIcon} />}
            </ItemMedia>
            <ItemContent>
              <ItemTitle>
                {item.summary.name}
                <Badge variant={hasError ? "destructive" : item.enabled ? "default" : "secondary"}>
                  {extensionStatusLabel(item.status, item.enabled)}
                </Badge>
                <Badge variant="secondary">
                  {item.installMode === "managed" ? "Aivo 托管" : "待迁移"}
                </Badge>
                <Badge variant="outline">{extensionRuntimeLabel(item.summary.runtimeType)}</Badge>
              </ItemTitle>
              <ItemDescription>{item.summary.description || item.id}</ItemDescription>
              <div className="flex flex-wrap gap-2">
                {extensionCapabilityBadges(item.summary).map((label) => (
                  <Badge key={label} variant="outline">{label}</Badge>
                ))}
                <Badge variant="outline">v{item.summary.version}</Badge>
              </div>
              {item.error ? <p className="text-xs text-destructive">{item.error}</p> : null}
            </ItemContent>
            <ItemActions>
              <Switch
                aria-label={`${item.summary.name}${item.enabled ? "停用" : "启用"}`}
                checked={item.enabled}
                disabled={pending}
                onCheckedChange={(enabled) => void setEnabled(item, enabled)}
              />
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button aria-label="卸载扩展" disabled={pending} size="icon" title="卸载扩展" variant="ghost">
                    <HugeiconsIcon icon={Delete02Icon} />
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogMedia>
                      <HugeiconsIcon icon={Alert02Icon} />
                    </AlertDialogMedia>
                    <AlertDialogTitle>卸载“{item.summary.name}”？</AlertDialogTitle>
                    <AlertDialogDescription>
                      Aivo 会停止扩展，并删除内部托管副本和安装记录。最初用于导入的源文件夹不会被修改或删除。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction onClick={() => void uninstall(item)} variant="destructive">
                      卸载
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </ItemActions>
          </Item>
        );
      })}
    </ItemGroup>
  );
}
