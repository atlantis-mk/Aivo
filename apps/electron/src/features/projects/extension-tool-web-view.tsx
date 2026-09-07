import type { ReactNode } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { ExtensionToolViewRef } from "@/features/projects/extension-tool-view-model";
import type { domain } from "@/types/codex-domain";

export function ExtensionToolWebView({
  fallback,
  toolCall,
  view,
}: {
  fallback: ReactNode;
  onRequestClose: () => void;
  toolCall: domain.ToolCall;
  view: ExtensionToolViewRef;
}) {
  void toolCall;
  void view;

  return (
    <div className="flex h-full min-h-0 flex-col gap-3 overflow-auto p-3">
      <Alert variant="destructive">
        <AlertTitle>扩展页面暂不支持</AlertTitle>
        <AlertDescription>
          已保留安全的原生工具详情，你仍可查看本次调用记录。
        </AlertDescription>
      </Alert>
      {fallback}
    </div>
  );
}
