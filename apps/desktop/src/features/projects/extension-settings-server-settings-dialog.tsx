import { Button } from "@/components/ui/button";
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { McpServerDraftForm } from "@/features/projects/extension-settings-mcp-form";
import { canSaveMcpDraft } from "@/features/projects/extension-settings-model";
import {
  McpPromptSection,
  McpResourceSection,
  McpServerPreviewStack,
  McpServerSummaryPanel,
} from "@/features/projects/extension-settings-server-settings-sections";
import { useMcpServerSettingsState } from "@/features/projects/extension-settings-server-settings-state";
import type { MCPServerListItem } from "@/services/aivo";

export function McpServerSettingsDialog({
  item,
  onClose,
  onReload,
  open,
  sessionId,
}: {
  item: MCPServerListItem;
  onClose: () => void;
  onReload: () => Promise<void>;
  open: boolean;
  sessionId?: string;
}) {
  const settings = useMcpServerSettingsState({
    item,
    onClose,
    onReload,
    open,
    sessionId,
  });

  return (
    <DialogContent className="flex h-[min(760px,85vh)] flex-col overflow-hidden sm:max-w-3xl">
      <DialogHeader>
        <DialogTitle>
          {settings.server.displayName || settings.server.name || settings.server.id}
        </DialogTitle>
        <DialogDescription>
          如需切换 MCP 服务器类型，请先卸载当前配置。
        </DialogDescription>
      </DialogHeader>
      <ScrollArea className="h-0 min-h-0 flex-1 overflow-hidden pr-3">
        <div className="grid gap-4">
          <McpServerDraftForm
            argRows={settings.argRows}
            draft={settings.draft}
            descriptionGenerationError={settings.descriptionGenerationError}
            descriptionGenerating={settings.generatingDescription}
            envRows={settings.envRows}
            headerRows={settings.headerRows}
            onArgRowsChange={settings.setArgRows}
            onDraftChange={settings.setDraft}
            onEnvRowsChange={settings.setEnvRows}
            onHeaderRowsChange={settings.setHeaderRows}
            onGenerateDescription={() => void settings.generateDescription()}
            onRootRowsChange={settings.setRootRows}
            rootRows={settings.rootRows}
          />

          <McpServerSummaryPanel
            loadingLog={settings.loadingLog}
            loadingOAuth={settings.loadingOAuth}
            onConnectOAuth={() => void settings.connectOAuth()}
            onLoadLog={() => void settings.loadLog()}
            onLoadOAuthDiscovery={() => void settings.loadOAuthDiscovery()}
            onProbe={() => void settings.probeServer()}
            onRefreshOAuthStatus={() => void settings.refreshOAuthStatus()}
            prompts={settings.prompts}
            resources={settings.resources}
            server={settings.server}
            templates={settings.templates}
            tools={settings.tools}
          />

          <McpPromptSection
            insertingPromptId={settings.insertingPromptId}
            loadingPromptId={settings.loadingPromptId}
            onInputChange={settings.updatePromptInput}
            onInsert={(prompt) => void settings.insertPrompt(prompt)}
            onRun={(prompt) => void settings.loadPrompt(prompt)}
            promptInputs={settings.promptInputs}
            prompts={settings.prompts}
            sessionId={sessionId}
          />

          <McpResourceSection
            insertingResourceId={settings.insertingResourceId}
            loadingResourceId={settings.loadingResourceId}
            onInsert={(resource, uri) =>
              void settings.insertResource(resource, uri)
            }
            onRead={(resource, uri) => void settings.loadResource(resource, uri)}
            onTemplateInputChange={settings.updateTemplateInput}
            resources={settings.resources}
            sessionId={sessionId}
            templateInputs={settings.templateInputs}
            templates={settings.templates}
          />

          <McpServerPreviewStack
            log={settings.log}
            logError={settings.logError}
            oauthDiscovery={settings.oauthDiscovery}
            oauthError={settings.oauthError}
            oauthStart={settings.oauthStart}
            oauthStatus={settings.oauthStatus}
            promptError={settings.promptError}
            promptResult={settings.promptResult}
            resourceError={settings.resourceError}
            resourceResult={settings.resourceResult}
            saveError={settings.saveError}
            serverError={settings.server.error}
          />
        </div>
      </ScrollArea>
      <DialogFooter>
        <DialogClose asChild>
          <Button type="button" variant="outline">
            取消
          </Button>
        </DialogClose>
        <Button
          disabled={
            settings.saving ||
            settings.generatingDescription ||
            !canSaveMcpDraft(settings.draft)
          }
          onClick={() => void settings.saveSettings()}
          type="button"
        >
          {settings.saving ? "保存中" : "保存"}
        </Button>
      </DialogFooter>
    </DialogContent>
  );
}
