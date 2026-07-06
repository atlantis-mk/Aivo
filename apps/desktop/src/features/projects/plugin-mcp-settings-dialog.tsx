import { type ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import {
  CheckCircle2,
  FileText,
  Layers3,
  type LucideIcon,
  MessageSquareText,
  MoreHorizontal,
  Plug,
  Plus,
  RefreshCw,
  Search,
  Server,
  Settings,
  Trash2,
  TriangleAlert,
  Wrench,
} from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import {
  installPluginFromPath,
  listMCPServers,
  listPlugins,
  listToolCatalog,
  getMCPPrompt,
  insertMCPPromptIntoSession,
  insertMCPResourceIntoSession,
  probeMCPServer,
  readMCPResource,
  readMCPServerLog,
  reloadPlugins,
  saveMCPServer,
  setMCPServerEnabled,
  setPluginEnabled,
  type MCPPromptGetResult,
  type MCPPromptRecord,
  type MCPResourceReadResult,
  type MCPResourceRecord,
  type MCPServerConfig,
  type MCPServerListItem,
  type MCPOAuthDiscoveryResult,
  type MCPOAuthStartResult,
  type MCPOAuthStatus,
  type PluginListItem,
  type ToolCatalogEntry,
  discoverMCPOAuth,
  getMCPOAuthStatus,
  startMCPOAuth,
} from "@/services/aivo";

type PluginMcpSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sessionId?: string;
  workspaceRoot: string;
};

export function PluginMcpSettingsDialog({
  open,
  onOpenChange,
  sessionId,
  workspaceRoot,
}: PluginMcpSettingsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[min(780px,86vh)] max-w-5xl flex-col gap-0 overflow-hidden p-0">
        <PluginMcpSettingsContent
          active={open}
          sessionId={sessionId}
          surface="dialog"
          workspaceRoot={workspaceRoot}
        />
      </DialogContent>
    </Dialog>
  );
}

type PluginMcpSettingsContentProps = {
  active?: boolean;
  className?: string;
  sessionId?: string;
  surface?: "dialog" | "page";
  workspaceRoot?: string;
};

type PluginSettingsSection = "plugins" | "apps" | "mcp" | "skills" | "tools";
type AddToolMode = "plugin" | "mcp";
type McpAddInputMode = "json" | "manual";

