import { Plug, TriangleAlert } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs } from "@/components/ui/tabs";
import { Dialog } from "@/components/ui/dialog";
import { ExtensionInstallDialog } from "@/features/projects/extension-install-dialog";
import { AddToolDialog } from "@/features/projects/extension-settings-add-dialog";
import type { ExtensionSettingsSection } from "@/features/projects/extension-settings-model";
import { useExtensionSettingsState } from "@/features/projects/extension-settings-state";
import { ExtensionSettingsTabPanels } from "@/features/projects/extension-settings-tab-panels";
import { ExtensionSettingsToolbar } from "@/features/projects/extension-settings-toolbar";
import { AgentModeEditorDialog } from "@/features/projects/extension-settings-agent-modes";
import { cn } from "@/lib/utils";

type ExtensionSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sessionId?: string;
  workspaceRoot: string;
};

export function ExtensionSettingsDialog({
  open,
  onOpenChange,
  sessionId,
  workspaceRoot,
}: ExtensionSettingsDialogProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className="max-w-none gap-0 overflow-hidden p-0 data-[side=right]:!w-[min(100vw,clamp(760px,60vw,1100px))] sm:data-[side=right]:!max-w-[min(100vw,clamp(760px,60vw,1100px))]"
        side="right"
      >
        <ExtensionSettingsContent
          active={open}
          sessionId={sessionId}
          surface="dialog"
          workspaceRoot={workspaceRoot}
        />
      </SheetContent>
    </Sheet>
  );
}

type ExtensionSettingsContentProps = {
  active?: boolean;
  className?: string;
  sessionId?: string;
  surface?: "dialog" | "page";
  workspaceRoot?: string;
};

export function ExtensionSettingsContent({
  active = true,
  className,
  sessionId,
  surface = "page",
  workspaceRoot,
}: ExtensionSettingsContentProps) {
  const {
    agentModeEditorOpen,
    agentModes,
    addMcpServer,
    addMcpServers,
    addOpen,
    activeToolSet,
    deleteManagedAgentMode,
    deleteSkill,
    editingAgentMode,
    editAgentMode,
    error,
    extensionInstallOpen,
    extensions,
    ignoreSkillCandidate,
    importSkillCandidate,
    loading,
    openAddDialog,
    providerCatalog,
    query,
    reload,
    saveManagedAgentMode,
    section,
    servers,
    setAddOpen,
    setAgentModeEditorOpen,
    setExtensionInstallOpen,
    setQuery,
    setSection,
    skills,
    toggleSkillEnabled,
    toggleTool,
    visibleAgentModes,
    visibleAllTools,
    visibleExtensions,
    visibleServers,
    visibleSkillCandidates,
    visibleSkills,
    visibleTools,
  } = useExtensionSettingsState({
    active,
    workspaceRoot,
  });

  return (
    <section className={cn("flex h-full min-h-0 flex-col overflow-hidden", className)}>
      <AddToolDialog
        loading={loading}
        onOpenChange={setAddOpen}
        onSaveMCPServer={addMcpServer}
        onSaveMCPServers={addMcpServers}
        open={addOpen}
      />
      <ExtensionInstallDialog
        onInstalled={reload}
        onOpenChange={setExtensionInstallOpen}
        open={extensionInstallOpen}
      />
      <Dialog open={agentModeEditorOpen} onOpenChange={setAgentModeEditorOpen}>
        <AgentModeEditorDialog
          disabled={loading}
          mode={editingAgentMode}
          modes={agentModes}
          onDelete={deleteManagedAgentMode}
          onOpenChange={setAgentModeEditorOpen}
          onSave={saveManagedAgentMode}
          open={agentModeEditorOpen}
          providerCatalog={providerCatalog}
        />
      </Dialog>
      {surface === "dialog" ? (
        <SheetHeader className="border-b px-5 py-4">
          <SheetTitle className="flex items-center gap-2 text-base">
            <Plug />
            扩展与 MCP
          </SheetTitle>
        </SheetHeader>
      ) : null}

      <Tabs
        value={section}
        onValueChange={(value) => setSection(value as ExtensionSettingsSection)}
        className="min-h-0 flex-1 gap-0"
      >
        <ExtensionSettingsToolbar
          agentModeCount={agentModes.length}
          extensionCount={extensions.length}
          loading={loading}
          onAdd={openAddDialog}
          onQueryChange={setQuery}
          onReload={() => void reload()}
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

        <ExtensionSettingsTabPanels
          activeToolSet={activeToolSet}
          loading={loading}
          onEditAgentMode={editAgentMode}
          onDeleteSkill={deleteSkill}
          onIgnoreSkillCandidate={ignoreSkillCandidate}
          onImportSkillCandidate={importSkillCandidate}
          onReload={reload}
          onToggleSkillEnabled={toggleSkillEnabled}
          onToggleTool={toggleTool}
          query={query}
          sessionId={sessionId}
          visibleAllTools={visibleAllTools}
          visibleAgentModes={visibleAgentModes}
          visibleExtensions={visibleExtensions}
          visibleServers={visibleServers}
          visibleSkillCandidates={visibleSkillCandidates}
          visibleSkills={visibleSkills}
        />
      </Tabs>
    </section>
  );
}
