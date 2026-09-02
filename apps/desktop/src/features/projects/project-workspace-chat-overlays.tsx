import { useEffect, useState } from "react";
import {
  CodeIcon,
  File01Icon,
  PlayIcon,
  Search01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { ArrowDown, File, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { TodoFloatingStatus } from "@/features/projects/project-todo-floating-status";
import { cn } from "@/lib/utils";
import { appNameFromConfig } from "@/lib/app-identity";
import { useAppConfig } from "@/lib/app-config";
import { listPromptDocuments, type TodoItem } from "@/services/aivo";

export function ProjectWorkspaceEmptyPrompt({
  onPromptChange,
  showConversationLayout,
}: {
  onPromptChange: (prompt: string) => void;
  showConversationLayout: boolean;
}) {
  const appName = appNameFromConfig(useAppConfig((state) => state.config));
  const [quickPrompts, setQuickPrompts] = useState<
    Array<{ id: string; label: string; prompt: string }>
  >([]);

  useEffect(() => {
    const load = () => {
      void listPromptDocuments()
        .then((items) =>
          setQuickPrompts(
            items
              .filter(
                (item) =>
                  item.category === "quick_prompt" &&
                  item.enabled &&
                  item.status !== "invalid",
              )
              .map((item) => ({
                id: item.id,
                label: item.title,
                prompt: item.body,
              })),
          ),
        )
        .catch(() => setQuickPrompts([]));
    };
    load();
    window.addEventListener("aivo:prompts-changed", load);
    return () => window.removeEventListener("aivo:prompts-changed", load);
  }, []);

  return (
    <div
      className={cn(
        "absolute left-1/2 top-[34%] z-10 flex w-[calc(100%-2rem)] max-w-[760px] -translate-x-1/2 flex-col items-center text-center transition-all duration-500 ease-out",
        showConversationLayout
          ? "pointer-events-none -translate-y-8 opacity-0"
          : "translate-y-0 opacity-100",
      )}
    >
      <h1 className="aivo-type-large-title text-foreground">{appName}</h1>
      <p className="aivo-type-title-3 mt-3 text-muted-foreground">
        描述目标，{appName} 会帮你推进
      </p>
      {quickPrompts.length > 0 ? (
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          {quickPrompts.map((item) => (
            <Button
              className="h-8 rounded-full border-border/80 bg-card px-3.5 text-[13px] font-medium shadow-sm shadow-foreground/[0.025]"
              key={item.id}
              onClick={() => onPromptChange(item.prompt)}
              type="button"
              variant="outline"
            >
              <HugeiconsIcon icon={quickPromptIcon(item.id)} strokeWidth={1.8} />
              {item.label}
            </Button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function quickPromptIcon(id: string) {
  if (id.includes("organize")) return File01Icon;
  if (id.includes("analyze")) return CodeIcon;
  if (id.includes("run")) return PlayIcon;
  return Search01Icon;
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
        <span>拖放文件或文件夹以添加到输入框</span>
      </div>
    </div>
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
