import { ArrowDown, File, X } from "lucide-react";

import { EnvironmentSummaryPanel } from "@/components/app-top-bar";
import { Button } from "@/components/ui/button";
import { TodoFloatingStatus } from "@/features/projects/project-todo-floating-status";
import { cn } from "@/lib/utils";
import type { TodoItem } from "@/services/aivo";

export function ProjectWorkspaceEmptyPrompt({
  showConversationLayout,
}: {
  showConversationLayout: boolean;
}) {
  return (
    <h1
      className={cn(
        "absolute left-1/2 top-[calc(50%-100px)] w-[calc(100%-2rem)] -translate-x-1/2 text-center text-3xl leading-none tracking-normal text-foreground transition-all duration-500 ease-out",
        showConversationLayout
          ? "-translate-y-8 opacity-0"
          : "translate-y-0 opacity-100",
      )}
    >
      我们该做什么？
    </h1>
  );
}

export function ProjectComposerDropOverlay({
  active,
}: {
  active: boolean;
}) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-4 z-40 flex items-center justify-center rounded-2xl border border-dashed border-primary/50 bg-background/75 text-sm font-medium text-foreground shadow-lg shadow-foreground/5 backdrop-blur-sm transition-opacity duration-150 sm:inset-6",
        active ? "opacity-100" : "opacity-0",
      )}
    >
      <div className="flex items-center gap-2 rounded-full bg-card px-4 py-2 shadow-sm">
        <File className="size-4 text-primary" />
        <span>拖放文件或图片以添加到输入框</span>
      </div>
    </div>
  );
}

export function ProjectEnvironmentSummaryAside({
  canDockPinnedSummary,
  onOpenSkills,
  onOpenTools,
}: {
  canDockPinnedSummary: boolean;
  onOpenSkills: () => void;
  onOpenTools: () => void;
}) {
  return (
    <aside
      className={cn(
        "absolute right-4 top-9 hidden min-[1050px]:block sm:right-6",
        canDockPinnedSummary ? "z-10" : "z-20",
      )}
    >
      <EnvironmentSummaryPanel
        onOpenSkills={onOpenSkills}
        onOpenTools={onOpenTools}
      />
    </aside>
  );
}

export function ProjectComposerFloatingControls({
  isVisibleTodoPlanComplete,
  onHideCompletedTodoPlan,
  onScrollToBottom,
  shouldShowTodoFloatingStatus,
  showScrollToBottomButton,
  todoItems,
}: {
  isVisibleTodoPlanComplete: boolean;
  onHideCompletedTodoPlan: () => void;
  onScrollToBottom: () => void;
  shouldShowTodoFloatingStatus: boolean;
  showScrollToBottomButton: boolean;
  todoItems: TodoItem[];
}) {
  if (!showScrollToBottomButton && !shouldShowTodoFloatingStatus) return null;

  return (
    <div className="absolute bottom-[calc(100%+0.75rem)] left-1/2 z-10 flex -translate-x-1/2 flex-col items-center gap-2">
      {showScrollToBottomButton ? (
        <Button
          aria-label="滚动到最新消息"
          className="rounded-full"
          onClick={onScrollToBottom}
          size="icon"
          type="button"
        >
          <ArrowDown />
        </Button>
      ) : null}
      {shouldShowTodoFloatingStatus ? (
        <div className="flex items-center gap-1">
          <TodoFloatingStatus todoItems={todoItems} />
          {isVisibleTodoPlanComplete ? (
            <Button
              aria-label="隐藏计划列表"
              onClick={onHideCompletedTodoPlan}
              size="icon"
              type="button"
            >
              <X />
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
