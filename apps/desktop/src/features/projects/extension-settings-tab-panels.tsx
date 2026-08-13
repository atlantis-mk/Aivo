import { ItemGroup } from "@/components/ui/item";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TabsContent } from "@/components/ui/tabs";
import {
  EmptyState,
  SkillManagementGroup,
  ToolItemGroup,
} from "@/features/projects/extension-settings-components";
import { ExtensionInstallList } from "@/features/projects/extension-install-list";
import { AgentModeManagementGroup } from "@/features/projects/extension-settings-agent-modes";
import { McpRow } from "@/features/projects/extension-settings-server-row";
import type {
  AgentModeDefinition,
  SkillEntry,
  SkillImportCandidate,
  ExtensionInstall,
  MCPServerListItem,
  ToolCatalogEntry,
} from "@/services/aivo";

export function ExtensionSettingsTabPanels({
  activeToolSet,
  loading,
  onEditAgentMode,
  onDeleteSkill,
  onIgnoreSkillCandidate,
  onImportSkillCandidate,
  onReload,
  onToggleSkillEnabled,
  onToggleTool,
  query,
  sessionId,
  visibleAllTools,
  visibleAgentModes,
  visibleExtensions,
  visibleServers,
  visibleSkillCandidates,
  visibleSkills,
}: {
  activeToolSet: Set<string>;
  loading: boolean;
  onEditAgentMode: (mode: AgentModeDefinition) => void;
  onDeleteSkill: (skill: SkillEntry) => void;
  onIgnoreSkillCandidate: (candidate: SkillImportCandidate) => void;
  onImportSkillCandidate: (candidate: SkillImportCandidate) => void;
  onReload: () => Promise<void>;
  onToggleSkillEnabled: (skill: SkillEntry, enabled: boolean) => void;
  onToggleTool: (toolName: string, enabled: boolean) => void;
  query: string;
  sessionId?: string;
  visibleAllTools: ToolCatalogEntry[];
  visibleAgentModes: AgentModeDefinition[];
  visibleExtensions: ExtensionInstall[];
  visibleServers: MCPServerListItem[];
  visibleSkillCandidates: SkillImportCandidate[];
  visibleSkills: SkillEntry[];
}) {
  return (
    <>
      <TabsContent className="min-h-0 p-0" value="extensions">
        <ScrollArea className="h-full">
          <div className="p-4">
            <ExtensionInstallList
              items={visibleExtensions}
              loading={loading}
              onReload={onReload}
              query={query}
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

      <TabsContent className="min-h-0 p-0" value="agents">
        <ScrollArea className="h-full">
          <div className="p-4">
            <AgentModeManagementGroup
              disabled={loading}
              modes={visibleAgentModes}
              onEdit={onEditAgentMode}
            />
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
              activeToolSet={activeToolSet}
              disabled={loading}
              emptyLabel={query ? "没有匹配的工具" : "没有可显示工具"}
              onToggle={onToggleTool}
              tools={visibleAllTools}
            />
          </div>
        </ScrollArea>
      </TabsContent>
    </>
  );
}
