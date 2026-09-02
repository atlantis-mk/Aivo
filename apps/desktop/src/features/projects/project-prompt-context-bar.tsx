import { GitBranch } from "lucide-react";

import { ProjectPicker } from "@/features/projects/project-picker-popover";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";
import { AgentModeMenu } from "@/features/projects/project-prompt-mode-menus";

type PromptContextBarProps = Pick<
  PromptComposerProps,
  | "agentMode"
  | "agentModes"
  | "onAgentModeSelect"
  | "onProjectAdd"
  | "onProjectClear"
  | "onProjectSelect"
  | "project"
  | "projectPath"
  | "projects"
>;

export function PromptContextBar({
  agentMode,
  agentModes,
  onAgentModeSelect,
  onProjectAdd,
  onProjectClear,
  onProjectSelect,
  project,
  projectPath,
  projects,
}: PromptContextBarProps) {
  const branchLabel = project?.gitBranch?.trim() ?? "";

  return (
    <div className="relative flex min-h-14 min-w-0 items-start overflow-hidden rounded-t-3xl bg-muted/70 px-3 pb-4 pt-2 ring-1 ring-foreground/10">
      <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <ProjectPicker
          onAddProject={onProjectAdd}
          onProjectClear={onProjectClear}
          onProjectSelect={onProjectSelect}
          project={project}
          projectPath={projectPath}
          projects={projects}
        />

        {branchLabel ? (
          <div
            className="flex min-w-0 shrink-0 items-center gap-2 px-2.5 text-xs font-medium"
            title={branchLabel}
          >
            <GitBranch aria-hidden="true" className="size-4 shrink-0" />
            <span className="max-w-52 truncate">
              {branchLabel}
              {project?.gitDirty ? " *" : ""}
            </span>
          </div>
        ) : null}

        <AgentModeMenu
          compact={false}
          mode={agentMode}
          modes={agentModes}
          onModeSelect={onAgentModeSelect}
        />
      </div>
    </div>
  );
}
