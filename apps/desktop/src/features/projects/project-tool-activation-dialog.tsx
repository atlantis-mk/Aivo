import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
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
          <DialogTitle className="flex flex-wrap items-center gap-2">
            <span>工具</span>
            <Badge variant="secondary">激活 {state.activeToolCount}</Badge>
            <Badge variant="outline">未激活 {state.inactiveToolCount}</Badge>
            <Badge variant="outline">已使用 {state.usedToolCount}</Badge>
            <Badge variant="outline">技能 {state.skillCount}</Badge>
          </DialogTitle>
        </DialogHeader>

        <ToolActivationDialogTabs
          activeSkillSet={state.activeSkillSet}
          activeToolSet={state.activeToolSet}
          disabled={state.loading || state.saving}
          groupedSkills={state.groupedSkills}
          groupedTools={state.groupedTools}
          loading={state.loading}
          onToggleSkill={state.toggleSkill}
          onToggleToolGroup={state.toggleToolGroup}
          toggleableToolCount={state.toggleableToolCount}
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
