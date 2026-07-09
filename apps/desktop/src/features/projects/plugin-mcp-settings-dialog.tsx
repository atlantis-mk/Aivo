import { Plug, TriangleAlert } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs } from "@/components/ui/tabs";
import { AddToolDialog } from "@/features/projects/plugin-mcp-settings-add-dialog";
import type { PluginSettingsSection } from "@/features/projects/plugin-mcp-settings-model";
import { usePluginMcpSettingsState } from "@/features/projects/plugin-mcp-settings-state";
import { PluginMcpSettingsTabPanels } from "@/features/projects/plugin-mcp-settings-tab-panels";
import { PluginMcpSettingsToolbar } from "@/features/projects/plugin-mcp-settings-toolbar";
import { cn } from "@/lib/utils";

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

export function PluginMcpSettingsContent({
  active = true,
  className,
  sessionId,
  surface = "page",
  workspaceRoot,
}: PluginMcpSettingsContentProps) {
  const {
    addMcpServer,
    addMcpServers,
    addMode,
    addOpen,
    applicationPlugins,
    deleteSkill,
    error,
    importSkillCandidate,
    installPlugin,
    installPluginPath,
    loading,
    openAddDialog,
    pluginPath,
    plugins,
    query,
    reload,
    reloadPluginCatalog,
    section,
    servers,
    setAddMode,
    setAddOpen,
    setPluginPath,
    setQuery,
    setSection,
    skills,
    toggleSkillEnabled,
    visibleAllTools,
    visiblePlugins,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  } = usePluginMcpSettingsState({
    active,
    workspaceRoot,
  });

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
        <PluginMcpSettingsToolbar
          applicationPluginCount={applicationPlugins.length}
          loading={loading}
          onAdd={openAddDialog}
          onQueryChange={setQuery}
          onReload={() => void reload()}
          onReloadPlugins={reloadPluginCatalog}
          pluginCount={plugins.length}
          query={query}
          section={section}
          serverCount={servers.length}
          skillCount={skills.length}
          toolCount={visibleTools.length}
        />

        {error ? (
          <div className="border-b p-4">
            <Alert variant="destructive">
              <TriangleAlert />
              <AlertTitle>加载失败</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </div>
        ) : null}

        <PluginMcpSettingsTabPanels
          loading={loading}
          onDeleteSkill={deleteSkill}
          onImportSkillCandidate={importSkillCandidate}
          onInstallPlugin={installPlugin}
          onPluginPathChange={setPluginPath}
          onReload={reload}
          onToggleSkillEnabled={toggleSkillEnabled}
          pluginPath={pluginPath}
          query={query}
          sessionId={sessionId}
          visibleAllTools={visibleAllTools}
          visiblePlugins={visiblePlugins}
          visibleServers={visibleServers}
          visibleSkillCandidates={visibleSkillCandidates}
          visibleSkills={visibleSkills}
        />
      </Tabs>
    </section>
  );
}
