import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { McpServerDraftForm } from "@/features/projects/extension-settings-mcp-form";
import {
  canSaveMcpDraft,
  compactStrings,
  emptyMcpServer,
  MCP_IMPORT_PLACEHOLDER,
  normalizeMcpDraft,
  parseMcpServersImport,
  rowsToMap,
  type KeyValueRow,
  type McpAddInputMode,
} from "@/features/projects/extension-settings-model";
import { type MCPServerConfig } from "@/services/aivo";

export function AddToolDialog({
  loading,
  onOpenChange,
  onSaveMCPServer,
  onSaveMCPServers,
  open,
}: {
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSaveMCPServer: (server: MCPServerConfig) => Promise<void>;
  onSaveMCPServers: (servers: MCPServerConfig[]) => Promise<void>;
  open: boolean;
}) {
  const [mcpImportText, setMcpImportText] = useState("");
  const [mcpInputMode, setMcpInputMode] = useState<McpAddInputMode>("json");
  const [draft, setDraft] = useState<MCPServerConfig>(() => ({
    ...emptyMcpServer(),
    enabled: true,
  }));
  const [argRows, setArgRows] = useState<string[]>([""]);
  const [envRows, setEnvRows] = useState<KeyValueRow[]>([{ key: "", value: "" }]);
  const [headerRows, setHeaderRows] = useState<KeyValueRow[]>([{ key: "", value: "" }]);
  const [rootRows, setRootRows] = useState<string[]>([""]);
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    if (!open) return;
    setLocalError("");
    setDraft({ ...emptyMcpServer(), enabled: true });
    setArgRows([""]);
    setEnvRows([{ key: "", value: "" }]);
    setHeaderRows([{ key: "", value: "" }]);
    setRootRows([""]);
    setMcpInputMode("json");
  }, [open]);

  const preparedMcpDraft = normalizeMcpDraft({
    ...draft,
    args: compactStrings(argRows),
    env: rowsToMap(envRows),
    headers: rowsToMap(headerRows),
    roots: compactStrings(rootRows),
  });
  const canSubmit = mcpInputMode === "json"
    ? Boolean(mcpImportText.trim())
    : canSaveMcpDraft(preparedMcpDraft);

  async function submit() {
    if (!canSubmit) return;
    setLocalError("");
    try {
      if (mcpInputMode === "json") {
        const servers = parseMcpServersImport(mcpImportText);
        await onSaveMCPServers(servers);
        setMcpImportText("");
      } else {
        await onSaveMCPServer(preparedMcpDraft);
      }
      onOpenChange(false);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[min(760px,85vh)] flex-col overflow-hidden sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>添加 MCP server</DialogTitle>
          <DialogDescription>
            添加 MCP server，把对应工具加入当前工具目录。
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1 pr-3">
            <Tabs
              value={mcpInputMode}
              onValueChange={(value) => setMcpInputMode(value as McpAddInputMode)}
              className="gap-3"
            >
              <TabsList>
                <TabsTrigger value="json">JSON</TabsTrigger>
                <TabsTrigger value="manual">手动</TabsTrigger>
              </TabsList>
              <TabsContent value="json" className="p-0">
                <div className="grid gap-3 rounded-md border p-3">
                  <Label>粘贴 MCP JSON</Label>
                  <Textarea
                    autoFocus
                    className="min-h-80 resize-y font-mono text-xs"
                    placeholder={MCP_IMPORT_PLACEHOLDER}
                    value={mcpImportText}
                    onChange={(event) => setMcpImportText(event.target.value)}
                  />
                </div>
              </TabsContent>
              <TabsContent value="manual" className="p-0">
                <McpServerDraftForm
                  argRows={argRows}
                  draft={draft}
                  envRows={envRows}
                  headerRows={headerRows}
                  onArgRowsChange={setArgRows}
                  onDraftChange={setDraft}
                  onEnvRowsChange={setEnvRows}
                  onHeaderRowsChange={setHeaderRows}
                  onRootRowsChange={setRootRows}
                  rootRows={rootRows}
                  showEnabledToggle
                  transportEditable
                />
              </TabsContent>
            </Tabs>
        </ScrollArea>

        {localError ? <div className="text-xs text-destructive">{localError}</div> : null}

        <DialogFooter>
          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button type="button" variant="outline">
                取消
              </Button>
            </DialogClose>
            <Button disabled={loading || !canSubmit} onClick={() => void submit()} type="button">
              {loading ? "处理中" : mcpInputMode === "json" ? "导入" : "添加"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
