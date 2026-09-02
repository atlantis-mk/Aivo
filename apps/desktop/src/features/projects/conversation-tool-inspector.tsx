import { useEffect, useMemo, useRef, useState } from "react";
import {
  Alert02Icon,
  ArrowLeft01Icon,
  ArrowRight01Icon,
  Cancel01Icon,
  CheckmarkCircle02Icon,
  Clock01Icon,
  CommandLineIcon,
  File01Icon,
  GitBranchIcon,
  Loading03Icon,
  Search01Icon,
  ToolsIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { ExtensionToolWebView } from "@/features/projects/extension-tool-web-view";
import {
  extensionToolViewRef,
  latestExtensionViewToolCallId,
  selectedExtensionViewToolCallId,
} from "@/features/projects/extension-tool-view-model";
import { toolTimelineDescription } from "@/features/projects/conversation-tool-inspector-model";
import {
  getToolCallCommand,
  getToolResultText,
} from "@/features/projects/conversation-timeline-tool-command-model";
import type {
  ToolCallActivity,
  ToolCallGroup,
} from "@/features/projects/conversation-timeline-tool-model";
import { stringArg } from "@/features/projects/conversation-timeline-value-model";
import { cn } from "@/lib/utils";
import type { domain } from "../../../bridge/go/models";

export function ConversationToolCallBadge({
  group,
  toolCall,
}: {
  group: ToolCallGroup;
  toolCall: domain.ToolCall;
}) {
  const Icon = toolGroupIcon(group.kind);

  return (
    <Badge variant={toolCallStatusVariant(toolCall.status)}>
      <HugeiconsIcon data-icon="inline-start" icon={Icon} strokeWidth={2} />
      {toolCall.name || "tool"}
    </Badge>
  );
}

export function ConversationToolInspector({
  activity,
  onClose,
}: {
  activity: ToolCallActivity | null;
  onClose: () => void;
}) {
  const [selectedToolCallId, setSelectedToolCallId] = useState("");
  const autoOpenedViewRef = useRef({ activityId: "", toolCallId: "" });
  const timelineEntries = useMemo(
    () => sortedActivityToolCalls(activity),
    [activity],
  );
  const latestViewToolCallId = useMemo(
    () => latestExtensionViewToolCallId(timelineEntries),
    [timelineEntries],
  );
  const effectiveSelectedToolCallId = selectedExtensionViewToolCallId({
    activityId: activity?.id ?? "",
    latestViewToolCallId,
    selectedToolCallId,
    trackedActivityId: autoOpenedViewRef.current.activityId,
  });
  const selectedTimelineEntry = useMemo(
    () =>
      timelineEntries.find(
        ({ toolCall }) => toolCall.id === effectiveSelectedToolCallId,
      ) ?? null,
    [effectiveSelectedToolCallId, timelineEntries],
  );
  const selectedToolCall = selectedTimelineEntry?.toolCall ?? null;
  const selectedToolDescription = selectedToolCall
    ? toolCallDescription(
        selectedToolCall,
        selectedTimelineEntry?.description ?? "",
      )
    : "";
  useEffect(() => {
    const activityId = activity?.id ?? "";
    const state = autoOpenedViewRef.current;
    if (!activityId) {
      autoOpenedViewRef.current = { activityId: "", toolCallId: "" };
      setSelectedToolCallId("");
      return;
    }
    if (state.activityId !== activityId) {
      autoOpenedViewRef.current = {
        activityId,
        toolCallId: latestViewToolCallId,
      };
      setSelectedToolCallId(latestViewToolCallId);
      return;
    }
    if (latestViewToolCallId && state.toolCallId !== latestViewToolCallId) {
      autoOpenedViewRef.current.toolCallId = latestViewToolCallId;
      setSelectedToolCallId(latestViewToolCallId);
    }
  }, [activity?.id, latestViewToolCallId]);

  const open = Boolean(activity);

  return (
    <aside
      aria-hidden={!open}
      className={cn(
        "h-full min-h-0 shrink-0 overflow-hidden transition-[width,opacity] duration-300 ease-out max-[760px]:absolute max-[760px]:inset-y-0 max-[760px]:right-0 max-[760px]:z-50",
        open
          ? "w-[min(38vw,420px)] opacity-100 max-[980px]:w-[min(45vw,380px)] max-[760px]:w-[min(92vw,380px)]"
          : "pointer-events-none w-0 opacity-0",
      )}
      data-state={open ? "open" : "closed"}
      data-tool-activity-inspector
    >
      <div
        className={cn(
          "h-full w-[min(38vw,420px)] p-2 transition-transform duration-300 ease-out max-[980px]:w-[min(45vw,380px)] max-[760px]:w-[min(92vw,380px)]",
          open ? "translate-x-0" : "translate-x-full",
        )}
      >
        {activity ? (
          <Card className="relative h-full min-h-0" size="sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Badge
                  variant={activityBadgeVariant(
                    timelineEntries.map(({ toolCall }) => toolCall),
                  )}
                >
                  {timelineEntries.length} 个工具
                </Badge>
                工具时间线
              </CardTitle>
              <CardDescription>
                按调用时间查看全部工具及执行状态
              </CardDescription>
              <CardAction>
                <Button
                  aria-label="关闭工具活动"
                  onClick={onClose}
                  size="icon-sm"
                  title="关闭"
                  type="button"
                  variant="ghost"
                >
                  <HugeiconsIcon icon={Cancel01Icon} strokeWidth={2} />
                </Button>
              </CardAction>
            </CardHeader>
            <CardContent className="relative min-h-0 flex-1 overflow-visible">
              <ToolCallTimeline
                entries={timelineEntries}
                onSelect={setSelectedToolCallId}
              />
              <ToolCallDetail
                description={selectedToolDescription}
                onBack={() => setSelectedToolCallId("")}
                toolCall={selectedToolCall}
              />
            </CardContent>
          </Card>
        ) : null}
      </div>
    </aside>
  );
}

function ToolCallTimeline({
  entries,
  onSelect,
}: {
  entries: ToolCallTimelineEntry[];
  onSelect: (toolCallId: string) => void;
}) {
  return (
    <ScrollArea className="h-full pr-2">
      <ItemGroup className="relative gap-0 py-1">
        {entries.map(({ description, toolCall }, index) => {
          const command = getToolCallCommand(toolCall);
          const invocationDescription = toolCallDescription(
            toolCall,
            description,
          );
          const previousEntry = entries[index - 1];
          const previousInvocationDescription = previousEntry
            ? toolCallDescription(
                previousEntry.toolCall,
                previousEntry.description,
              )
            : "";
          const title = toolTimelineDescription(
            invocationDescription,
            previousInvocationDescription,
          );
          return (
            <div className="relative pb-2 last:pb-0" key={toolCall.id}>
              {index < entries.length - 1 ? (
                <Separator
                  className="pointer-events-none absolute top-[22px] -bottom-8 left-[22px]"
                  orientation="vertical"
                />
              ) : null}
              <Item asChild size="sm">
                <button
                  aria-label={`查看 ${toolCall.name || command.label} 详情`}
                  className="items-start text-left hover:bg-muted/50"
                  onClick={() => onSelect(toolCall.id)}
                  type="button"
                >
                  <ItemMedia variant="icon">
                    <Badge
                      className="relative z-10 size-5 p-0"
                      variant={toolCallStatusVariant(toolCall.status)}
                    >
                      <ToolCallStatusIcon status={toolCall.status} />
                    </Badge>
                  </ItemMedia>
                  <ItemContent>
                    <ItemDescription>
                      {formatToolTimelineTime(toolCall.timeCreated)}
                    </ItemDescription>
                    {title ? (
                      <ItemTitle>{title}</ItemTitle>
                    ) : null}
                    <ItemDescription>
                      {toolCall.name || command.label}
                    </ItemDescription>
                  </ItemContent>
                  <ItemActions>
                    <Badge variant={toolCallStatusVariant(toolCall.status)}>
                      {toolCallStatusLabel(toolCall.status)}
                    </Badge>
                    <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} />
                  </ItemActions>
                </button>
              </Item>
            </div>
          );
        })}
      </ItemGroup>
    </ScrollArea>
  );
}

