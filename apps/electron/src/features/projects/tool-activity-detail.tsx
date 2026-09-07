import { CommandActivityDetail } from "@/features/projects/tool-activity-command-detail";
import { FileActivityDetail } from "@/features/projects/tool-activity-file-detail";
import type {
  ToolActivityFileTab,
  ToolActivityTab,
} from "@/features/projects/tool-activity-model";

export function ToolActivityDetail({
  onApplyFileState,
  tab,
  workspaceRoot,
}: {
  onApplyFileState?: (
    tab: ToolActivityFileTab,
    targetState: "before" | "after",
  ) => void;
  tab: ToolActivityTab;
  workspaceRoot: string;
}) {
  if (tab.kind === "file") {
    return <FileActivityDetail onApplyFileState={onApplyFileState} tab={tab} />;
  }
  return <CommandActivityDetail tab={tab} workspaceRoot={workspaceRoot} />;
}
