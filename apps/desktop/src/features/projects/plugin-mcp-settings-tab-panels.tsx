import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ItemGroup } from "@/components/ui/item";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TabsContent } from "@/components/ui/tabs";
import {
  EmptyState,
  PluginItemGroup,
  SkillManagementGroup,
  ToolItemGroup,
} from "@/features/projects/plugin-mcp-settings-components";
import { McpRow } from "@/features/projects/plugin-mcp-settings-server-row";
import type {
  SkillEntry,
  SkillImportCandidate,
  MCPServerListItem,
  PluginListItem,
  ToolCatalogEntry,
} from "@/services/aivo";

export function PluginMcpSettingsTabPanels({
  loading,
  onDeleteSkill,
  onIgnoreSkillCandidate,
  onImportSkillCandidate,
  onInstallPlugin,
  onPluginPathChange,
  onReload,
  onToggleSkillEnabled,
  pluginPath,
  query,
  sessionId,
  visibleAllTools,
  visiblePlugins,
  visibleServers,
  visibleSkillCandidates,
  visibleSkills,
}: {
  loading: boolean;
  onDeleteSkill: (skill: SkillEntry) => void;
  onIgnoreSkillCandidate: (candidate: SkillImportCandidate) => void;
  onImportSkillCandidate: (candidate: SkillImportCandidate) => void;
  onInstallPlugin: () => void;
  onPluginPathChange: (path: string) => void;
  onReload: () => Promise<void>;
  onToggleSkillEnabled: (skill: SkillEntry, enabled: boolean) => void;
  pluginPath: string;
  query: string;
  sessionId?: string;
  visibleAllTools: ToolCatalogEntry[];
  visiblePlugins: PluginListItem[];
  visibleServers: MCPServerListItem[];
  visibleSkillCandidates: SkillImportCandidate[];
  visibleSkills: SkillEntry[];
}) {
  return (
    <>
      <TabsContent className="min-h-0 p-0" value="plugins">
        <ScrollArea className="h-full">
          <div className="flex flex-col gap-4 p-4">
            <div className="flex gap-2">
              <Input
                onChange={(event) => onPluginPathChange(event.target.value)}
                placeholder="/path/to/plugin"
                value={pluginPath}
              />
              <Button
                disabled={loading || !pluginPath.trim()}
                onClick={onInstallPlugin}
              >
                安装
              </Button>
            </div>
            <PluginItemGroup
              emptyLabel={query ? "没有匹配的插件" : "没有已安装插件"}
              items={visiblePlugins}
              onReload={onReload}
            />
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent className="min-h-0 p-0" value="apps">
        <ScrollArea className="h-full">
          <div className="p-4">
            <PluginItemGroup
              emptyLabel={query ? "没有匹配的应用" : "没有应用插件"}
              items={visiblePlugins}
              onReload={onReload}
            />
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent className="min-h-0 p-0" value="mcp">
        <ScrollArea className="h-full">
          <div className="p-4">
            {visibleServers.length === 0 ? (
              <EmptyState
                label={query ? "没有匹配的 MCP server" : "没有 MCP server"}
              />
            ) : (
              <ItemGroup className="gap-3">
                {visibleServers.map((item) => (
                  <McpRow
                    item={item}
                    key={item.server.id}
                    onReload={onReload}
                    sessionId={sessionId}
                  />
                ))}
              </ItemGroup>
            )}
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent className="min-h-0 p-0" value="skills">
        <ScrollArea className="h-full">
          <div className="p-4">
            <SkillManagementGroup
              candidates={visibleSkillCandidates}
              loading={loading}
              onDelete={onDeleteSkill}
              onIgnore={onIgnoreSkillCandidate}
              onImport={onImportSkillCandidate}
              onToggleEnabled={onToggleSkillEnabled}
              skills={visibleSkills}
            />
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent className="min-h-0 p-0" value="tools">
        <ScrollArea className="h-full">
          <div className="p-4">
            <ToolItemGroup
              emptyLabel={query ? "没有匹配的工具" : "没有可显示工具"}
              tools={visibleAllTools}
            />
          </div>
        </ScrollArea>
      </TabsContent>
    </>
  );
}