function ToolCallDetail({
  description,
  onBack,
  toolCall,
}: {
  description: string;
  onBack: () => void;
  toolCall: domain.ToolCall | null;
}) {
  const command = toolCall ? getToolCallCommand(toolCall) : null;
  const resultText = toolCall ? getToolResultText(toolCall) : "";
  const argumentsText = toolCall ? formatToolArguments(toolCall.arguments) : "";
  const view = extensionToolViewRef(toolCall);

  return (
    <div
      aria-hidden={!toolCall}
      className={cn(
        "absolute -inset-y-1 -right-1 left-3 transition-[transform,opacity] duration-200 ease-out",
        toolCall
          ? "translate-x-0 opacity-100"
          : "pointer-events-none translate-x-[calc(100%+1rem)] opacity-0",
      )}
    >
      {toolCall && command ? (
        <Card className="h-full min-h-0 shadow-lg" size="sm">
          <CardHeader>
            {description ? (
              <CardTitle>{description}</CardTitle>
            ) : null}
            <CardDescription>{view?.title || command.label}</CardDescription>
            <CardAction className="flex items-center gap-1">
              <Button
                aria-label="返回工具时间线"
                onClick={onBack}
                size="icon-sm"
                title="返回"
                type="button"
                variant="ghost"
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} strokeWidth={2} />
              </Button>
            </CardAction>
          </CardHeader>
          <CardContent
            className={cn(
              "min-h-0 flex-1",
              view && "overflow-hidden p-0",
            )}
          >
            {view ? (
              <ExtensionToolWebView
                fallback={
                  <NativeToolCallDetails
                    argumentsText={argumentsText}
                    resultText={resultText}
                    toolCall={toolCall}
                  />
                }
                onRequestClose={onBack}
                toolCall={toolCall}
                view={view}
              />
            ) : (
              <NativeToolCallDetails
                argumentsText={argumentsText}
                resultText={resultText}
                toolCall={toolCall}
              />
            )}
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

function NativeToolCallDetails({
  argumentsText,
  resultText,
  toolCall,
}: {
  argumentsText: string;
  resultText: string;
  toolCall: domain.ToolCall;
}) {
  return (
    <ScrollArea className="h-full pr-2">
      <div className="flex flex-col gap-3 px-1 pb-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={toolCallStatusVariant(toolCall.status)}>
            {toolCallStatusLabel(toolCall.status)}
          </Badge>
          <Badge variant="outline">{formatToolTime(toolCall.timeCreated)}</Badge>
          {toolCall.timeUpdated &&
          toolCall.timeUpdated !== toolCall.timeCreated ? (
            <Badge variant="outline">
              更新于 {formatToolTime(toolCall.timeUpdated)}
            </Badge>
          ) : null}
        </div>
        <DetailCard
          content={argumentsText || "无参数"}
          description="发送给工具的结构化参数"
          title="参数"
        />
        <DetailCard
          content={resultText || toolCall.error || "暂无结果"}
          description="工具返回的安全摘要或可见内容"
          destructive={Boolean(toolCall.error)}
          title="结果"
        />
      </div>
    </ScrollArea>
  );
}

function DetailCard({
  content,
  description,
  destructive = false,
  title,
}: {
  content: string;
  description: string;
  destructive?: boolean;
  title: string;
}) {
  return (
    <section className="overflow-hidden rounded-2xl border border-border/80 bg-card text-card-foreground shadow-sm shadow-foreground/[0.03]">
      <div className="flex min-h-11 flex-col justify-center gap-0.5 px-4 pt-3 pb-2">
        <div className="text-sm font-semibold">{title}</div>
        <div className="text-xs text-muted-foreground">{description}</div>
      </div>
      <div className="px-4 pb-4 pt-1">
        <ScrollArea className="h-48 pr-3">
          <pre
            className={cn(
              "whitespace-pre-wrap break-words font-mono text-xs leading-relaxed",
              destructive && "text-destructive",
            )}
          >
            {content}
          </pre>
        </ScrollArea>
      </div>
    </section>
  );
}

function ToolCallStatusIcon({ status }: { status: string }) {
  if (status === "running") {
    return (
      <HugeiconsIcon
        className="animate-spin"
        icon={Loading03Icon}
        strokeWidth={2}
      />
    );
  }
  if (status === "pending_approval") {
    return <HugeiconsIcon icon={Clock01Icon} strokeWidth={2} />;
  }
  if (status === "failed") {
    return <HugeiconsIcon icon={Alert02Icon} strokeWidth={2} />;
  }
  return <HugeiconsIcon icon={CheckmarkCircle02Icon} strokeWidth={2} />;
}

function toolCallDescription(
  toolCall: domain.ToolCall,
  activityDescription = "",
) {
  return (
    activityDescription.trim() ||
    stringArg(toolCall.arguments ?? {}, "description").trim()
  );
}

function toolGroupIcon(kind: string) {
  switch (kind) {
    case "read":
    case "list":
      return File01Icon;
    case "search":
    case "tool-search":
      return Search01Icon;
    case "write":
      return File01Icon;
    case "git":
      return GitBranchIcon;
    case "shell":
      return CommandLineIcon;
    default:
      return ToolsIcon;
  }
}

function activityBadgeVariant(toolCalls: domain.ToolCall[]) {
  if (toolCalls.some((toolCall) => toolCall.status === "failed")) {
    return "destructive" as const;
  }
  if (
    toolCalls.some(
      (toolCall) =>
        toolCall.status === "running" ||
        toolCall.status === "pending_approval",
    )
  ) {
    return "secondary" as const;
  }
  return "outline" as const;
}

function toolCallStatusVariant(status: string) {
  if (status === "failed") return "destructive" as const;
  if (status === "running" || status === "pending_approval") {
    return "secondary" as const;
  }
  return "outline" as const;
}

function toolCallStatusLabel(status: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "pending_approval":
      return "等待批准";
    case "failed":
      return "失败";
    default:
      return "已完成";
  }
}

function formatToolArguments(value: Record<string, unknown> | undefined) {
  if (!value || Object.keys(value).length === 0) return "";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function formatToolTime(value: string | undefined) {
  if (!value) return "时间未知";
  const time = new Date(value);
  if (Number.isNaN(time.getTime())) return "时间未知";
  return new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(time);
}

function formatToolTimelineTime(value: string | undefined) {
  if (!value) return "调用时间未知";
  const time = new Date(value);
  if (Number.isNaN(time.getTime())) return "调用时间未知";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(time);
}

function sortedActivityToolCalls(activity: ToolCallActivity | null) {
  if (!activity) return [];
  return activity.groups
    .flatMap((group) =>
      group.calls.map((toolCall) => ({
        description: group.description ?? "",
        toolCall,
      })),
    )
    .map((entry, index) => ({ entry, index }))
    .toSorted((left, right) => {
      const leftTime = Date.parse(left.entry.toolCall.timeCreated || "");
      const rightTime = Date.parse(right.entry.toolCall.timeCreated || "");
      if (!Number.isNaN(leftTime) && !Number.isNaN(rightTime)) {
        const timeDelta = leftTime - rightTime;
        if (timeDelta !== 0) return timeDelta;
      } else if (!Number.isNaN(leftTime)) {
        return -1;
      } else if (!Number.isNaN(rightTime)) {
        return 1;
      }
      return left.index - right.index;
    })
    .map(({ entry }) => entry);
}

type ToolCallTimelineEntry = {
  description: string;
  toolCall: domain.ToolCall;
};
