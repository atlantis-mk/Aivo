import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  ToolActivationDialogFooter,
  ToolActivationDialogTabs,
} from "@/features/projects/project-tool-activation-dialog-sections";
import {
  type ToolActivationDialogProps,
  useToolActivationDialogState,
} from "@/features/projects/project-tool-activation-dialog-state";

export function ToolActivationDialog(props: ToolActivationDialogProps) {
  const { onOpenChange, open } = props;
  const state = useToolActivationDialogState(props);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[min(760px,86vh)] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="border-b px-5 py-4 pr-12">
          <DialogTitle>工具</DialogTitle>
          <DialogDescription>管理当前对话可用工具和默认激活列表。</DialogDescription>
        </DialogHeader>

        <ToolActivationDialogTabs
          activeSkillSet={state.activeSkillSet}
          activeToolCount={state.activeToolCount}
          activeToolSet={state.activeToolSet}
          candidates={state.candidates}
          disabled={state.loading || state.saving}
          groupedTools={state.groupedTools}
          inactiveToolCount={state.inactiveToolCount}
          loading={state.loading}
          onImportCandidate={state.importCandidate}
          onLoadSkill={state.loadSkill}
          onToggleSkill={state.toggleSkill}
          onToggleTool={state.toggleTool}
          skillCount={state.skillCount}
          skills={state.skills}
          toggleableToolCount={state.toggleableToolCount}
          usedToolCount={state.usedToolCount}
          usedToolSet={state.usedToolSet}
        />

        <ToolActivationDialogFooter
          hasDraftChanges={state.hasDraftChanges}
          loading={state.loading}
          onSubmit={() => void state.submitActiveToolNames()}
          saving={state.saving}
        />
      </DialogContent>
    </Dialog>
  );
}
