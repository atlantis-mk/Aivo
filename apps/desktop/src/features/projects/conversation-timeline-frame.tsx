import { useState, type ReactNode } from "react";
import { ChevronDown, LoaderCircle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { AnimatedDisclosure } from "@/features/projects/conversation-timeline-disclosure";
import type { ConversationSystemNote } from "@/features/projects/conversation-timeline-model";
import { hostToolSelectionFromSystemNote } from "@/features/projects/conversation-system-note-model";
import {
  toolInjectionResourceKinds,
  toolInjectionResourceLabels,
} from "@/features/projects/tool-injection-resource-model";
import { cn } from "@/lib/utils";

export function TimelineRowFrame({
  children,
  role,
  turnId,
}: {
  children: ReactNode;
  role: "assistant" | "user";
  turnId: string;
}) {
  return (
    <div data-timeline-role={role} data-turn-id={turnId}>
      {children}
    </div>
  );
}

export function SystemNoteRow({ note }: { note: ConversationSystemNote }) {
  const toolSelection = hostToolSelectionFromSystemNote(note);
  if (toolSelection) {
    return (
      <InitialToolSelectionRow
        lifetime={toolSelection.lifetime}
        status={toolSelection.status}
        resources={toolSelection.resources}
      />
    );
  }
  return (
    <div className="my-2 flex justify-center">
      <div className="max-w-[min(90%,34rem)] rounded-md border border-border/70 bg-muted/35 px-3 py-2 text-center text-xs text-muted-foreground">
        {note.content}
      </div>
    </div>
  );
}

function InitialToolSelectionRow({
  lifetime,
  resources,
  status,
}: {
  lifetime?: "conversation" | "request";
  resources: NonNullable<ReturnType<typeof hostToolSelectionFromSystemNote>>["resources"];
  status: "completed" | "failed" | "running";
}) {
  const [open, setOpen] = useState(false);
  const toolCount = resources.reduce(
    (total, resource) => total + resource.toolCount,
    0,
  );
  let summary = "未注入额外工具";
  if (status === "running") {
    summary = "正在调用辅助模型…";
  } else if (status === "failed") {
    summary = "注入失败";
  } else if (resources.length > 0) {
    summary =
      lifetime === "request"
        ? `本次临时注入 ${resources.length} 个资源（${toolCount} 个工具）`
        : `已为当前对话注入 ${resources.length} 个资源（${toolCount} 个工具）`;
  }

  return (
    <div className="my-2 flex justify-center">
      <div
        aria-busy={status === "running"}
        aria-live="polite"
        className="w-full overflow-hidden rounded-lg border border-border/70 bg-muted/35 text-left text-xs text-muted-foreground"
      >
        <button
          aria-expanded={open}
          className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left outline-none transition-colors hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          onClick={() => setOpen((current) => !current)}
          type="button"
        >
          <span className="font-medium text-foreground">前置工具搜索</span>
          <span
            className={cn(
              "min-w-0 flex-1 truncate",
              status === "failed" && "text-destructive",
            )}
          >
            {summary}
          </span>
          {status === "running" ? (
            <LoaderCircle
              aria-hidden="true"
              className="size-3.5 shrink-0 animate-spin"
            />
          ) : null}
          <ChevronDown
            aria-hidden="true"
            className={cn(
              "size-3.5 shrink-0 transition-transform",
              open && "rotate-180",
            )}
          />
        </button>
        <AnimatedDisclosure open={open}>
          <div className="border-t border-border/60 px-3 py-2.5">
            {status === "running" ? (
              <div className="flex items-center gap-2">
                <LoaderCircle
                  aria-hidden="true"
                  className="size-3.5 animate-spin"
                />
                <span>正在搜索并准备可用工具…</span>
              </div>
            ) : status === "failed" ? (
              <p className="text-destructive">
                未能完成前置工具注入，请重试当前任务。
              </p>
            ) : resources.length > 0 ? (
              <div
                aria-label="前置搜索注入的资源"
                className="space-y-2.5"
              >
                {toolInjectionResourceKinds.map((kind) => {
                  const items = resources.filter(
                    (resource) => resource.kind === kind,
                  );
                  if (items.length === 0) return null;
                  return (
                    <div className="flex items-start gap-2" key={kind}>
                      <span className="w-9 shrink-0 pt-1 font-medium text-foreground">
                        {toolInjectionResourceLabels[kind]}
                      </span>
                      <div className="flex min-w-0 flex-1 flex-wrap gap-1.5">
                        {items.map((resource) => (
                          <Badge
                            className="h-auto max-w-full gap-1.5 break-all py-1 whitespace-normal"
                            key={`${resource.kind}:${resource.id}`}
                            title={`${resource.id} · ${resource.toolCount} 个工具`}
                            variant="outline"
                          >
                            <span>{resource.name}</span>
                            <span className="text-muted-foreground">
                              {resource.toolCount}
                            </span>
                          </Badge>
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p>当前任务未匹配到需要额外注入的工具。</p>
            )}
          </div>
        </AnimatedDisclosure>
      </div>
    </div>
  );
}