export function PluginMcpSettingsContent({
  active = true,
  className,
  sessionId,
  surface = "page",
  workspaceRoot,
}: PluginMcpSettingsContentProps) {
  const [plugins, setPlugins] = useState<PluginListItem[]>([]);
  const [servers, setServers] = useState<MCPServerListItem[]>([]);
  const [tools, setTools] = useState<ToolCatalogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [pluginPath, setPluginPath] = useState("");
  const [query, setQuery] = useState("");
  const [section, setSection] = useState<PluginSettingsSection>("plugins");
  const [addOpen, setAddOpen] = useState(false);
  const [addMode, setAddMode] = useState<AddToolMode>("plugin");
  const visibleTools = useMemo(
    () =>
      workspaceRoot || tools.length > 0
        ? tools
        : mergeToolCatalogEntries([
            plugins.flatMap(pluginToolsForDisplay),
            servers.flatMap((item) =>
              (item.tools ?? []).map((tool) => ({
                name: tool.name,
                description: tool.description,
                inputSchema: tool.inputSchema,
                namespace: item.server.name || item.server.id,
                capability: tool.capability,
                riskLevel: tool.riskLevel,
                source: "mcp",
                sourceId: item.server.id,
                registrationId: tool.id,
                enabled: item.server.enabled,
              })),
            ),
          ]),
    [plugins, servers, tools, workspaceRoot],
  );
  const applicationPlugins = useMemo(
    () =>
      plugins.filter((item) => {
        const keywords = item.plugin.manifest.keywords ?? [];
        return (
          keywords.some((keyword) => /app|application|connector/i.test(keyword)) ||
          Boolean(item.plugin.manifest.hooks?.length)
        );
      }),
    [plugins],
  );
  const pluginSkillTools = useMemo(
    () => visibleTools.filter((tool) => tool.source === "plugin"),
    [visibleTools],
  );
  const visiblePlugins = useMemo(
    () =>
      filterPlugins(
        section === "apps" ? applicationPlugins : plugins,
        query,
      ),
    [applicationPlugins, plugins, query, section],
  );
  const visibleServers = useMemo(
    () => filterServers(servers, query),
    [query, servers],
  );
  const visibleSkillTools = useMemo(
    () => filterTools(pluginSkillTools, query),
    [pluginSkillTools, query],
  );
  const visibleAllTools = useMemo(
    () => filterTools(visibleTools, query),
    [query, visibleTools],
  );
  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    const results = await Promise.allSettled([
      listPlugins(true),
      listMCPServers(true, true),
      listToolCatalog(workspaceRoot ?? ""),
    ] as const);

    if (results[0].status === "fulfilled") {
      setPlugins(results[0].value);
    }
    if (results[1].status === "fulfilled") {
      setServers(results[1].value);
    }
    if (results[2].status === "fulfilled") {
      setTools(results[2].value);
    }

    const failures = results
      .map((result, index) => {
        if (result.status === "fulfilled") return "";
        const label = index === 0 ? "Plugins" : index === 1 ? "MCP" : "Tools";
        const reason = result.reason;
        return `${label}: ${reason instanceof Error ? reason.message : String(reason)}`;
      })
      .filter(Boolean);

    setError(failures.join("\n"));
    setLoading(false);
  }, [workspaceRoot]);

  useEffect(() => {
    if (!active) return;
    void reload();
  }, [active, reload]);

  async function installPluginPath(path: string) {
    if (!path.trim()) return;
    setLoading(true);
    setError("");
    try {
      await installPluginFromPath(path.trim(), true);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      setLoading(false);
    }
  }

  async function installPlugin() {
    if (!pluginPath.trim()) return;
    await installPluginPath(pluginPath);
    setPluginPath("");
  }

  async function addMcpServer(server: MCPServerConfig) {
    await addMcpServers([server], true);
  }

  async function addMcpServers(serversToAdd: MCPServerConfig[], failOnProbeError = false) {
    setLoading(true);
    setError("");
    const failures: string[] = [];
    let savedCount = 0;
    try {
      for (const server of serversToAdd) {
        const normalized = normalizeMcpDraft(server);
        const label = normalized.displayName || normalized.name || normalized.id || "MCP server";
        try {
          const saved = await saveMCPServer(normalized);
          savedCount += 1;
          if (normalized.enabled) {
            try {
              await probeMCPServer(saved.id || normalized.id);
            } catch (err) {
              const message = err instanceof Error ? err.message : String(err);
              failures.push(`${label}: ${message}`);
              if (failOnProbeError) {
                throw err;
              }
            }
          }
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          failures.push(`${label}: ${message}`);
          if (failOnProbeError) {
            throw err;
          }
        }
      }
      await reload();
      if (failures.length > 0) {
        const message = `部分 MCP 已保存，但存在问题：\n${failures.join("\n")}`;
        setError(message);
        if (savedCount === 0) {
          throw new Error(message);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      throw err;
    } finally {
      setLoading(false);
    }
  }

  function openAddDialog() {
    setAddMode(addToolModeForSection(section));
    setAddOpen(true);
  }

  return (
    <section className={cn("flex h-full min-h-0 flex-col overflow-hidden", className)}>
      <AddToolDialog
        loading={loading}
        mode={addMode}
        onInstallPlugin={installPluginPath}
        onModeChange={setAddMode}
        onOpenChange={setAddOpen}
        onSaveMCPServer={addMcpServer}
        onSaveMCPServers={addMcpServers}
        open={addOpen}
      />
      {surface === "dialog" ? (
        <DialogHeader className="border-b px-5 py-4">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Plug />
            Plugins & MCP
          </DialogTitle>
        </DialogHeader>
      ) : null}

      <Tabs
        value={section}
        onValueChange={(value) => setSection(value as PluginSettingsSection)}
        className="min-h-0 flex-1 gap-0"
      >
        <div className="flex items-center justify-between gap-4 border-b p-4">
          <TabsList>
            <TabsTrigger value="plugins">
              插件 <span>{plugins.length}</span>
            </TabsTrigger>
            <TabsTrigger value="apps">
              应用 <span>{applicationPlugins.length}</span>
            </TabsTrigger>
            <TabsTrigger value="mcp">
              MCP <span>{servers.length}</span>
            </TabsTrigger>
            <TabsTrigger value="skills">
              技能 <span>{pluginSkillTools.length}</span>
            </TabsTrigger>
            <TabsTrigger value="tools">
              工具 <span>{visibleTools.length}</span>
            </TabsTrigger>
          </TabsList>
          <div className="flex min-w-0 items-center gap-2">
            <InputGroup className="max-w-sm">
              <InputGroupAddon>
                <Search />
              </InputGroupAddon>
              <InputGroupInput
                aria-label="搜索插件"
                placeholder="搜索插件"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
            </InputGroup>
            <Button
              aria-label={addButtonLabel(section)}
              onClick={openAddDialog}
              size="icon"
              title={addButtonLabel(section)}
              type="button"
              variant="ghost"
            >
              <Plus />
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button size="icon" variant="ghost" aria-label="插件操作">
                  <MoreHorizontal />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  <DropdownMenuItem onSelect={() => void reload()} disabled={loading}>
                    <RefreshCw />
                    刷新
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onSelect={() => void reloadPlugins().then(reload)}
                    disabled={loading}
                  >
                    <Plug />
                    重载插件
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        {error ? (
          <div className="border-b p-4">
            <Alert variant="destructive">
              <TriangleAlert />
              <AlertTitle>加载失败</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </div>
        ) : null}

        <TabsContent value="plugins" className="min-h-0 p-0">
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-4 p-4">
              <div className="flex gap-2">
                <Input
                  placeholder="/path/to/plugin"
                  value={pluginPath}
                  onChange={(event) => setPluginPath(event.target.value)}
                />
                <Button onClick={installPlugin} disabled={loading || !pluginPath.trim()}>
                  安装
                </Button>
              </div>
              <PluginItemGroup
                emptyLabel={query ? "没有匹配的插件" : "没有已安装插件"}
                items={visiblePlugins}
                onReload={reload}
              />
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="apps" className="min-h-0 p-0">
          <ScrollArea className="h-full">
            <div className="p-4">
              <PluginItemGroup
                emptyLabel={query ? "没有匹配的应用" : "没有应用插件"}
                items={visiblePlugins}
                onReload={reload}
              />
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="mcp" className="min-h-0 p-0">
          <ScrollArea className="h-full">
            <div className="p-4">
              {visibleServers.length === 0 ? (
                <EmptyState label={query ? "没有匹配的 MCP server" : "没有 MCP server"} />
              ) : (
                <ItemGroup className="gap-3">
                  {visibleServers.map((item) => (
                    <McpRow key={item.server.id} item={item} onReload={reload} sessionId={sessionId} />
                  ))}
                </ItemGroup>
              )}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="skills" className="min-h-0 p-0">
          <ScrollArea className="h-full">
            <div className="p-4">
              <ToolItemGroup
                emptyLabel={query ? "没有匹配的技能" : "没有可显示技能"}
                tools={visibleSkillTools}
              />
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="tools" className="min-h-0 p-0">
          <ScrollArea className="h-full">
            <div className="p-4">
              <ToolItemGroup
                emptyLabel={query ? "没有匹配的工具" : "没有可显示工具"}
                tools={visibleAllTools}
              />
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </section>
  );
}

function PluginItemGroup({
  emptyLabel,
  items,
  onReload,
}: {
  emptyLabel: string;
  items: PluginListItem[];
  onReload: () => Promise<void>;
}) {
  if (items.length === 0) {
    return <EmptyState label={emptyLabel} />;
  }
  return (
    <ItemGroup>
      {items.map((item) => (
        <PluginRow key={item.plugin.id} item={item} onReload={onReload} />
      ))}
    </ItemGroup>
  );
}

function ToolItemGroup({
  emptyLabel,
  tools,
}: {
  emptyLabel: string;
  tools: ToolCatalogEntry[];
}) {
  if (tools.length === 0) {
    return <EmptyState label={emptyLabel} />;
  }
  return (
    <ItemGroup>
      {tools.map((tool) => (
        <Item key={`${tool.source}:${tool.sourceId ?? ""}:${tool.registrationId ?? ""}:${tool.name}`}>
          <ItemMedia variant="icon">
            <Wrench />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>{tool.name}</ItemTitle>
            <ItemDescription>{tool.description || tool.capability || tool.namespace}</ItemDescription>
          </ItemContent>
          <ItemActions>
            <Badge variant="outline">{tool.source}</Badge>
            {tool.enabled ? <CheckCircle2 /> : null}
          </ItemActions>
        </Item>
      ))}
    </ItemGroup>
  );
}

function AddToolDialog({
  loading,
  mode,
  onInstallPlugin,
  onModeChange,
  onOpenChange,
  onSaveMCPServer,
  onSaveMCPServers,
  open,
}: {
  loading: boolean;
  mode: AddToolMode;
  onInstallPlugin: (path: string) => Promise<void>;
  onModeChange: (mode: AddToolMode) => void;
  onOpenChange: (open: boolean) => void;
  onSaveMCPServer: (server: MCPServerConfig) => Promise<void>;
  onSaveMCPServers: (servers: MCPServerConfig[]) => Promise<void>;
  open: boolean;
}) {
  const [pluginPath, setPluginPath] = useState("");
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
  const canSubmit =
    mode === "plugin"
      ? Boolean(pluginPath.trim())
      : mcpInputMode === "json"
        ? Boolean(mcpImportText.trim())
        : canSaveMcpDraft(preparedMcpDraft);

  async function submit() {
    if (!canSubmit) return;
    setLocalError("");
    try {
      if (mode === "plugin") {
        await onInstallPlugin(pluginPath);
        setPluginPath("");
      } else if (mcpInputMode === "json") {
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
      <DialogContent
        className={cn(
          "flex flex-col overflow-hidden sm:max-w-2xl",
          mode === "mcp" ? "h-[min(760px,85vh)]" : "max-h-[85vh]",
        )}
      >
        <DialogHeader>
          <DialogTitle>添加工具</DialogTitle>
          <DialogDescription>
            通过安装插件或添加 MCP server，把对应工具加入当前工具目录。
          </DialogDescription>
        </DialogHeader>

        {mode === "plugin" ? (
          <div className="grid gap-2">
            <Label>插件路径</Label>
            <Input
              autoFocus
              placeholder="/path/to/plugin"
              value={pluginPath}
              onChange={(event) => setPluginPath(event.target.value)}
            />
          </div>
        ) : (
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
                <div className="grid gap-4">
                  <div className="grid gap-3 rounded-md border p-3">
                    <McpField label="服务器 ID">
                      <Input
                        placeholder="filesystem"
                        value={draft.id}
                        onChange={(event) => setDraft({ ...draft, id: event.target.value })}
                      />
                    </McpField>
                    <McpField label="显示名称">
                      <Input
                        placeholder="Filesystem"
                        value={draft.displayName ?? ""}
                        onChange={(event) => setDraft({ ...draft, displayName: event.target.value })}
                      />
                    </McpField>
                    <McpField label="服务器类型">
                      <Select
                        value={draft.transport}
                        onValueChange={(transport) =>
                          setDraft({
                            ...draft,
                            transport: transport as MCPServerConfig["transport"],
                          })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="服务器类型" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="stdio">stdio</SelectItem>
                            <SelectItem value="streamable_http">Streamable HTTP</SelectItem>
                            <SelectItem value="sse">SSE</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </McpField>
                  </div>

                  {draft.transport === "stdio" ? (
                    <div className="grid gap-3 rounded-md border p-3">
                      <McpField label="启动命令">
                        <Input
                          placeholder="npx"
                          value={draft.command ?? ""}
                          onChange={(event) => setDraft({ ...draft, command: event.target.value })}
                        />
                      </McpField>
                      <McpStringRows
                        addLabel="添加参数"
                        label="参数"
                        onChange={setArgRows}
                        placeholder="参数"
                        rows={argRows}
                      />
                    </div>
                  ) : (
                    <div className="grid gap-3 rounded-md border p-3">
                      <McpField label="Server URL">
                        <Input
                          placeholder="https://example.com/mcp"
                          value={draft.url ?? ""}
                          onChange={(event) => setDraft({ ...draft, url: event.target.value })}
                        />
                      </McpField>
                      <McpField label="认证方式">
                        <Select
                          value={draft.authType ?? "none"}
                          onValueChange={(authType) =>
                            setDraft({
                              ...draft,
                              authType: authType as MCPServerConfig["authType"],
                            })
                          }
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="认证方式" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              <SelectItem value="none">No auth</SelectItem>
                              <SelectItem value="bearer">Bearer env</SelectItem>
                              <SelectItem value="oauth">OAuth bearer env</SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      </McpField>
                      {draft.authType === "bearer" || draft.authType === "oauth" ? (
                        <McpField label="Token 环境变量">
                          <Input
                            placeholder="MCP_TOKEN"
                            value={draft.bearerTokenEnv ?? ""}
                            onChange={(event) =>
                              setDraft({ ...draft, bearerTokenEnv: event.target.value })
                            }
                          />
                        </McpField>
                      ) : null}
                      {draft.authType === "oauth" ? (
                        <>
                          <McpField label="OAuth issuer URL">
                            <Input
                              value={draft.oauthIssuerUrl ?? ""}
                              onChange={(event) =>
                                setDraft({ ...draft, oauthIssuerUrl: event.target.value })
                              }
                            />
                          </McpField>
                          <McpField label="OAuth client id">
                            <Input
                              value={draft.oauthClientId ?? ""}
                              onChange={(event) =>
                                setDraft({ ...draft, oauthClientId: event.target.value })
                              }
                            />
                          </McpField>
                          <McpField label="OAuth scopes">
                            <Input
                              value={(draft.oauthScopes ?? []).join(" ")}
                              onChange={(event) =>
                                setDraft({ ...draft, oauthScopes: parseWords(event.target.value) })
                              }
                            />
                          </McpField>
                        </>
                      ) : null}
                      <McpKeyValueRows
                        addLabel="添加请求头"
                        label="请求头"
                        leftPlaceholder="键"
                        onChange={setHeaderRows}
                        rightPlaceholder="值"
                        rows={headerRows}
                      />
                    </div>
                  )}

                  <div className="grid gap-3 rounded-md border p-3">
                    <McpKeyValueRows
                      addLabel="添加环境变量"
                      label="环境变量"
                      leftPlaceholder="键"
                      onChange={setEnvRows}
                      rightPlaceholder="值"
                      rows={envRows}
                    />
                    <McpStringRows
                      addLabel="添加变量"
                      label="环境变量传递"
                      onChange={setRootRows}
                      placeholder="变量"
                      rows={rootRows}
                    />
                    <McpField label="工作目录">
                      <Input
                        placeholder="~/code"
                        value={draft.cwd ?? ""}
                        onChange={(event) => setDraft({ ...draft, cwd: event.target.value })}
                      />
                    </McpField>
                    <div className="flex items-center justify-between gap-3">
                      <Label>保存后启用并探测</Label>
                      <Switch
                        checked={draft.enabled}
                        onCheckedChange={(enabled) => setDraft({ ...draft, enabled })}
                      />
                    </div>
                  </div>
                </div>
              </TabsContent>
            </Tabs>
          </ScrollArea>
        )}

        {localError ? <div className="text-xs text-destructive">{localError}</div> : null}

        <DialogFooter className="flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex gap-2">
            <Button
              onClick={() => onModeChange("plugin")}
              type="button"
              variant={mode === "plugin" ? "secondary" : "outline"}
            >
              <Plug />
              插件
            </Button>
            <Button
              onClick={() => {
                onModeChange("mcp");
                setMcpInputMode("json");
              }}
              type="button"
              variant={mode === "mcp" ? "secondary" : "outline"}
            >
              <Server />
              MCP
            </Button>
          </div>
          <div className="flex justify-end gap-2">
            <DialogClose asChild>
              <Button type="button" variant="outline">
                取消
              </Button>
            </DialogClose>
            <Button disabled={loading || !canSubmit} onClick={() => void submit()} type="button">
              {loading ? "处理中" : mode === "mcp" && mcpInputMode === "json" ? "导入" : "添加"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PluginRow({ item, onReload }: { item: PluginListItem; onReload: () => Promise<void> }) {
  const plugin = item.plugin;
  const tools = pluginToolsForDisplay(item);
  const Icon = pluginIcon(plugin.manifest);
  return (
    <Item variant={plugin.error ? "outline" : "default"}>
      <ItemMedia variant="icon">
        <Icon />
      </ItemMedia>
      <ItemContent>
        <ItemTitle>
          {plugin.manifest.displayName || plugin.manifest.name}
          <StatusBadge status={plugin.status} />
        </ItemTitle>
        <ItemDescription>
          {plugin.manifest.description || plugin.rootPath}
        </ItemDescription>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{tools.length} tools</Badge>
          <Badge variant="outline">{plugin.manifest.hooks?.length ?? 0} hooks</Badge>
          {plugin.error ? <Badge variant="destructive">{plugin.error}</Badge> : null}
        </div>
      </ItemContent>
      <ItemActions>
        <Switch
          checked={plugin.enabled}
          onCheckedChange={(enabled) =>
            void setPluginEnabled(plugin.id, enabled).then(onReload)
          }
        />
      </ItemActions>
    </Item>
  );
}

function McpRow({
  item,
  onReload,
  sessionId,
}: {
  item: MCPServerListItem;
  onReload: () => Promise<void>;
  sessionId?: string;
}) {
  const server = item.server;
  const tools = item.tools ?? [];
  const prompts = item.prompts ?? [];
  const resources = item.resources ?? [];
  const templates = item.resourceTemplates ?? [];
  const [log, setLog] = useState("");
  const [logError, setLogError] = useState("");
  const [loadingLog, setLoadingLog] = useState(false);
  const [oauthDiscovery, setOauthDiscovery] = useState<MCPOAuthDiscoveryResult | null>(null);
  const [oauthStart, setOauthStart] = useState<MCPOAuthStartResult | null>(null);
  const [oauthStatus, setOauthStatus] = useState<MCPOAuthStatus | null>(null);
  const [oauthError, setOauthError] = useState("");
  const [loadingOAuth, setLoadingOAuth] = useState(false);
  const [promptInputs, setPromptInputs] = useState<Record<string, Record<string, string>>>({});
  const [promptResult, setPromptResult] = useState<MCPPromptGetResult | null>(null);
  const [promptError, setPromptError] = useState("");
  const [loadingPromptId, setLoadingPromptId] = useState("");
  const [insertingPromptId, setInsertingPromptId] = useState("");
  const [templateInputs, setTemplateInputs] = useState<Record<string, Record<string, string>>>({});
  const [resourceResult, setResourceResult] = useState<MCPResourceReadResult | null>(null);
  const [resourceError, setResourceError] = useState("");
  const [loadingResourceId, setLoadingResourceId] = useState("");
  const [insertingResourceId, setInsertingResourceId] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [draft, setDraft] = useState<MCPServerConfig>(() => mcpServerToDraft(server));
  const [argRows, setArgRows] = useState<string[]>(() => nonEmptyStrings(server.args));
  const [envRows, setEnvRows] = useState<KeyValueRow[]>(() => mapToRows(server.env));
  const [headerRows, setHeaderRows] = useState<KeyValueRow[]>(() => mapToRows(server.headers));
  const [rootRows, setRootRows] = useState<string[]>(() => nonEmptyStrings(server.roots));

  useEffect(() => {
    if (!settingsOpen) return;
    setDraft(mcpServerToDraft(server));
    setArgRows(nonEmptyStrings(server.args));
    setEnvRows(mapToRows(server.env));
    setHeaderRows(mapToRows(server.headers));
    setRootRows(nonEmptyStrings(server.roots));
    setSaveError("");
  }, [server, settingsOpen]);

  async function saveSettings() {
    setSaving(true);
    setSaveError("");
    try {
      await saveMCPServer(
        normalizeMcpDraft({
          ...draft,
          args: compactStrings(argRows),
          env: rowsToMap(envRows),
          headers: rowsToMap(headerRows),
          roots: compactStrings(rootRows),
        }),
      );
      await onReload();
      setSettingsOpen(false);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function loadLog() {
    setLoadingLog(true);
    setLogError("");
    try {
      const result = await readMCPServerLog(server.id);
      setLog(result.content || "没有日志输出");
    } catch (err) {
      setLogError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingLog(false);
    }
  }

  async function loadOAuthDiscovery() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      setOauthDiscovery(await discoverMCPOAuth(server.id));
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function connectOAuth() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      const result = await startMCPOAuth(server.id);
      setOauthStart(result);
      if (result.url) {
        window.open(result.url, "_blank", "noopener,noreferrer");
      }
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function refreshOAuthStatus() {
    setLoadingOAuth(true);
    setOauthError("");
    try {
      setOauthStatus(await getMCPOAuthStatus(server.id));
    } catch (err) {
      setOauthError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingOAuth(false);
    }
  }

  async function loadPrompt(prompt: MCPPromptRecord) {
    setLoadingPromptId(prompt.id);
    setPromptError("");
    try {
      const values = promptInputs[prompt.id] ?? {};
      const args = Object.fromEntries(
        (prompt.arguments ?? []).map((argument) => [argument.name, values[argument.name] ?? ""]),
      );
      setPromptResult(await getMCPPrompt(server.id, prompt.name, args));
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingPromptId("");
    }
  }

  async function insertPrompt(prompt: MCPPromptRecord) {
    if (!sessionId) return;
    setInsertingPromptId(prompt.id);
    setPromptError("");
    try {
      const values = promptInputs[prompt.id] ?? {};
      const args = Object.fromEntries(
        (prompt.arguments ?? []).map((argument) => [argument.name, values[argument.name] ?? ""]),
      );
      await insertMCPPromptIntoSession(sessionId, server.id, prompt.name, args);
      setPromptResult({
        serverId: server.id,
        name: prompt.name,
        content: "已插入当前会话",
      });
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : String(err));
    } finally {
      setInsertingPromptId("");
    }
  }

  async function loadResource(resource: MCPResourceRecord, uri: string) {
    setLoadingResourceId(resource.id);
    setResourceError("");
    try {
      setResourceResult(await readMCPResource(server.id, uri));
    } catch (err) {
      setResourceError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingResourceId("");
    }
  }

  async function insertResource(resource: MCPResourceRecord, uri: string) {
    if (!sessionId) return;
    setInsertingResourceId(resource.id);
    setResourceError("");
    try {
      await insertMCPResourceIntoSession(sessionId, server.id, uri);
      setResourceResult({
        serverId: server.id,
        uri,
        content: "已插入当前会话",
      });
    } catch (err) {
      setResourceError(err instanceof Error ? err.message : String(err));
    } finally {
      setInsertingResourceId("");
    }
  }

  return (
    <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
      <Item>
        <ItemMedia variant="image" className="bg-muted text-muted-foreground">
          <Plug />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>{server.displayName || server.name || server.id}</ItemTitle>
        </ItemContent>
        <ItemActions>
          <Button
            aria-label={`设置 ${server.displayName || server.name || server.id}`}
            onClick={() => setSettingsOpen(true)}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Settings />
          </Button>
          <Switch
            checked={server.enabled}
            onCheckedChange={(enabled) =>
              void setMCPServerEnabled(server.id, enabled).then(onReload)
            }
          />
        </ItemActions>
      </Item>
      <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{server.displayName || server.name || server.id}</DialogTitle>
          <DialogDescription>
            如需切换 MCP 服务器类型，请先卸载当前配置。
          </DialogDescription>
        </DialogHeader>
        <ScrollArea className="min-h-0 flex-1 pr-3">
          <div className="grid gap-4">
            <div className="grid gap-3 rounded-md border p-3">
              <McpField label="服务器 ID">
                <Input
                  value={draft.id}
                  onChange={(event) => setDraft({ ...draft, id: event.target.value })}
                />
              </McpField>
              <McpField label="显示名称">
                <Input
                  value={draft.displayName ?? ""}
                  onChange={(event) => setDraft({ ...draft, displayName: event.target.value })}
                />
              </McpField>
              <McpField label="服务器类型">
                <Input value={mcpTransportLabel(draft.transport)} disabled />
              </McpField>
            </div>

            {draft.transport === "stdio" ? (
              <div className="grid gap-3 rounded-md border p-3">
                <McpField label="启动命令">
                  <Input
                    value={draft.command ?? ""}
                    onChange={(event) => setDraft({ ...draft, command: event.target.value })}
                  />
                </McpField>
                <McpStringRows
                  addLabel="添加参数"
                  label="参数"
                  onChange={setArgRows}
                  placeholder="参数"
                  rows={argRows}
                />
              </div>
            ) : (
              <div className="grid gap-3 rounded-md border p-3">
                <McpField label="Server URL">
                  <Input
                    value={draft.url ?? ""}
                    onChange={(event) => setDraft({ ...draft, url: event.target.value })}
                  />
                </McpField>
                <McpField label="认证方式">
                  <Select
                    value={draft.authType ?? "none"}
                    onValueChange={(authType) =>
                      setDraft({
                        ...draft,
                        authType: authType as MCPServerConfig["authType"],
                      })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="认证方式" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="none">No auth</SelectItem>
                        <SelectItem value="bearer">Bearer env</SelectItem>
                        <SelectItem value="oauth">OAuth bearer env</SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </McpField>
                {draft.authType === "bearer" || draft.authType === "oauth" ? (
                  <McpField label="Token 环境变量">
                    <Input
                      value={draft.bearerTokenEnv ?? ""}
                      onChange={(event) =>
                        setDraft({ ...draft, bearerTokenEnv: event.target.value })
                      }
                    />
                  </McpField>
                ) : null}
                {draft.authType === "oauth" ? (
                  <>
                    <McpField label="OAuth issuer URL">
                      <Input
                        value={draft.oauthIssuerUrl ?? ""}
                        onChange={(event) =>
                          setDraft({ ...draft, oauthIssuerUrl: event.target.value })
                        }
                      />
                    </McpField>
                    <McpField label="OAuth client id">
                      <Input
                        value={draft.oauthClientId ?? ""}
                        onChange={(event) =>
                          setDraft({ ...draft, oauthClientId: event.target.value })
                        }
                      />
                    </McpField>
                    <McpField label="OAuth scopes">
                      <Input
                        value={(draft.oauthScopes ?? []).join(" ")}
                        onChange={(event) =>
                          setDraft({ ...draft, oauthScopes: parseWords(event.target.value) })
                        }
                      />
                    </McpField>
                  </>
                ) : null}
                <McpKeyValueRows
                  addLabel="添加请求头"
                  label="请求头"
                  leftPlaceholder="键"
                  onChange={setHeaderRows}
                  rightPlaceholder="值"
                  rows={headerRows}
                />
              </div>
            )}

            <div className="grid gap-3 rounded-md border p-3">
              <McpKeyValueRows
                addLabel="添加环境变量"
                label="环境变量"
                leftPlaceholder="键"
                onChange={setEnvRows}
                rightPlaceholder="值"
                rows={envRows}
              />
              <McpStringRows
                addLabel="添加变量"
                label="环境变量传递"
                onChange={setRootRows}
                placeholder="变量"
                rows={rootRows}
              />
              <McpField label="工作目录">
                <Input
                  placeholder="~/code"
                  value={draft.cwd ?? ""}
                  onChange={(event) => setDraft({ ...draft, cwd: event.target.value })}
                />
              </McpField>
            </div>

            <div className="grid gap-3 rounded-md border p-3">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge status={server.status || (server.enabled ? "enabled" : "disabled")} />
                <Badge variant="secondary">{tools.length} tools</Badge>
                <Badge variant="outline">{prompts.length} prompts</Badge>
                <Badge variant="outline">{resources.length} resources</Badge>
                <Badge variant="outline">{templates.length} templates</Badge>
              </div>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="outline" onClick={loadLog} disabled={loadingLog}>
                  日志
                </Button>
                {server.authType === "oauth" && server.transport !== "stdio" ? (
                  <>
                    <Button size="sm" variant="outline" onClick={loadOAuthDiscovery} disabled={loadingOAuth}>
                      OAuth 信息
                    </Button>
                    <Button size="sm" variant="outline" onClick={connectOAuth} disabled={loadingOAuth}>
                      连接
                    </Button>
                    <Button size="sm" variant="outline" onClick={refreshOAuthStatus} disabled={loadingOAuth}>
                      状态
                    </Button>
                  </>
                ) : null}
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => void probeMCPServer(server.id).then(onReload)}
                >
                  探测
                </Button>
              </div>
              {tools.length > 0 ? (
                <div className="grid gap-1 text-xs text-muted-foreground">
                  {tools.slice(0, 3).map((tool) => (
                    <CapabilityLine key={tool.id} label="tool" name={tool.name} detail={tool.description} />
                  ))}
                </div>
              ) : null}
            </div>

            {prompts.length > 0 ? (
              <div className="grid gap-3 rounded-md border p-3">
                <SectionHeading icon={MessageSquareText} label="Prompts" />
                <div className="grid gap-2">
                  {prompts.map((prompt) => (
                    <PromptActionLine
                      key={prompt.id}
                      prompt={prompt}
                      inputs={promptInputs[prompt.id] ?? {}}
                      loading={loadingPromptId === prompt.id}
                      inserting={insertingPromptId === prompt.id}
                      onInputChange={(name, value) =>
                        setPromptInputs((current) => ({
                          ...current,
                          [prompt.id]: { ...(current[prompt.id] ?? {}), [name]: value },
                        }))
                      }
                      onInsert={sessionId ? () => void insertPrompt(prompt) : undefined}
                      onRun={() => void loadPrompt(prompt)}
                    />
                  ))}
                </div>
              </div>
            ) : null}

            {resources.length + templates.length > 0 ? (
              <div className="grid gap-3 rounded-md border p-3">
                <SectionHeading icon={FileText} label="Resources" />
                <div className="grid gap-2">
                  {resources.map((resource) => (
                    <ResourceActionLine
                      key={resource.id}
                      resource={resource}
                      loading={loadingResourceId === resource.id}
                      inserting={insertingResourceId === resource.id}
                      onInsert={sessionId ? (uri) => void insertResource(resource, uri) : undefined}
                      onRead={(uri) => void loadResource(resource, uri)}
                    />
                  ))}
                  {templates.map((template) => (
                    <ResourceActionLine
                      key={template.id}
                      resource={template}
                      inputs={templateInputs[template.id] ?? {}}
                      loading={loadingResourceId === template.id}
                      inserting={insertingResourceId === template.id}
                      onInputChange={(name, value) =>
                        setTemplateInputs((current) => ({
                          ...current,
                          [template.id]: { ...(current[template.id] ?? {}), [name]: value },
                        }))
                      }
                      onInsert={sessionId ? (uri) => void insertResource(template, uri) : undefined}
                      onRead={(uri) => void loadResource(template, uri)}
                    />
                  ))}
                </div>
              </div>
            ) : null}

            {saveError ? <div className="text-xs text-destructive">{saveError}</div> : null}
            {promptError ? <div className="text-xs text-destructive">{promptError}</div> : null}
            {promptResult ? (
              <PreviewBlock
                label={`Prompt: ${promptResult.name}`}
                value={promptResult.content || stringifyStructured(promptResult.structured)}
              />
            ) : null}
            {resourceError ? <div className="text-xs text-destructive">{resourceError}</div> : null}
            {resourceResult ? (
              <PreviewBlock
                label={`Resource: ${resourceResult.uri}`}
                value={resourcePreview(resourceResult)}
              />
            ) : null}
            {oauthError ? <div className="text-xs text-destructive">{oauthError}</div> : null}
            {oauthDiscovery ? (
              <PreviewBlock label="OAuth discovery" value={oauthDiscoveryPreview(oauthDiscovery)} />
            ) : null}
            {oauthStart ? (
              <PreviewBlock label="OAuth authorization" value={oauthStartPreview(oauthStart)} />
            ) : null}
            {oauthStatus ? (
              <PreviewBlock label="OAuth status" value={oauthStatusPreview(oauthStatus)} />
            ) : null}
            {logError ? <div className="text-xs text-destructive">{logError}</div> : null}
            {log ? (
              <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
                {log}
              </pre>
            ) : null}
            {server.error ? <div className="text-xs text-destructive">{server.error}</div> : null}
          </div>
        </ScrollArea>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline">
              取消
            </Button>
          </DialogClose>
          <Button
            disabled={saving || !canSaveMcpDraft(draft)}
            onClick={() => void saveSettings()}
            type="button"
          >
            {saving ? "保存中" : "保存"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type KeyValueRow = {
  key: string;
  value: string;
};

function McpField({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function McpStringRows({
  addLabel,
  label,
  onChange,
  placeholder,
  rows,
}: {
  addLabel: string;
  label: string;
  onChange: (rows: string[]) => void;
  placeholder: string;
  rows: string[];
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {rows.map((row, index) => (
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2" key={index}>
          <Input
            placeholder={placeholder}
            value={row}
            onChange={(event) =>
              onChange(rows.map((item, itemIndex) => (itemIndex === index ? event.target.value : item)))
            }
          />
          <Button
            aria-label={`删除${label}`}
            disabled={rows.length === 1}
            onClick={() => onChange(rows.filter((_item, itemIndex) => itemIndex !== index))}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 />
          </Button>
        </div>
      ))}
      <Button
        onClick={() => onChange([...rows, ""])}
        type="button"
        variant="secondary"
      >
        <Plus />
        {addLabel}
      </Button>
    </div>
  );
}

function McpKeyValueRows({
  addLabel,
  label,
  leftPlaceholder,
  onChange,
  rightPlaceholder,
  rows,
}: {
  addLabel: string;
  label: string;
  leftPlaceholder: string;
  onChange: (rows: KeyValueRow[]) => void;
  rightPlaceholder: string;
  rows: KeyValueRow[];
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {rows.map((row, index) => (
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-2" key={index}>
          <Input
            placeholder={leftPlaceholder}
            value={row.key}
            onChange={(event) =>
              onChange(
                rows.map((item, itemIndex) =>
                  itemIndex === index ? { ...item, key: event.target.value } : item,
                ),
              )
            }
          />
          <Input
            placeholder={rightPlaceholder}
            value={row.value}
            onChange={(event) =>
              onChange(
                rows.map((item, itemIndex) =>
                  itemIndex === index ? { ...item, value: event.target.value } : item,
                ),
              )
            }
          />
          <Button
            aria-label={`删除${label}`}
            disabled={rows.length === 1}
            onClick={() => onChange(rows.filter((_item, itemIndex) => itemIndex !== index))}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Trash2 />
          </Button>
        </div>
      ))}
      <Button
        onClick={() => onChange([...rows, { key: "", value: "" }])}
        type="button"
        variant="secondary"
      >
        <Plus />
        {addLabel}
      </Button>
    </div>
  );
}

function SectionHeading({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <div className="flex items-center gap-2 text-xs font-medium text-foreground">
      <Icon />
      <span>{label}</span>
    </div>
  );
}

function PromptActionLine({
  prompt,
  inputs,
  loading,
  inserting,
  onInputChange,
  onInsert,
  onRun,
}: {
  prompt: MCPPromptRecord;
  inputs: Record<string, string>;
  loading: boolean;
  inserting?: boolean;
  onInputChange: (name: string, value: string) => void;
  onInsert?: () => void;
  onRun: () => void;
}) {
  const missingRequired = (prompt.arguments ?? []).some(
    (argument) => argument.required && !inputs[argument.name]?.trim(),
  );
  return (
    <div className="grid gap-2 text-xs">
      <CapabilityLine label="prompt" name={prompt.name} detail={prompt.description} />
      {prompt.arguments?.length ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {prompt.arguments.map((argument) => (
            <Input
              key={argument.name}
              className="h-8 text-xs"
              placeholder={`${argument.name}${argument.required ? " *" : ""}`}
              title={argument.description || argument.name}
              value={inputs[argument.name] ?? ""}
              onChange={(event) => onInputChange(argument.name, event.target.value)}
            />
          ))}
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button size="sm" variant="outline" onClick={onRun} disabled={loading || missingRequired}>
          {loading ? "读取中" : "预览 prompt"}
        </Button>
        {onInsert ? (
          <Button size="sm" variant="outline" onClick={onInsert} disabled={inserting || missingRequired}>
            {inserting ? "插入中" : "插入会话"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function ResourceActionLine({
  resource,
  inputs = {},
  loading,
  inserting,
  onInputChange,
  onInsert,
  onRead,
}: {
  resource: MCPResourceRecord;
  inputs?: Record<string, string>;
  loading: boolean;
  inserting?: boolean;
  onInputChange?: (name: string, value: string) => void;
  onInsert?: (uri: string) => void;
  onRead: (uri: string) => void;
}) {
  const template = resource.uriTemplate ?? "";
  const variables = templateVariables(template);
  const resolvedURI = resource.template ? applySimpleTemplate(template, inputs) : (resource.uri ?? "");
  const missingRequired = resource.template && variables.some((name) => !inputs[name]?.trim());
  return (
    <div className="grid gap-2 text-xs">
      <CapabilityLine
        label={resource.template ? "template" : "resource"}
        name={resource.name}
        detail={resource.uri || resource.uriTemplate || resource.description}
      />
      {resource.template && variables.length > 0 ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {variables.map((name) => (
            <Input
              key={name}
              className="h-8 text-xs"
              placeholder={name}
              value={inputs[name] ?? ""}
              onChange={(event) => onInputChange?.(name, event.target.value)}
            />
          ))}
        </div>
      ) : null}
      <div className="flex min-w-0 items-center gap-2">
        <Button size="sm" variant="outline" onClick={() => onRead(resolvedURI)} disabled={loading || !resolvedURI || missingRequired}>
          {loading ? "读取中" : "读取"}
        </Button>
        {onInsert ? (
          <Button size="sm" variant="outline" onClick={() => onInsert(resolvedURI)} disabled={inserting || !resolvedURI || missingRequired}>
            {inserting ? "插入中" : "插入会话"}
          </Button>
        ) : null}
        {resource.template ? <span className="min-w-0 truncate text-muted-foreground">{resolvedURI}</span> : null}
      </div>
    </div>
  );
}

function CapabilityLine({
  label,
  name,
  detail,
}: {
  label: string;
  name: string;
  detail?: string;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wide">
        {label}
      </span>
      <span className="min-w-0 truncate text-foreground">{name}</span>
      {detail ? <span className="min-w-0 truncate">{detail}</span> : null}
    </div>
  );
}

function PreviewBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="mt-3">
      <div className="mb-1 truncate text-xs font-medium text-foreground">{label}</div>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
        {value || "没有可预览内容"}
      </pre>
    </div>
  );
}

function StatusBadge({ status }: { status?: string }) {
  const ok = status === "ready" || status === "enabled";
  return (
    <Badge variant={ok ? "secondary" : "outline"}>
      {ok ? <CheckCircle2 /> : <TriangleAlert />}
      {status || "unknown"}
    </Badge>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <Empty>
      <EmptyMedia variant="icon">
        <Layers3 />
      </EmptyMedia>
      <EmptyHeader>
        <EmptyTitle>{label}</EmptyTitle>
        <EmptyDescription>调整搜索条件或刷新后重试。</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

function addToolModeForSection(section: PluginSettingsSection): AddToolMode {
  if (section === "mcp" || section === "tools") {
    return "mcp";
  }
  return "plugin";
}

function addButtonLabel(section: PluginSettingsSection) {
  if (section === "mcp" || section === "tools") {
    return "添加 MCP 工具";
  }
  return "添加插件工具";
}

function filterPlugins(items: PluginListItem[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((item) => {
    const plugin = item.plugin;
    return matchesSearch(
      [
        plugin.id,
        plugin.manifest.name,
        plugin.manifest.displayName,
        plugin.manifest.description,
        plugin.rootPath,
        ...(plugin.manifest.keywords ?? []),
      ],
      normalized,
    );
  });
}

function filterServers(items: MCPServerListItem[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((item) => {
    const server = item.server;
    return matchesSearch(
      [
        server.id,
        server.name,
        server.displayName,
        server.description,
        server.command,
        server.url,
        server.transport,
      ],
      normalized,
    );
  });
}

function filterTools(items: ToolCatalogEntry[], query: string) {
  const normalized = normalizeSearch(query);
  if (!normalized) {
    return items;
  }
  return items.filter((tool) =>
    matchesSearch(
      [
        tool.name,
        tool.description,
        tool.namespace,
        tool.capability,
        tool.category,
        tool.source,
        ...(tool.toolsets ?? []),
      ],
      normalized,
    ),
  );
}

function matchesSearch(values: Array<string | undefined>, query: string) {
  return values.some((value) => normalizeSearch(value).includes(query));
}

function normalizeSearch(value?: string) {
  return value?.trim().toLowerCase() ?? "";
}

function pluginIcon(manifest: PluginListItem["plugin"]["manifest"]): LucideIcon {
  const text = normalizeSearch(
    [
      manifest.displayName,
      manifest.name,
      manifest.description,
      ...(manifest.keywords ?? []),
    ]
      .filter(Boolean)
      .join(" "),
  );
  if (/mcp|server/.test(text)) {
    return Server;
  }
  if (/skill|tool|template/.test(text)) {
    return Wrench;
  }
  if (/app|browser|chrome|figma|github|notion/.test(text)) {
    return Layers3;
  }
  return Plug;
}

function pluginToolsForDisplay(item: PluginListItem): ToolCatalogEntry[] {
  if (item.tools?.length) {
    return item.tools;
  }
  return (item.plugin.manifest.tools ?? []).map((tool) => ({
    name: tool.name,
    description: tool.description,
    inputSchema: tool.inputSchema,
    namespace: item.plugin.manifest.name || item.plugin.id,
    capability: tool.capability,
    riskLevel: tool.riskLevel,
    category: tool.category,
    toolsets: tool.toolsets,
    source: "plugin",
    sourceId: item.plugin.id,
    enabled: item.plugin.enabled,
  }));
}

function mergeToolCatalogEntries(groups: ToolCatalogEntry[][]) {
  const merged = new Map<string, ToolCatalogEntry>();
  for (const group of groups) {
    for (const tool of group) {
      const key = [
        tool.source,
        tool.sourceId ?? "",
        tool.registrationId ?? "",
        tool.name,
      ].join(":");
      if (!merged.has(key)) {
        merged.set(key, tool);
      }
    }
  }
  return Array.from(merged.values());
}

const MCP_IMPORT_PLACEHOLDER = `{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/project"]
    }
  }
}`;

function parseMcpServersImport(raw: string): MCPServerConfig[] {
  const trimmed = raw.trim();
  if (!trimmed) {
    throw new Error("请先粘贴 MCP JSON 配置");
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`JSON 解析失败：${message}`);
  }

  if (!isPlainRecord(parsed)) {
    throw new Error("MCP JSON 必须是对象");
  }

  const serversValue = isPlainRecord(parsed.mcpServers) ? parsed.mcpServers : parsed;
  const servers = Object.entries(serversValue).map(([id, value]) =>
    mcpServerFromImportEntry(id, value),
  );
  if (servers.length === 0) {
    throw new Error("没有找到可导入的 MCP server");
  }
  return servers;
}

function mcpServerFromImportEntry(id: string, value: unknown): MCPServerConfig {
  if (!isPlainRecord(value)) {
    throw new Error(`${id} 的配置必须是对象`);
  }

  const cleanID = id.trim();
  if (!cleanID) {
    throw new Error("MCP server id 不能为空");
  }

  const command = optionalString(value.command);
  const url = optionalString(value.url);
  const transport =
    normalizeImportedMcpTransport(optionalString(value.transport) || optionalString(value.type)) ||
    (command ? "stdio" : "streamable_http");
  const server: MCPServerConfig = {
    ...emptyMcpServer(),
    id: cleanID,
    name: optionalString(value.name) || cleanID,
    displayName: optionalString(value.displayName),
    description: optionalString(value.description),
    transport,
    command: transport === "stdio" ? command : "",
    args: transport === "stdio" ? optionalStringArray(value.args, `${cleanID}.args`) : [],
    cwd: optionalString(value.cwd),
    env: optionalStringMap(value.env, `${cleanID}.env`),
    url: transport === "stdio" ? "" : url,
    headers: optionalStringMap(value.headers, `${cleanID}.headers`),
    authType: normalizeImportedMcpAuthType(optionalString(value.authType)),
    bearerTokenEnv: optionalString(value.bearerTokenEnv),
    oauthIssuerUrl: optionalString(value.oauthIssuerUrl),
    oauthClientId: optionalString(value.oauthClientId),
    oauthScopes: optionalStringArray(value.oauthScopes, `${cleanID}.oauthScopes`),
    roots: optionalStringArray(value.roots, `${cleanID}.roots`),
    timeoutSeconds: optionalNumber(value.timeoutSeconds),
    connectTimeoutSeconds: optionalNumber(value.connectTimeoutSeconds),
    enabled: optionalBoolean(value.enabled) ?? !optionalBoolean(value.disabled),
  };

  if (server.transport === "stdio" && !server.command?.trim()) {
    throw new Error(`${cleanID} 缺少 command`);
  }
  if (server.transport !== "stdio" && !server.url?.trim()) {
    throw new Error(`${cleanID} 缺少 url`);
  }
  return server;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function optionalString(value: unknown) {
  return typeof value === "string" ? value.trim() : undefined;
}

function optionalBoolean(value: unknown) {
  return typeof value === "boolean" ? value : undefined;
}

function optionalNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function optionalStringArray(value: unknown, label: string) {
  if (value == null) return [];
  if (!Array.isArray(value)) {
    throw new Error(`${label} 必须是字符串数组`);
  }
  return value.map((item, index) => {
    if (typeof item !== "string") {
      throw new Error(`${label}[${index}] 必须是字符串`);
    }
    return item;
  });
}

function optionalStringMap(value: unknown, label: string) {
  if (value == null) return {};
  if (!isPlainRecord(value)) {
    throw new Error(`${label} 必须是对象`);
  }
  const out: Record<string, string> = {};
  for (const [key, item] of Object.entries(value)) {
    if (typeof item !== "string") {
      throw new Error(`${label}.${key} 必须是字符串`);
    }
    out[key] = item;
  }
  return out;
}

function normalizeImportedMcpTransport(value?: string): MCPServerConfig["transport"] | undefined {
  switch (value) {
    case "stdio":
    case "streamable_http":
    case "sse":
      return value;
    case "http":
      return "streamable_http";
    default:
      return undefined;
  }
}

function normalizeImportedMcpAuthType(value?: string): MCPServerConfig["authType"] {
  switch (value) {
    case "bearer":
    case "oauth":
      return value;
    case "none":
    default:
      return "none";
  }
}

function emptyMcpServer(): MCPServerConfig {
  return { id: "", name: "", transport: "stdio", command: "", args: [], roots: [], authType: "none", enabled: false };
}

function mcpServerToDraft(server: MCPServerConfig): MCPServerConfig {
  return {
    ...emptyMcpServer(),
    ...server,
    args: server.args ?? [],
    env: server.env ?? {},
    headers: server.headers ?? {},
    roots: server.roots ?? [],
    authType: server.authType ?? "none",
  };
}

function normalizeMcpDraft(draft: MCPServerConfig): MCPServerConfig {
  const httpAuthType = draft.authType ?? "none";
  return {
    ...draft,
    name: draft.name || draft.id,
    args: draft.transport === "stdio" ? (draft.args ?? []) : [],
    command: draft.transport === "stdio" ? draft.command : "",
    url: draft.transport === "stdio" ? "" : draft.url,
    authType: draft.transport === "stdio" ? "none" : httpAuthType,
    bearerTokenEnv: draft.transport !== "stdio" && (httpAuthType === "bearer" || httpAuthType === "oauth") ? draft.bearerTokenEnv : "",
    oauthIssuerUrl: draft.transport !== "stdio" && httpAuthType === "oauth" ? draft.oauthIssuerUrl : "",
    oauthClientId: draft.transport !== "stdio" && httpAuthType === "oauth" ? draft.oauthClientId : "",
    oauthScopes: draft.transport !== "stdio" && httpAuthType === "oauth" ? (draft.oauthScopes ?? []) : [],
    roots: draft.roots ?? [],
  };
}

function nonEmptyStrings(values?: string[]) {
  const next = values && values.length > 0 ? values : [""];
  return [...next];
}

function compactStrings(values: string[]) {
  return values.map((value) => value.trim()).filter(Boolean);
}

function mapToRows(value?: Record<string, string>): KeyValueRow[] {
  const rows = Object.entries(value ?? {}).map(([key, rowValue]) => ({
    key,
    value: rowValue,
  }));
  return rows.length > 0 ? rows : [{ key: "", value: "" }];
}

function rowsToMap(rows: KeyValueRow[]) {
  const next: Record<string, string> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (key) {
      next[key] = row.value;
    }
  }
  return next;
}

function mcpTransportLabel(transport: MCPServerConfig["transport"]) {
  switch (transport) {
    case "streamable_http":
      return "Streamable HTTP";
    case "sse":
      return "SSE";
    case "stdio":
    default:
      return "stdio";
  }
}

function parseWords(value: string) {
  return value
    .split(/\s+/g)
    .map((word) => word.trim())
    .filter(Boolean);
}

function canSaveMcpDraft(draft: MCPServerConfig) {
  if (!draft.id.trim()) return false;
  if (draft.transport === "stdio") return Boolean(draft.command?.trim());
  if (draft.authType === "bearer" && !draft.bearerTokenEnv?.trim()) return false;
  return Boolean(draft.url?.trim());
}

function templateVariables(template: string) {
  const variables: string[] = [];
  for (const match of template.matchAll(/\{([A-Za-z0-9_.-]+)\}/g)) {
    const name = match[1];
    if (!variables.includes(name)) {
      variables.push(name);
    }
  }
  return variables;
}

function applySimpleTemplate(template: string, values: Record<string, string>) {
  return template.replace(/\{([A-Za-z0-9_.-]+)\}/g, (_match, name: string) =>
    encodeURIComponent(values[name] ?? ""),
  );
}

function resourcePreview(result: MCPResourceReadResult) {
  if (result.content?.trim()) {
    return result.content;
  }
  const blobCount = result.contents?.filter((content) => content.blob).length ?? 0;
  if (blobCount > 0) {
    return `${blobCount} binary content block${blobCount === 1 ? "" : "s"}`;
  }
  return stringifyStructured(result.structured);
}

function oauthDiscoveryPreview(result: MCPOAuthDiscoveryResult) {
  const lines = [
    ["Resource metadata", result.resourceMetadataUrl],
    ["Resource", result.resource],
    ["Issuer", result.selectedIssuer],
    ["Authorize", result.authorizationEndpoint],
    ["Token", result.tokenEndpoint],
    ["Register", result.registrationEndpoint],
    ["Scopes", result.scopesSupported?.join(" ")],
    ["PKCE", result.codeChallengeMethods?.join(", ")],
    ["Dynamic registration", result.requiresDynamicClientRegistration ? "available" : ""],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`);
  if (result.discoveryErrors?.length) {
    lines.push("", "Discovery warnings:", ...result.discoveryErrors);
  }
  return lines.join("\n") || stringifyStructured(result.authorizationMetadata);
}

function oauthStartPreview(result: MCPOAuthStartResult) {
  return [
    ["Status", result.status],
    ["URL", result.url],
    ["Expires", result.expiresAt],
    ["Instructions", result.instructions],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`)
    .join("\n");
}

function oauthStatusPreview(result: MCPOAuthStatus) {
  return [
    ["Status", result.status],
    ["Connected", result.connected ? "yes" : "no"],
    ["Client ID", result.clientId],
    ["Token source", result.tokenSource],
    ["Expires", result.expiresAt],
    ["Error", result.error],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`)
    .join("\n");
}

function stringifyStructured(value: Record<string, unknown> | undefined) {
  if (!value) {
    return "";
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
