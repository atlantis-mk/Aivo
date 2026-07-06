import {
 memo,
 useEffect,
 useLayoutEffect,
 useMemo,
 useRef,
 useState,
 type ReactNode,
 type RefObject,
} from "react";
import {
 Check,
 ChevronDown,
 Copy,
 ExternalLink,
 Expand,
 File,
 Image,
 Pencil,
 RotateCcw,
 ThumbsDown,
 ThumbsUp,
 Trash2,
} from "lucide-react";

import { Markdown } from "@/components/markdown";
import {
 Attachment,
 AttachmentContent,
 AttachmentDescription,
 AttachmentMedia,
 AttachmentTitle,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import { Marker, MarkerContent } from "@/components/ui/marker";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
 sameToolCalls,
 type ConversationAssistantTextPart,
 type ConversationSystemNote,
 type ConversationTurn,
 type ConversationUserAttachment,
} from "@/features/projects/conversation-timeline-model";
import { readRetainedOutput, type AgentRun } from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

const COLLAPSED_USER_MESSAGE_HEIGHT = 420;
const INLINE_SHELL_OUTPUT_CHARS = 4_000;
const DISCLOSURE_ANIMATION_MS = 200;

type ConversationTimelineRow =
 | {
 type: "turn-gap";
 key: string;
 turnId: string;
 }
 | {
 type: "user-message";
 key: string;
 turn: ConversationTurn;
 }
 | {
 type: "assistant-preamble";
 key: string;
 text: string;
 turnId: string;
 }
 | {
 type: "tool-group";
 key: string;
 group: ToolCallGroup;
 turnId: string;
 }
 | {
 type: "assistant-status";
 key: string;
 turn: ConversationTurn;
 }
 | {
 type: "assistant-response";
 key: string;
 turn: ConversationTurn;
 }
 | {
 type: "system-note";
 key: string;
 note: ConversationSystemNote;
 turnId: string;
 }
 | {
 type: "thinking";
 actionHeading?: string;
 key: string;
 showSkeleton: boolean;
 turnId: string;
 }
 | {
 type: "stopped";
 key: string;
 stoppedSeconds: number;
 turnId: string;
 };

type ConversationTimelineActions = {
 onDeleteAssistantMessage?: (turn: ConversationTurn) => void;
 onDeleteTurn?: (turn: ConversationTurn) => void;
 onEditUserMessage?: (turn: ConversationTurn) => void;
 onRetryTurn?: (turn: ConversationTurn) => void;
};

function AnimatedDisclosure({
 children,
 className,
 open,
}: {
 children: ReactNode;
 className?: string;
 open: boolean;
}) {
 const [shouldRender, setShouldRender] = useState(open);
 const renderChildren = open || shouldRender;

 useEffect(() => {
 if (open) {
 setShouldRender(true);
 return;
 }

 const timeout = window.setTimeout(
 () => setShouldRender(false),
 DISCLOSURE_ANIMATION_MS,
 );
 return () => window.clearTimeout(timeout);
 }, [open]);

 if (!open && !renderChildren) return null;

 return (
 <div
 aria-hidden={!open}
 className={cn(
 "grid overflow-hidden transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
 open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
 className,
 )}
 >
 {renderChildren ? (
 <div className="min-h-0 overflow-hidden">{children}</div>
 ) : null}
 </div>
 );
}

export const SubmittedPromptContent = memo(function SubmittedPromptContent({
 agentRuns = [],
 contentRef,
 dockPinnedSummary,
 onDeleteAssistantMessage,
 onDeleteTurn,
 onEditUserMessage,
 onOpenSession,
 onRetryTurn,
 revealFromHistory,
 reservePermissionDock,
 turns,
}: {
 agentRuns?: AgentRun[];
 contentRef: RefObject<HTMLDivElement | null>;
 dockPinnedSummary: boolean;
 onDeleteAssistantMessage?: (turn: ConversationTurn) => void;
 onDeleteTurn?: (turn: ConversationTurn) => void;
 onEditUserMessage?: (turn: ConversationTurn) => void;
 onOpenSession?: (sessionId: string) => void;
 onRetryTurn?: (turn: ConversationTurn) => void;
 revealFromHistory: boolean;
 reservePermissionDock: boolean;
 turns: ConversationTurn[];
}) {
 const rows = useMemo(() => constructConversationTimelineRows(turns), [turns]);
 const actions = useMemo(
 () => ({
 onDeleteAssistantMessage,
 onDeleteTurn,
 onEditUserMessage,
 onRetryTurn,
 }),
 [
 onDeleteAssistantMessage,
 onDeleteTurn,
 onEditUserMessage,
 onRetryTurn,
 ],
 );

 return (
 <div
 className={cn(
 "mx-auto flex w-[calc(100%-2rem)] max-w-[680px] flex-col px-0 pt-12 transition-transform ease-out sm:w-[calc(100%-48px)]",
 reservePermissionDock
 ? "pb-[19rem]"
 : "pb-[calc(var(--conversation-bottom-height)+3rem)]",
 revealFromHistory
 ? "animate-in fade-in duration-200 [&_.animate-in]:animate-none"
 : "animate-in fade-in slide-in-from-bottom-3 duration-500",
 dockPinnedSummary && "min-[1050px]:-translate-x-40",
 )}
 ref={contentRef}
 >
 {rows.map((row) => (
 <ConversationTimelineRowView
 actions={actions}
 agentRuns={agentRuns}
 key={row.key}
 onOpenSession={onOpenSession}
 row={row}
 />
 ))}
 </div>
 );
});

const ConversationTimelineRowView = memo(function ConversationTimelineRowView({
 actions,
 agentRuns,
 onOpenSession,
 row,
}: {
 actions: ConversationTimelineActions;
 agentRuns: AgentRun[];
 onOpenSession?: (sessionId: string) => void;
 row: ConversationTimelineRow;
}) {
 switch (row.type) {
 case "turn-gap":
 return <div aria-hidden="true" className="h-7" />;
 case "user-message":
 return <UserMessageRow actions={actions} turn={row.turn} />;
 case "assistant-preamble":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turnId}>
 <AssistantPreamble text={row.text} />
 </TimelineRowFrame>
 );
 case "tool-group":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turnId}>
 <TimelineToolGroup
 agentRuns={agentRuns}
 group={row.group}
 onOpenSession={onOpenSession}
 />
 </TimelineRowFrame>
 );
 case "assistant-status":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turn.id}>
 <AssistantStatus
 actionHeading={undefined}
 completed={Boolean(row.turn.responseCompletedAt)}
 responseSeconds={row.turn.thinkingSeconds}
 />
 </TimelineRowFrame>
 );
 case "assistant-response":
 return (
      <TimelineRowFrame role="assistant" turnId={row.turn.id}>
 <AssistantResponse
 actions={actions}
 completedAt={row.turn.responseCompletedAt}
 responseText={row.turn.responseText}
 turn={row.turn}
 />
 </TimelineRowFrame>
 );
 case "system-note":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turnId}>
 <SystemNoteRow note={row.note} />
 </TimelineRowFrame>
 );
 case "thinking":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turnId}>
 <ThinkingResponse
 actionHeading={row.actionHeading}
 showSkeleton={row.showSkeleton}
 />
 </TimelineRowFrame>
 );
 case "stopped":
 return (
 <TimelineRowFrame role="assistant" turnId={row.turnId}>
 <StoppedResponse stoppedSeconds={row.stoppedSeconds} />
 </TimelineRowFrame>
 );
 }
}, areTimelineRowPropsEqual);

function areTimelineRowPropsEqual(
 previous: {
 actions: ConversationTimelineActions;
 agentRuns: AgentRun[];
 onOpenSession?: (sessionId: string) => void;
 row: ConversationTimelineRow;
 },
 next: {
 actions: ConversationTimelineActions;
 agentRuns: AgentRun[];
 onOpenSession?: (sessionId: string) => void;
 row: ConversationTimelineRow;
 },
) {
 return (
 sameTimelineRow(previous.row, next.row) &&
 previous.actions === next.actions &&
 previous.onOpenSession === next.onOpenSession &&
 (!timelineRowUsesAgentRuns(previous.row) ||
 sameAgentRuns(previous.agentRuns, next.agentRuns))
 );
}

function timelineRowUsesAgentRuns(row: ConversationTimelineRow) {
 return row.type === "tool-group" && row.group.kind === "delegate";
}

function sameTimelineRow(
 previous: ConversationTimelineRow,
 next: ConversationTimelineRow,
) {
 if (previous === next) return true;
 if (previous.type !== next.type || previous.key !== next.key) return false;

 switch (previous.type) {
 case "turn-gap":
 return previous.turnId === (next as typeof previous).turnId;
 case "user-message":
 return previous.turn === (next as typeof previous).turn;
 case "assistant-preamble":
 return (
 previous.turnId === (next as typeof previous).turnId &&
 previous.text === (next as typeof previous).text
 );
 case "tool-group": {
 const nextGroup = (next as typeof previous).group;
 return (
 previous.turnId === (next as typeof previous).turnId &&
 previous.group.id === nextGroup.id &&
 previous.group.kind === nextGroup.kind &&
 previous.group.title === nextGroup.title &&
 sameToolCalls(previous.group.calls, nextGroup.calls)
 );
 }
 case "assistant-status":
 case "assistant-response":
 return previous.turn === (next as typeof previous).turn;
 case "system-note":
 return (
 previous.turnId === (next as typeof previous).turnId &&
 previous.note === (next as typeof previous).note
 );
 case "thinking":
 return (
 previous.turnId === (next as typeof previous).turnId &&
 previous.actionHeading === (next as typeof previous).actionHeading &&
 previous.showSkeleton === (next as typeof previous).showSkeleton
 );
 case "stopped":
 return (
 previous.turnId === (next as typeof previous).turnId &&
 previous.stoppedSeconds === (next as typeof previous).stoppedSeconds
 );
 }
}

function sameAgentRuns(a: AgentRun[], b: AgentRun[]) {
  if (a.length !== b.length) return false;
  return a.every((item, index) => {
    const other = b[index];
    return (
      other &&
      item.id === other.id &&
      item.sessionId === other.sessionId &&
      item.status === other.status &&
      item.result === other.result &&
      item.error === other.error &&
      item.timeUpdated === other.timeUpdated
    );
  });
}

function TimelineRowFrame({
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

function SystemNoteRow({ note }: { note: ConversationSystemNote }) {
  return (
    <div className="my-2 flex justify-center">
      <div className="max-w-[min(90%,34rem)] rounded-md border border-border/70 bg-muted/35 px-3 py-2 text-center text-xs text-muted-foreground">
        {note.content}
      </div>
    </div>
  );
}

const UserMessageRow = memo(function UserMessageRow({
 actions,
 turn,
}: {
 actions: ConversationTimelineActions;
 turn: ConversationTurn;
}) {
 const [userMessageExpanded, setUserMessageExpanded] = useState(false);
 const [userMessageContentHeight, setUserMessageContentHeight] = useState<
 number | null
 >(null);
 const userMessageContentRef = useRef<HTMLDivElement>(null);
 const userMessageExpandable = isExpandableUserMessage(turn.prompt);
 const userMessageAnimatedHeight =
 userMessageContentHeight === null
 ? undefined
 : userMessageExpanded
 ? userMessageContentHeight
 : Math.min(userMessageContentHeight, COLLAPSED_USER_MESSAGE_HEIGHT);

 useLayoutEffect(() => {
 if (!userMessageExpandable) return;

 const contentElement = userMessageContentRef.current;
 if (!contentElement) return;

 const updateContentHeight = () => {
 const nextHeight = contentElement.scrollHeight;
 setUserMessageContentHeight((current) =>
 current === nextHeight ? current : nextHeight,
 );
 };

 const frame = requestAnimationFrame(updateContentHeight);
 const observer = new ResizeObserver(updateContentHeight);
 observer.observe(contentElement);

 return () => {
 cancelAnimationFrame(frame);
 observer.disconnect();
 };
 }, [turn.prompt, userMessageExpandable]);

 return (
 <TimelineRowFrame role="user" turnId={turn.id}>
 <div className="group/user-message ml-auto flex  flex-col items-end">
 {turn.attachments?.length ? (
 <UserMessageAttachments attachments={turn.attachments} />
 ) : null}
 <div
 className={cn(
 "max-w-[min(90%,42rem)] rounded-xl bg-card px-4 py-3 text-sm text-card-foreground shadow-lg shadow-foreground/5 sm:px-5",
 !userMessageExpandable &&
 "flex min-h-[52px] items-center justify-center",
 )}
 >
 <div
 className={cn(
 "whitespace-pre-wrap break-words text-left",
 userMessageExpandable &&
 "overflow-hidden transition-[height] duration-300 ease-out",
 !userMessageExpandable && "line-clamp-3",
 )}
 style={
 userMessageExpandable && userMessageAnimatedHeight !== undefined
 ? { height: `${userMessageAnimatedHeight}px` }
 : undefined
 }
 >
 {userMessageExpandable ? (
 <div ref={userMessageContentRef}>{turn.prompt}</div>
 ) : (
 turn.prompt
 )}
 </div>
 {userMessageExpandable ? (
 <Button
 className="mt-2 h-auto gap-1 px-0 py-0 text-xs  text-muted-foreground hover:bg-transparent hover:text-foreground"
 onClick={() => setUserMessageExpanded((expanded) => !expanded)}
 type="button"
 variant="ghost"
 >
 {userMessageExpanded ? "收起" : "显示更多"}
 <ChevronDown
 className={cn(
 "size-3.5 transition-transform",
 userMessageExpanded && "rotate-180",
 )}
 />
 </Button>
 ) : null}
 </div>
 <div className="mt-1.5 flex items-center gap-2 opacity-0 transition-opacity duration-200 ease-out group-hover/user-message:opacity-100 group-focus-within/user-message:opacity-100">
 <span className="text-sm ">
 {formatCompletionTime(turn.submittedAt)}
 </span>
 <CopyTextButton ariaLabel="复制消息" text={turn.prompt} />
 {turn.userEventId && actions.onEditUserMessage ? (
 <Button
 aria-label="编辑消息"
 onClick={() => actions.onEditUserMessage?.(turn)}
 size="icon-sm"
 title="编辑消息"
 type="button"
 variant="ghost"
 >
 <Pencil />
 </Button>
 ) : null}
 {(turn.userEventId || turn.assistantEventId) && actions.onDeleteTurn ? (
 <Button
 aria-label="删除本轮"
 onClick={() => actions.onDeleteTurn?.(turn)}
 size="icon-sm"
 title="删除本轮"
 type="button"
 variant="ghost"
 >
 <Trash2 />
 </Button>
 ) : null}
 </div>
 </div>
 </TimelineRowFrame>
 );
}, areConversationTurnPropsEqual);

function UserMessageAttachments({
 attachments,
}: {
 attachments: ConversationUserAttachment[];
}) {
 return (
 <div className="mb-2 flex max-w-[min(90%,42rem)] flex-wrap justify-end gap-2">
 {attachments.map((attachment) =>
 attachment.kind === "image" ? (
 <Attachment
 className="size-24 overflow-hidden bg-card p-0 shadow-lg shadow-foreground/5"
 key={attachment.id}
 orientation="vertical"
 size="sm"
 >
 {attachment.previewUrl ? (
 <img
 alt={attachment.name}
 className="absolute inset-0 size-full object-cover"
 src={attachment.previewUrl}
 />
 ) : (
 <div className="absolute inset-0 flex items-center justify-center bg-muted text-muted-foreground">
 <Image />
 </div>
 )}
 </Attachment>
 ) : (
 <Attachment
 className="max-w-52 bg-card shadow-lg shadow-foreground/5"
 key={attachment.id}
 orientation="horizontal"
 size="default"
 >
 <AttachmentMedia variant="icon">
 <File />
 </AttachmentMedia>
 <AttachmentContent>
 <AttachmentTitle>{attachment.name}</AttachmentTitle>
 <AttachmentDescription>
 {formatTimelineAttachmentMeta(attachment)}
 </AttachmentDescription>
 </AttachmentContent>
 </Attachment>
 ),
 )}
 </div>
 );
}

function formatTimelineAttachmentMeta(attachment: ConversationUserAttachment) {
 const type =
 attachment.kind === "image"
 ? "图片"
 : readableTimelineAttachmentType(attachment.mimeType);
 return attachment.size === undefined
 ? type
 : `${type} · ${formatTimelineBytes(attachment.size)}`;
}

function readableTimelineAttachmentType(mimeType: string) {
 if (mimeType === "application/pdf") return "PDF";
 if (mimeType.startsWith("text/")) return "文本";
 if (mimeType.includes("json")) return "JSON";
 if (mimeType.includes("csv")) return "CSV";
 if (mimeType === "application/octet-stream") return "文件";
 return mimeType.split("/").at(-1)?.toUpperCase() || "文件";
}

function formatTimelineBytes(size: number) {
 if (size < 1024) return `${size} B`;
 if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
 return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function areConversationTurnPropsEqual(
 previous: { actions: ConversationTimelineActions; turn: ConversationTurn },
 next: { actions: ConversationTimelineActions; turn: ConversationTurn },
) {
 return previous.turn === next.turn && previous.actions === next.actions;
}

function AssistantPreamble({ text }: { text: string }) {
 return (
 <div className="animate-in fade-in slide-in-from-bottom-2 text-sm duration-300">
 <Markdown content={text} isFinished />
 </div>
 );
}

const TimelineToolGroup = memo(function TimelineToolGroup({
 agentRuns,
 group,
 onOpenSession,
}: {
 agentRuns: AgentRun[];
 group: ToolCallGroup;
 onOpenSession?: (sessionId: string) => void;
}) {
 const [open, setOpen] = useState(false);

 if (group.kind === "delegate") {
 const delegateCalls = uniqueDelegateToolCalls(group.calls, agentRuns);
 return (
 <div
 className="flex flex-col gap-2 py-1"
 data-assistant-hover-ignore="true"
 >
 {delegateCalls.map((toolCall) => (
 <SubagentToolCard
 agentRun={findAgentRunForToolCall(toolCall, agentRuns)}
 key={toolCall.id}
 onOpenSession={onOpenSession}
 toolCall={toolCall}
 />
 ))}
 </div>
 );
 }

 return (
 <div
 className="flex  flex-col overflow-hidden rounded-md border-0 bg-transparent"
 data-assistant-hover-ignore="true"
 >
 <ToolCallGroupView
 group={group}
 onToggle={() => setOpen((current) => !current)}
 open={open}
 />
 </div>
 );
}, areTimelineToolGroupPropsEqual);

function areTimelineToolGroupPropsEqual(
 previous: {
 agentRuns: AgentRun[];
 group: ToolCallGroup;
 onOpenSession?: (sessionId: string) => void;
 },
 next: {
 agentRuns: AgentRun[];
 group: ToolCallGroup;
 onOpenSession?: (sessionId: string) => void;
 },
) {
 return (
 previous.group.id === next.group.id &&
 previous.group.kind === next.group.kind &&
 previous.group.title === next.group.title &&
 previous.onOpenSession === next.onOpenSession &&
 (previous.group.kind !== "delegate" ||
 sameAgentRuns(previous.agentRuns, next.agentRuns)) &&
 sameToolCalls(previous.group.calls, next.group.calls)
 );
}

const ToolCallGroupView = memo(function ToolCallGroupView({
 group,
 onToggle,
 open,
}: {
 group: ToolCallGroup;
 onToggle: (groupId: string) => void;
 open: boolean;
}) {
 return (
 <div className="border-0 bg-transparent data-open:bg-transparent">
 <button
 aria-expanded={open}
 className="group/accordion-trigger relative flex flex-none items-start justify-between gap-2 border border-transparent px-0 py-1 text-left text-sm text-muted-foreground transition-all outline-none hover:no-underline disabled:pointer-events-none disabled:opacity-50"
 onClick={() => onToggle(group.id)}
 type="button"
 >
 <span>{group.title}</span>
 <ChevronDown
 className={cn(
 "mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform",
 open && "rotate-180",
 )}
 />
 </button>
 <AnimatedDisclosure open={open}>
 <div className="pt-0 pb-1.5 pl-3 [&_a]:underline [&_a]:underline-offset-3 [&_a]:hover:text-foreground [&_p:not(:last-child)]:mb-4">
 {group.calls.map((toolCall) => (
 <ToolCallCommandLine key={toolCall.id} toolCall={toolCall} />
 ))}
 </div>
 </AnimatedDisclosure>
 </div>
 );
}, areToolCallGroupPropsEqual);

function areToolCallGroupPropsEqual(
 previous: {
 group: ToolCallGroup;
 open: boolean;
 onToggle: (groupId: string) => void;
 },
 next: {
 group: ToolCallGroup;
 open: boolean;
 onToggle: (groupId: string) => void;
 },
) {
 return (
 previous.open === next.open &&
 previous.group.id === next.group.id &&
 previous.group.title === next.group.title &&
 sameToolCalls(previous.group.calls, next.group.calls)
 );
}

function SubagentToolCard({
  agentRun,
  onOpenSession,
  toolCall,
}: {
  agentRun?: AgentRun;
  onOpenSession?: (sessionId: string) => void;
  toolCall: domain.ToolCall;
}) {
  const sessionId = agentRun?.sessionId ?? delegateToolCallSessionId(toolCall);
  const status = agentRun?.status ?? toolCall.status;
  const prompt =
    agentRun?.prompt ||
    stringArg(toolCall.arguments ?? {}, "prompt") ||
    stringArg(toolCall.arguments ?? {}, "goal") ||
    "子代理任务";
  const title =
    stringArg(toolCall.arguments ?? {}, "title") || truncateInline(prompt, 72);
  const mode =
    agentRun?.mode ||
    stringArg(toolCall.arguments ?? {}, "mode") ||
    "assistant";
  const clickable = Boolean(sessionId && onOpenSession);
  const modeLabel = agentModeDisplayName(mode);
  const statusLabel = subagentStatusLabel(status);

  return (
    <button
      aria-label={`子代理 ${modeLabel}：${title}，${statusLabel}`}
      className={cn(
        "group/subagent inline-flex max-w-full items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-left text-sm shadow-sm transition-colors",
        clickable && "hover:bg-muted/50",
        !clickable && "cursor-default opacity-80",
      )}
      disabled={!clickable}
      onClick={() => {
        if (sessionId) onOpenSession?.(sessionId);
      }}
      title={`${modeLabel} · ${title} · ${statusLabel}`}
      type="button"
    >
      <span className="shrink-0 font-semibold text-amber-600 dark:text-amber-400">
        {modeLabel}
      </span>
      <span className="min-w-0 truncate text-foreground">
        {title}
      </span>
      <span className={cn("shrink-0 text-xs", subagentStatusClass(status))}>
        {statusLabel}
      </span>
      {clickable ? (
        <span className="-mr-1 flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity group-hover/subagent:opacity-100 group-focus-visible/subagent:opacity-100">
          <ExternalLink className="size-3" />
        </span>
      ) : null}
    </button>
  );
}

function ToolCallCommandLine({ toolCall }: { toolCall: domain.ToolCall }) {
 const failed = toolCall.status === "failed";
 const running = toolCall.status === "running";
 const pendingApproval = toolCall.status === "pending_approval";
 const commandTool = isCommandToolCall(toolCall);
 const command = getToolCallCommand(toolCall);
 const fileChanges = getToolCallFileChanges(toolCall);
 const retainedRefs = getRetainedOutputRefs(toolCall);
 const retainedRef = retainedRefs[0] ?? "";
 const retainedRefsKey = retainedRefs.join("\n");
 const showFileChanges = toolCallKind(toolCall) === "write" && fileChanges.length > 0;
 const resultText =
 toolCall.name === "list_files" || commandTool || failed
 ? getToolResultText(toolCall)
 : "";
 const showCommandLine = commandTool || (!resultText && !showFileChanges);
 const showRunningStatus = running && !showFileChanges;
 const showStatusLine = showRunningStatus || pendingApproval || failed;
 const failedRunDetail =
 failed && !showCommandLine ? failedToolRunDetail(toolCall, resultText) : "";
 const resultDetailsExpandable = Boolean(
 resultText && (failed || commandTool) && retainedRefs.length === 0,
 );
 const [resultDetailsOpen, setResultDetailsOpen] = useState(false);
 const [retainedOutput, setRetainedOutput] =
 useState<RetainedOutputViewState | null>(null);
 const hasRetainedOutputRef = retainedRefs.length > 0;
 const showInlineResult = Boolean(
 resultText &&
 (!resultDetailsExpandable || resultDetailsOpen) &&
 !hasRetainedOutputRef,
 );

 useEffect(() => {
 let cancelled = false;
 setResultDetailsOpen(false);
 setRetainedOutput(null);
 const ref = retainedRef;
 if (ref) {
 void loadRetainedOutput(ref, () => cancelled);
 }
 return () => {
 cancelled = true;
 };
 }, [toolCall.id, resultText, retainedRef, retainedRefsKey]);

 async function loadRetainedOutput(
 ref: string,
 isCancelled: () => boolean,
 ) {
 let content = "";
 let nextOffset = 0;
 let size = 0;
 let truncated = true;
 setRetainedOutput({
 ref,
 content,
 nextOffset,
 size,
 truncated: false,
 loading: true,
 });
 try {
 while (!isCancelled() && truncated) {
 const offset = nextOffset;
 const chunk = await readRetainedOutput({
 ref,
 offset,
 limit: 100_000,
 });
 content += chunk.content;
 nextOffset = chunk.nextOffset;
 size = chunk.size;
 truncated = Boolean(chunk.truncated) && nextOffset > offset;
 if (isCancelled()) return;
 setRetainedOutput({
 ref: chunk.ref,
 content,
 nextOffset,
 size,
 truncated,
 loading: truncated,
 });
 }
 if (isCancelled()) return;
 setRetainedOutput({
 ref,
 content,
 nextOffset,
 size,
 truncated: false,
 loading: false,
 });
 } catch (error) {
 if (isCancelled()) return;
 setRetainedOutput({
 ref,
 content,
 nextOffset,
 size,
 truncated: false,
 loading: false,
 error: error instanceof Error ? error.message : String(error),
 });
 }
 }

 const statusLineClassName = cn(
 "flex min-w-0 items-baseline gap-2 text-sm",
 failed && "text-destructive",
 running && "text-muted-foreground",
 pendingApproval && "text-amber-700 dark:text-amber-300",
 resultDetailsExpandable && "cursor-pointer rounded-sm hover:bg-muted/50",
 );
 const statusLineContent = (
 <>
 {showCommandLine ? (
 <>
 <span className="shrink-0 text-foreground">
 {command.label}
 </span>
 <span className="min-w-0 truncate text-muted-foreground">
 {command.detail}
 </span>
 </>
 ) : null}
 {showRunningStatus ? (
 <span className="shrink-0 text-muted-foreground">运行中</span>
 ) : null}
 {pendingApproval ? (
 <span className="shrink-0 text-amber-700 dark:text-amber-300">
 等待批准
 </span>
 ) : null}
 {failedRunDetail ? (
 <>
 <span className="shrink-0 text-muted-foreground">已运行</span>
 <span className="min-w-0 truncate text-muted-foreground">
 {failedRunDetail}
 </span>
 </>
 ) : null}
 {failed ? (
 <span className="shrink-0 text-destructive">失败</span>
 ) : null}
 {resultDetailsExpandable ? (
 <ChevronDown
 className={cn(
 "size-3 shrink-0 self-center text-muted-foreground transition-transform",
 resultDetailsOpen && "rotate-180",
 )}
 />
 ) : null}
 </>
 );

 return (
 <div className="flex min-w-0 flex-col gap-1">
 {showCommandLine || showStatusLine ? (
 resultDetailsExpandable ? (
 <button
 aria-expanded={resultDetailsOpen}
 className={statusLineClassName}
 onClick={() => setResultDetailsOpen((current) => !current)}
 type="button"
 >
 {statusLineContent}
 </button>
 ) : (
 <div className={statusLineClassName}>{statusLineContent}</div>
 )
 ) : null}
 <AnimatedDisclosure open={showFileChanges}>
 <ToolFileChangeLines files={fileChanges} live={running} />
 </AnimatedDisclosure>
 <AnimatedDisclosure
 open={showInlineResult}
 >
 {resultText && commandTool ? (
 <InlineShellPreview resultText={resultText} toolCall={toolCall} />
 ) : resultText ? (
 <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
 {resultText}
 </pre>
 ) : null}
 </AnimatedDisclosure>
 {retainedRefs.length > 0 ? (
 <div className="flex min-w-0 flex-col gap-1">
 {retainedOutput?.loading && !retainedOutput.content ? (
 <div className="text-xs text-muted-foreground">加载完整输出...</div>
 ) : null}
 {retainedOutput?.error ? (
 <div className="text-xs text-destructive">{retainedOutput.error}</div>
 ) : null}
 <AnimatedDisclosure open={Boolean(retainedOutput?.content)}>
 {retainedOutput?.content ? (
 <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
 {retainedOutput.content}
 </pre>
 ) : null}
 </AnimatedDisclosure>
 </div>
 ) : null}
 </div>
 );
}

type RetainedOutputViewState = {
 ref: string;
 content: string;
 nextOffset: number;
 size: number;
 truncated: boolean;
 loading?: boolean;
 error?: string;
};

function ToolFileChangeLines({
 files,
 live,
}: {
 files: ToolFileChange[];
 live: boolean;
}) {
 const [expandedFileKey, setExpandedFileKey] = useState<string | null>(null);
 return (
 <div className="flex min-w-0 flex-col gap-0.5 text-sm">
 {files.map((file) => {
 const fileKey = toolFileChangeKey(file);
 const expanded = expandedFileKey === fileKey;
 const canExpand = Boolean(file.diff?.trim());
 return (
 <div className="flex min-w-0 flex-col gap-1" key={fileKey}>
 <button
 aria-expanded={canExpand ? expanded : undefined}
 className={cn(
 "flex min-w-0 items-baseline gap-1.5 text-left",
 canExpand && "cursor-pointer rounded-sm hover:bg-muted/50",
 )}
 disabled={!canExpand}
 onClick={() =>
 setExpandedFileKey((current) => (current === fileKey ? null : fileKey))
 }
 type="button"
 >
 <span className="shrink-0 text-muted-foreground">
 {toolFileChangeLabel(file, live)}
 </span>
 <span className="min-w-0 truncate text-sky-500 dark:text-sky-300">
 {toolFileChangePath(file)}
 </span>
 <span className="inline-flex shrink-0 items-baseline text-emerald-500 dark:text-emerald-400">
 <span>+</span>
 <RollingCount value={file.additions} live={live} direction="up" />
 </span>
 <span className="inline-flex shrink-0 items-baseline text-rose-500 dark:text-rose-400">
 <span>-</span>
 <RollingCount value={file.deletions} live={live} direction="down" />
 </span>
 {canExpand ? (
 <ChevronDown
 className={cn(
 "size-3 shrink-0 self-center text-muted-foreground transition-transform",
 expanded && "rotate-180",
 )}
 />
 ) : null}
 </button>
 {canExpand ? (
 <AnimatedDisclosure open={expanded}>
 <ToolDiffPreview text={file.diff} />
 </AnimatedDisclosure>
 ) : null}
 </div>
 );
 })}
 </div>
 );
}

function ToolDiffPreview({ text }: { text?: string }) {
 if (!text?.trim()) return null;
 return (
 <div className="ml-2 max-h-80 overflow-hidden rounded-md border border-border/70 bg-muted/35">
 <ScrollArea className="max-h-80 [&_[data-radix-scroll-area-viewport]]:p-2 [&_[data-slot=scroll-area-viewport]]:max-h-80">
 <pre className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">
 {text.split("\n").map((line, index) => (
 <span
 className={cn(
 "block min-h-[1.25em]",
 line.startsWith("+") && !line.startsWith("+++")
 ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
 : line.startsWith("-") && !line.startsWith("---")
 ? "bg-rose-500/10 text-rose-700 dark:text-rose-300"
 : line.startsWith("@@")
 ? "bg-sky-500/10 text-sky-700 dark:text-sky-300"
 : "text-muted-foreground",
 )}
 key={`${index}:${line}`}
 >
 {line || " "}
 </span>
 ))}
 </pre>
 </ScrollArea>
 </div>
 );
}

function InlineShellPreview({
 resultText,
 toolCall,
}: {
 resultText: string;
 toolCall: domain.ToolCall;
}) {
 const entries = shellPreviewEntries(toolCall, resultText);
 if (entries.length === 0) return null;

 return (
 <div className="max-h-56 overflow-auto rounded-md bg-muted/50 px-3 py-2">
 <div className="flex min-w-0 flex-col gap-2">
 {entries.map((entry) => (
 <div
 className="min-w-0 border-b border-border/50 pb-2 last:border-b-0 last:pb-0"
 key={entry.id}
 >
 <div className="mb-1 min-w-0 truncate font-mono text-[11px] text-muted-foreground">
 {entry.toolName}
 {entry.exitCode !== undefined ? ` exit ${entry.exitCode}` : ""}
 </div>
 <pre className="m-0 whitespace-pre-wrap break-all font-mono text-[12px] leading-[1.45] text-foreground [overflow-wrap:anywhere]">
 <span>{shellPrompt(entry.cwd)}</span>
 <span>{entry.command}</span>
 {"\n"}
 {entry.stdout ? <span>{terminalOutputSegment(entry.stdout)}</span> : null}
 {entry.stderr ? (
 <span className="text-destructive">
 {terminalOutputSegment(entry.stderr)}
 </span>
 ) : null}
 {entry.error && !entry.stderr ? (
 <span className="text-destructive">
 {terminalOutputSegment(entry.error)}
 </span>
 ) : null}
 {!entry.stdout && !entry.stderr && !entry.error ? (
 <span className="text-muted-foreground">暂无输出{"\n"}</span>
 ) : null}
 </pre>
 </div>
 ))}
    </div>
  </div>
);
}

type ShellPreviewEntry = {
 command: string;
 cwd?: string;
 error?: string;
 exitCode?: number;
 id: string;
 stderr: string;
 stdout: string;
 toolName: string;
};

function shellPreviewEntries(
 toolCall: domain.ToolCall,
 resultText: string,
): ShellPreviewEntry[] {
 const structured = objectRecord(toolCall.result?.structured);
 if (toolCall.name === "run_tests") {
 const commands = arrayRecords(structured?.commands);
 if (commands.length > 0) {
 return commands.map((command, index) =>
 shellPreviewEntryFromStructured(toolCall, command, index),
 );
 }
 }

 if (structured) {
 return [shellPreviewEntryFromStructured(toolCall, structured, 0)];
 }

 return [shellPreviewEntryFromResultText(toolCall, resultText)];
}

function shellPreviewEntryFromStructured(
 toolCall: domain.ToolCall,
 structured: Record<string, unknown>,
 index: number,
): ShellPreviewEntry {
 const args = toolCall.arguments ?? {};
 return {
 command: stringValue(structured.command) || shellCommandFromToolArgs(toolCall),
 cwd: stringValue(structured.cwd) || stringArg(args, "cwd"),
 error: toolCall.error || stringValue(toolCall.result?.error),
 exitCode: optionalNumberValue(structured.exitCode),
 id: `${toolCall.id}:${index}`,
 stderr: previewInlineShellOutput(stringValue(structured.stderr)),
 stdout: previewInlineShellOutput(stringValue(structured.stdout)),
 toolName: toolCall.name,
 };
}

function shellPreviewEntryFromResultText(
 toolCall: domain.ToolCall,
 resultText: string,
): ShellPreviewEntry {
 const parsed = parseCommandResultText(resultText);
 return {
 command: parsed.command || shellCommandFromToolArgs(toolCall),
 cwd: parsed.cwd || stringArg(toolCall.arguments ?? {}, "cwd"),
 error: toolCall.error || parsed.error || stringValue(toolCall.result?.error),
 exitCode: parsed.exitCode,
 id: `${toolCall.id}:0`,
 stderr: previewInlineShellOutput(parsed.stderr),
 stdout: previewInlineShellOutput(parsed.stdout),
 toolName: toolCall.name,
 };
}

function shellCommandFromToolArgs(toolCall: domain.ToolCall) {
 const args = toolCall.arguments ?? {};
 if (toolCall.name === "bash") {
 return stringArg(args, "command") || "bash";
 }
 if (toolCall.name === "run_tests") {
 return [stringArg(args, "target") || "all", stringArg(args, "kind") || "auto"]
 .filter(Boolean)
 .join(":");
 }
 return toolCall.name || "tool";
}

function parseCommandResultText(text: string) {
 const lines = text.split("\n");
 const parsed: {
 command?: string;
 cwd?: string;
 error?: string;
 exitCode?: number;
 stderr: string;
 stdout: string;
 } = {
 stderr: "",
 stdout: "",
 };
 let stream: "stdout" | "stderr" | null = null;

 for (const line of lines) {
 if (line.startsWith("STDOUT:")) {
 stream = "stdout";
 continue;
 }
 if (line.startsWith("STDERR:")) {
 stream = "stderr";
 continue;
 }
 if (stream === "stdout") {
 parsed.stdout += `${line}\n`;
 continue;
 }
 if (stream === "stderr") {
 parsed.stderr += `${line}\n`;
 continue;
 }
 if (line.startsWith("Command:")) {
 parsed.command = line.replace(/^Command:\s*/, "").trim();
 } else if (line.startsWith("CWD:")) {
 parsed.cwd = line.replace(/^CWD:\s*/, "").trim();
 } else if (line.startsWith("Exit code:")) {
 parsed.exitCode = optionalNumberValue(
 Number(line.replace(/^Exit code:\s*/, "").trim()),
 );
 } else if (line.startsWith("Error:")) {
 parsed.error = line.replace(/^Error:\s*/, "").trim();
 }
 }

 parsed.stdout = parsed.stdout.trimEnd();
 parsed.stderr = parsed.stderr.trimEnd();
 return parsed;
}

function previewInlineShellOutput(text: string) {
 if (text.length <= INLINE_SHELL_OUTPUT_CHARS) return text;
 const omitted = text.length - INLINE_SHELL_OUTPUT_CHARS;
 return `${text.slice(0, INLINE_SHELL_OUTPUT_CHARS).trimEnd()}\n... 已省略 ${omitted.toLocaleString()} 个字符 ...`;
}

function terminalOutputSegment(content: string) {
 return content.endsWith("\n") ? content : `${content}\n`;
}

function shellPrompt(cwd?: string) {
 return `agent@aivo ${shellCwdLabel(cwd)} % `;
}

function shellCwdLabel(cwd?: string) {
 const value = cwd?.trim();
 if (!value) return "~";
 const parts = value.split("/").filter(Boolean);
 return parts.at(-1) || "/";
}

function RollingCount({
 value,
 live,
 direction,
}: {
 value: number;
 live: boolean;
 direction: "up" | "down";
}) {
 const previousValueRef = useRef(value);
 const [roll, setRoll] = useState<{
 from: number;
 to: number;
 active: boolean;
 } | null>(null);

 useEffect(() => {
 const previous = previousValueRef.current;
 if (previous === value) {
 return;
 }
 previousValueRef.current = value;
 if (!live) {
 setRoll(null);
 return;
 }

 setRoll({ from: previous, to: value, active: false });
 const frame = requestAnimationFrame(() => {
 setRoll((current) => (current ? { ...current, active: true } : current));
 });
 const timeout = window.setTimeout(() => setRoll(null), 360);
 return () => {
 cancelAnimationFrame(frame);
 window.clearTimeout(timeout);
 };
 }, [value, live]);

 if (!roll) {
 return (
 <span className="inline-block tabular-nums" style={{ minWidth: `${String(value).length}ch` }}>
 {value}
 </span>
 );
 }

 const width = Math.max(String(roll.from).length, String(roll.to).length);
 const values = direction === "up" ? [roll.from, roll.to] : [roll.to, roll.from];
 const translate = direction === "up"
 ? roll.active
 ? "translateY(-50%)"
 : "translateY(0)"
 : roll.active
 ? "translateY(0)"
 : "translateY(-50%)";

 return (
 <span
 className="inline-block h-[1em] overflow-hidden align-[-0.08em] tabular-nums"
 style={{ minWidth: `${width}ch` }}
 >
 <span
 className="flex flex-col transition-transform duration-300 ease-out motion-reduce:transition-none"
 style={{ transform: translate }}
 >
 {values.map((item, index) => (
 <span className="h-[1em] leading-none" key={`${item}:${index}`}>
 {item}
 </span>
 ))}
 </span>
 </span>
 );
}

type ToolCallGroup = {
 id: string;
 kind: string;
 calls: domain.ToolCall[];
 timeCreated?: string;
 title: string;
};

type ToolFileChange = {
 path: string;
 movePath?: string;
 type: string;
 additions: number;
 deletions: number;
 diff?: string;
};

function toolFileChangeKey(file: ToolFileChange) {
 return `${file.type}:${file.path}:${file.movePath ?? ""}`;
}

function constructConversationTimelineRows(
 turns: ConversationTurn[],
): ConversationTimelineRow[] {
 return turns.flatMap((turn, index) => {
 const rows: ConversationTimelineRow[] = [];
 if (index > 0) {
 rows.push({
 key: `turn-gap:${turn.id}`,
 turnId: turn.id,
 type: "turn-gap",
 });
 }

 rows.push({
 key: `user-message:${turn.id}`,
 turn,
 type: "user-message",
 });

 const preambleParts = assistantPreambleParts(turn);
 const toolGroups = groupToolCalls(
 filterVisibleToolCalls(turn.toolCalls),
 preambleParts,
 );
 const hasVisibleToolCalls = toolGroups.length > 0;
 const hasPreambleText = preambleParts.some((part) => part.text.trim());
 pushAssistantActivityRows(rows, turn, preambleParts, toolGroups);

  if (turn.stopped) {
 rows.push({
 key: `stopped:${turn.id}`,
 stoppedSeconds: turn.thinkingSeconds,
 turnId: turn.id,
 type: "stopped",
 });
 pushSystemNotes(rows, turn);
 return rows;
 }

 if (turn.responseVisible || turn.responseText.trim()) {
 rows.push({
 key: `assistant-status:${turn.id}`,
 turn,
 type: "assistant-status",
 });
 if (turn.responseText.trim()) {
 rows.push({
 key: `assistant-response:${turn.id}`,
 turn,
 type: "assistant-response",
 });
 }
 pushSystemNotes(rows, turn);
 return rows;
 }

 rows.push({
 actionHeading: hasVisibleToolCalls ? undefined : toolActionHeading(toolGroups),
 key: `thinking:${turn.id}`,
 showSkeleton:
 !turn.activityVisible && !hasPreambleText && !hasVisibleToolCalls,
 turnId: turn.id,
 type: "thinking",
 });
 pushSystemNotes(rows, turn);
 return rows;
 });
}

function assistantPreambleParts(
 turn: ConversationTurn,
): ConversationAssistantTextPart[] {
 const parts = (turn.assistantPreambles ?? []).filter((part) =>
 part.text.trim(),
 );
 if (parts.length > 0) return parts;
 if (!turn.preToolText.trim()) return [];
 return [
 {
 id: `legacy-preamble:${turn.id}`,
 text: turn.preToolText,
 timeCreated: undefined,
 },
 ];
}

function pushAssistantActivityRows(
 rows: ConversationTimelineRow[],
 turn: ConversationTurn,
 preambleParts: ConversationAssistantTextPart[],
 toolGroups: ToolCallGroup[],
) {
 if (preambleParts.length === 0) {
 pushToolGroups(rows, turn, toolGroups);
 return;
 }

 const activityItems = [
 ...preambleParts.map((part, index) => ({
 index,
 key: `assistant-preamble:${turn.id}:${part.id}`,
 kind: "text" as const,
 part,
 sortTime: timelineSortTime(part.timeCreated, index),
 })),
 ...toolGroups.map((group, index) => ({
 index,
 group,
 key: `tool-group:${turn.id}:${group.id}`,
 kind: "tool" as const,
 sortTime: timelineSortTime(group.timeCreated, preambleParts.length + index),
 })),
 ].toSorted((a, b) => {
 const timeDelta = a.sortTime - b.sortTime;
 if (timeDelta !== 0) return timeDelta;
 if (a.kind !== b.kind) return a.kind === "text" ? -1 : 1;
 return a.index - b.index;
 });

 for (const item of activityItems) {
 if (item.kind === "text") {
 rows.push({
 key: item.key,
 text: item.part.text,
 turnId: turn.id,
 type: "assistant-preamble",
 });
 continue;
 }
 rows.push({
 group: item.group,
 key: item.key,
 turnId: turn.id,
 type: "tool-group",
 });
 }
}

function timelineSortTime(value: string | undefined, fallbackOffset: number) {
 if (!value) return Number.MIN_SAFE_INTEGER + fallbackOffset;
 const time = Date.parse(value);
 if (Number.isNaN(time)) return Number.MIN_SAFE_INTEGER + fallbackOffset;
 return time;
}

function pushSystemNotes(rows: ConversationTimelineRow[], turn: ConversationTurn) {
 for (const note of turn.systemNotes ?? []) {
 if (!note.content.trim()) continue;
 rows.push({
 key: `system-note:${turn.id}:${note.id}`,
 note,
 turnId: turn.turnId ?? turn.id,
 type: "system-note",
 });
 }
}

function pushToolGroups(
 rows: ConversationTimelineRow[],
 turn: ConversationTurn,
 toolGroups: ToolCallGroup[],
) {
 for (const group of toolGroups) {
 rows.push({
 group,
 key: `tool-group:${turn.id}:${group.id}`,
 turnId: turn.id,
 type: "tool-group",
 });
 }
}

function toolActionHeading(toolGroups: ToolCallGroup[]) {
 if (toolGroups.length === 0) return undefined;
 const activeGroup =
 toolGroups.find((group) =>
 group.calls.some(
 (call) =>
 call.status === "running" || call.status === "pending_approval",
 ),
 ) ?? toolGroups.at(-1);
 return activeGroup?.title;
}

function groupToolCalls(
 toolCalls: domain.ToolCall[],
 separators: ConversationAssistantTextPart[] = [],
): ToolCallGroup[] {
 const groups: ToolCallGroup[] = [];
 const separatorTimes = separators
 .map((part) => Date.parse(part.timeCreated ?? ""))
 .filter((time) => !Number.isNaN(time))
 .toSorted((a, b) => a - b);
 for (const toolCall of toolCalls) {
 const kind = toolCallKind(toolCall);
 const last = groups.at(-1);
 if (
 last?.kind === kind &&
 !hasSeparatorBetweenToolCalls(separatorTimes, last.calls.at(-1), toolCall)
 ) {
 last.calls.push(toolCall);
 last.title = toolGroupTitle(kind, last.calls);
 continue;
 }
 groups.push({
 id: `${kind}:${toolCall.id}`,
 kind,
 calls: [toolCall],
 timeCreated: toolCall.timeCreated,
 title: toolGroupTitle(kind, [toolCall]),
 });
 }
 return groups;
}

function hasSeparatorBetweenToolCalls(
 separatorTimes: number[],
 previous: domain.ToolCall | undefined,
 next: domain.ToolCall,
) {
 if (!previous || separatorTimes.length === 0) return false;
 const previousTime = Date.parse(previous.timeCreated ?? "");
 const nextTime = Date.parse(next.timeCreated ?? "");
 if (Number.isNaN(previousTime) || Number.isNaN(nextTime)) return false;
 return separatorTimes.some((time) => time > previousTime && time <= nextTime);
}

function toolCallKind(toolCall: domain.ToolCall) {
 switch (toolCall.name) {
 case "read_file":
 return "read";
 case "glob":
 case "search_files":
 return "search";
 case "list_files":
 return "list";
 case "tool_resolve":
 return "tool-resolve";
 case "tool_search":
 return "tool-search";
 case "tool_list":
 return "tool-list";
 case "tool_detail":
 return "tool-detail";
 case "tool_call":
 return "tool-bridge";
 case "apply_patch":
 case "write_file":
 case "edit_file":
 return "write";
 case "git_status":
 case "git_diff":
 return "git";
case "bash":
case "run_tests":
 return "shell";
 case "agent_delegate_task":
 return "delegate";
 default:
 return "tool";
 }
}

function filterVisibleToolCalls(toolCalls: domain.ToolCall[]) {
 const visible: domain.ToolCall[] = [];
 const laterGlobPatterns: string[] = [];

 for (let index = toolCalls.length - 1; index >= 0; index -= 1) {
 const toolCall = toolCalls[index];
 if (hiddenToolCallNames.has(toolCall.name)) continue;

 if (toolCall.name === "glob") {
 const pattern = stringArg(toolCall.arguments ?? {}, "pattern")
 .trim()
 .toLowerCase();
 if (pattern) laterGlobPatterns.push(pattern);
 visible.push(toolCall);
 continue;
 }

 if (toolCall.name === "search_files") {
 const query = stringArg(toolCall.arguments ?? {}, "query")
 .trim()
 .toLowerCase();
 if (query && laterGlobPatterns.some((pattern) => pattern.includes(query))) {
 continue;
 }
 }

 visible.push(toolCall);
 }

 return visible.reverse();
}

const hiddenToolCallNames = new Set(["update_plan"]);

function toolGroupTitle(kind: string, calls: domain.ToolCall[]) {
 const count = calls.length;
 switch (kind) {
 case "read":
 return `已探索 ${count} 次读取`;
 case "search":
 return `已探索 ${count} 次搜索`;
 case "list":
 return `已探索 ${count} 次列出`;
 case "tool-resolve":
 return `已解析 ${count} 次工具`;
 case "tool-search":
 return `已搜索 ${count} 次工具`;
 case "tool-list":
 return `已列出 ${count} 次工具`;
 case "tool-detail":
 return `已查看 ${count} 次工具详情`;
 case "tool-bridge":
 return `已调用 ${count} 次工具`;
 case "write":
 return writeToolGroupTitle(calls);
 case "git":
 return `已检查 ${count} 次 Git`;
 case "shell":
 return shellToolGroupTitle(calls);
 case "delegate":
 return count === 1 ? "已启动 1 个子代理" : `已启动 ${count} 个子代理`;
 default:
 return `已探索 ${count} 次工具调用`;
 }
}

function shellToolGroupTitle(calls: domain.ToolCall[]) {
 const failed = calls.some((call) => call.status === "failed");
 const pending = calls.some((call) => call.status === "pending_approval");
 if (pending) return `等待批准 ${calls.length} 条命令`;
 if (failed) return `已运行 ${calls.length} 条命令，存在失败`;
 return `已运行 ${calls.length} 条命令`;
}

function writeToolGroupTitle(calls: domain.ToolCall[]) {
 const files = uniqueToolFileChanges(calls.flatMap(getToolCallFileChanges));
 if (files.length === 0) return `已请求 ${calls.length} 次写入`;
 const labels = new Set(files.map((file) => toolFileChangeLabel(file)));
 const label = labels.size === 1 ? [...labels][0] : "已更新";
 return `${label} ${files.length} 个文件`;
}

function getToolCallFileChanges(toolCall: domain.ToolCall): ToolFileChange[] {
 const resultFiles = parseToolFileChanges(toolCall.result?.files);
 if (resultFiles.length > 0) return resultFiles;
 return parseToolFileChanges(toolCall.arguments?.files);
}

function parseToolFileChanges(value: unknown): ToolFileChange[] {
 if (!Array.isArray(value)) return [];
 return value.flatMap((file) => {
 if (!file || typeof file !== "object") return [];
 const record = file as Record<string, unknown>;
 const path = typeof record.path === "string" ? record.path : "";
 if (!path) return [];
 return [
 {
 path,
 movePath:
 typeof record.movePath === "string" ? record.movePath : undefined,
 type: typeof record.type === "string" ? record.type : "update",
 additions: numberValue(record.additions),
 deletions: numberValue(record.deletions),
 diff: typeof record.diff === "string" ? record.diff : undefined,
 },
 ];
 });
}

function numberValue(value: unknown) {
 return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function optionalNumberValue(value: unknown) {
 return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function uniqueToolFileChanges(files: ToolFileChange[]) {
 const unique = new Map<string, ToolFileChange>();
 for (const file of files) {
 unique.set(`${file.path}\u0000${file.movePath ?? ""}`, file);
 }
 return [...unique.values()];
}

function toolFileChangeLabel(file: ToolFileChange, live = false) {
 switch (file.type) {
 case "add":
 return live ? "创建中" : "已创建";
 case "delete":
 return live ? "删除中" : "已删除";
 case "move":
 return live ? "移动中" : "已移动";
 default:
 return live ? "编辑中" : "已编辑";
 }
}

function toolFileChangePath(file: ToolFileChange) {
 return file.movePath ? `${file.path} -> ${file.movePath}` : file.path;
}

function getToolCallCommand(toolCall: domain.ToolCall) {
 const args = toolCall.arguments ?? {};
 switch (toolCall.name) {
 case "read_file":
 return {
 label: "读取",
 detail: joinCommandParts([
 stringArg(args, "path"),
 ...visibleToolArgs(args, ["path"]),
 ]),
 };
 case "list_files":
 return {
 label: "列出",
 detail: joinCommandParts([
 stringArg(args, "path"),
 ...visibleToolArgs(args, ["path"]),
 ]),
 };
 case "glob":
 return {
 label: "Glob",
 detail: joinCommandParts([
 stringArg(args, "path"),
 stringArg(args, "pattern")
 ? `pattern=${stringArg(args, "pattern")}`
 : "",
 ...visibleToolArgs(args, ["path", "pattern"]),
 ]),
 };
 case "search_files":
 return {
 label: "搜索",
 detail: joinCommandParts([
 stringArg(args, "query"),
 ...visibleToolArgs(args, ["query"]),
 ]),
 };
 case "tool_resolve":
 return {
 label: "解析工具",
 detail: joinCommandParts([
 stringArg(args, "intent"),
 scalarArg(args, "maxTools") ? `maxTools=${scalarArg(args, "maxTools")}` : "",
 stringArg(args, "category") ? `category=${stringArg(args, "category")}` : "",
 ...visibleToolArgs(args, ["intent", "maxTools", "category", "source", "riskLevel", "required"]),
 ]),
 };
 case "tool_search":
 return {
 label: "搜索工具",
 detail: joinCommandParts([
 stringArg(args, "query"),
 scalarArg(args, "limit") ? `limit=${scalarArg(args, "limit")}` : "",
 ...visibleToolArgs(args, ["query", "limit"]),
 ]),
 };
 case "tool_list":
 return {
 label: "列出工具",
 detail: joinCommandParts([
 stringArg(args, "source") ? `source=${stringArg(args, "source")}` : "",
 stringArg(args, "category") ? `category=${stringArg(args, "category")}` : "",
 stringArg(args, "query"),
 scalarArg(args, "limit") ? `limit=${scalarArg(args, "limit")}` : "",
 ...visibleToolArgs(args, ["source", "category", "query", "limit", "offset"]),
 ]),
 };
 case "tool_detail":
 return {
 label: "工具详情",
 detail: joinCommandParts([
 stringArg(args, "name"),
 ...visibleToolArgs(args, ["name"]),
 ]),
 };
 case "tool_call":
 return {
 label: "调用工具",
 detail: joinCommandParts([
 stringArg(args, "name"),
 ...visibleToolArgs(args, ["name", "arguments"]),
 ]),
 };
 case "apply_patch":
 return {
 label: "补丁",
 detail: joinCommandParts([
 patchSummary(args),
 ...visibleToolArgs(args, ["patchText"]),
 ]),
 };
 case "write_file":
 return {
 label: "写入",
 detail: joinCommandParts([
 stringArg(args, "path"),
 ...visibleToolArgs(args, ["path", "content"]),
 ]),
 };
 case "edit_file":
 return {
 label: "编辑",
 detail: joinCommandParts([
 stringArg(args, "path"),
 ...visibleToolArgs(args, [
 "path",
 "oldString",
 "newString",
 "replaceAll",
 ]),
 ]),
 };
 case "git_status":
 return {
 label: "Git status",
 detail: "",
 };
 case "git_diff":
 return {
 label: "Git diff",
 detail: joinCommandParts([
 stringArg(args, "path"),
 ...visibleToolArgs(args, ["path"]),
 ]),
 };
 case "bash":
 return {
 label: "Bash",
 detail: joinCommandParts([
 stringArg(args, "command") || stringArg(args, "normalizedCommand"),
 stringArg(args, "cwd") ? `cwd=${stringArg(args, "cwd")}` : "",
 ]),
 };
 case "run_tests":
 return {
 label: "Run tests",
 detail: joinCommandParts([
 stringArg(args, "command") ||
 [stringArg(args, "target"), stringArg(args, "kind")]
 .filter(Boolean)
 .join(":"),
 ]),
 };
 default:
 return {
 label: toolCall.name || "工具",
 detail: joinCommandParts(toolCallArgumentLabels(args)),
 };
 }
}

function isCommandToolCall(toolCall: domain.ToolCall) {
 return toolCall.name === "bash" || toolCall.name === "run_tests";
}

function findAgentRunForToolCall(
  toolCall: domain.ToolCall,
  agentRuns: AgentRun[],
) {
  const runByToolCallId = agentRuns.find(
    (run) => run.metadata?.toolCallId === toolCall.id,
  );
  if (runByToolCallId) return runByToolCallId;

  const embeddedRun = delegateToolCallAgentRun(toolCall);
  if (embeddedRun?.id) {
    return agentRuns.find((run) => run.id === embeddedRun.id) ?? embeddedRun;
  }
  const sessionId = delegateToolCallSessionId(toolCall);
  if (sessionId) {
    return agentRuns.find((run) => run.sessionId === sessionId);
  }
  const prompt = stringArg(toolCall.arguments ?? {}, "prompt");
  const mode = stringArg(toolCall.arguments ?? {}, "mode");
  const matchingRuns = agentRuns.filter(
    (run) =>
      (!prompt || run.prompt === prompt) && (!mode || run.mode === mode),
  );
  return matchingRuns.length === 1 ? matchingRuns[0] : undefined;
}

function uniqueDelegateToolCalls(
  toolCalls: domain.ToolCall[],
  agentRuns: AgentRun[],
) {
  const callsByKey = new Map<string, domain.ToolCall>();
  for (const toolCall of toolCalls) {
    const key = delegateToolCallDisplayKey(toolCall, agentRuns);
    const existing = callsByKey.get(key);
    callsByKey.set(
      key,
      existing ? preferredDelegateToolCall(existing, toolCall) : toolCall,
    );
  }
  return [...callsByKey.values()];
}

function delegateToolCallDisplayKey(
  toolCall: domain.ToolCall,
  agentRuns: AgentRun[],
) {
  const agentRun = findAgentRunForToolCall(toolCall, agentRuns);
  if (agentRun?.id) return `run:${agentRun.id}`;
  if (agentRun?.sessionId) return `session:${agentRun.sessionId}`;
  const embeddedRun = delegateToolCallAgentRun(toolCall);
  if (embeddedRun?.id) return `run:${embeddedRun.id}`;
  if (embeddedRun?.sessionId) return `session:${embeddedRun.sessionId}`;
  return `call:${toolCall.id}`;
}

function preferredDelegateToolCall(
  current: domain.ToolCall,
  next: domain.ToolCall,
) {
  const currentRun = delegateToolCallAgentRun(current);
  const nextRun = delegateToolCallAgentRun(next);
  if (!currentRun && nextRun) return next;
  if (current.status === "running" && next.status !== "running") return next;
  if (!current.result && next.result) return next;
  if (parseTime(next.timeUpdated).getTime() > parseTime(current.timeUpdated).getTime()) {
    return next;
  }
  return current;
}

function delegateToolCallAgentRun(toolCall: domain.ToolCall): AgentRun | undefined {
  const structured = objectRecord(toolCall.result?.structured);
  const result = objectRecord(structured?.result);
  if (!result) return undefined;
  const id = stringValue(result.id);
  const sessionId = stringValue(result.sessionId);
  if (!id && !sessionId) return undefined;
  return {
    id: id || sessionId || toolCall.id,
    parentSessionId: stringValue(result.parentSessionId),
    sessionId,
    mode: (stringValue(result.mode) || "assistant") as AgentRun["mode"],
    status: stringValue(result.status) || toolCall.status,
    prompt: stringValue(result.prompt),
    result: stringValue(result.result),
    error: stringValue(result.error),
    metadata: objectStringRecord(result.metadata),
    timeCreated: stringValue(result.timeCreated) || toolCall.timeCreated,
    timeUpdated: stringValue(result.timeUpdated) || toolCall.timeUpdated,
    timeCompleted: stringValue(result.timeCompleted),
  };
}

function delegateToolCallSessionId(toolCall: domain.ToolCall) {
  return delegateToolCallAgentRun(toolCall)?.sessionId || "";
}

function subagentStatusLabel(status: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "completed":
    case "success":
      return "已完成";
    case "failed":
      return "失败";
    case "cancelled":
      return "已取消";
    case "pending_approval":
      return "等待批准";
    default:
      return status || "未知";
  }
}

function subagentStatusClass(status: string) {
  if (status === "completed" || status === "success") {
    return "text-emerald-600 dark:text-emerald-400";
  }
  if (status === "failed" || status === "cancelled") {
    return "text-destructive";
  }
  return "text-muted-foreground";
}

function agentModeDisplayName(mode: string) {
  switch (mode) {
    case "planner":
      return "规划";
    case "assistant":
      return "助手";
	case "scheduler_worker":
      return "计划任务";
    default:
      return mode || "助手";
  }
}

function getToolResultText(toolCall: domain.ToolCall) {
 const result = toolCall.result ?? {};
 const content = result.content;
 if (typeof content === "string" && content.trim()) return content;
 if (toolCall.resultSummary?.trim()) return toolCall.resultSummary;
 const error = result.error;
 if (typeof error === "string" && error.trim()) return error;
 return "";
}

function getRetainedOutputRefs(toolCall: domain.ToolCall) {
 const refs = toolCall.result?.retainedOutputRefs;
 if (!Array.isArray(refs)) return [];
 return refs.filter((ref): ref is string => typeof ref === "string" && Boolean(ref.trim()));
}

function failedToolRunDetail(toolCall: domain.ToolCall, resultText: string) {
 if (toolCall.name === "format_code") return "format code";
 return commandFromToolResultText(resultText);
}

function commandFromToolResultText(text: string) {
 const commandLine = text
 .split("\n")
 .find((line) => line.trimStart().startsWith("Command:"));
 return commandLine?.replace(/^\s*Command:\s*/, "").trim() ?? "";
}

function patchSummary(args: Record<string, unknown>) {
 const patch = args.patchText;
 if (typeof patch !== "string") return "";
 const fileCount = new Set(
 patch
 .split("\n")
 .filter(
 (line) =>
 line.startsWith("*** Add File: ") ||
 line.startsWith("*** Update File: ") ||
 line.startsWith("*** Delete File: "),
 )
 .map((line) =>
 line
 .replace(/^\*\*\* Add File: /, "")
 .replace(/^\*\*\* Update File: /, "")
 .replace(/^\*\*\* Delete File: /, "")
 .trim(),
 )
 .filter((path) => path),
 ).size;
 if (fileCount <= 0) return "patch";
 return `${fileCount} 个文件`;
}

function stringArg(args: Record<string, unknown>, key: string) {
 const value = args[key];
 return typeof value === "string" ? value : "";
}

function scalarArg(args: Record<string, unknown>, key: string) {
 const value = args[key];
 if (
 typeof value === "string" ||
 typeof value === "number" ||
 typeof value === "boolean"
 ) {
 return String(value);
 }
 return "";
}

function stringValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

function parseTime(value?: string) {
  if (!value) return new Date();
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? new Date() : date;
}

function objectRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function arrayRecords(value: unknown): Record<string, unknown>[] {
 if (!Array.isArray(value)) return [];
 return value.flatMap((item) => {
 const record = objectRecord(item);
 return record ? [record] : [];
 });
}

function objectStringRecord(value: unknown): Record<string, string> | undefined {
  const record = objectRecord(value);
  if (!record) return undefined;
  return Object.fromEntries(
    Object.entries(record).filter(
      (entry): entry is [string, string] => typeof entry[1] === "string",
    ),
  );
}

function truncateInline(value: string, max = 120) {
  const compact = value.trim().replace(/\s+/g, " ");
  if (compact.length <= max) return compact;
  return `${compact.slice(0, Math.max(0, max - 1)).trimEnd()}…`;
}

function visibleToolArgs(args: Record<string, unknown>, skippedKeys: string[]) {
 const skipped = new Set(skippedKeys);
 return Object.entries(args)
 .filter(([key]) => !skipped.has(key))
 .flatMap(([key, value]) => {
 if (
 typeof value === "string" ||
 typeof value === "number" ||
 typeof value === "boolean"
 ) {
 return [`${key}=${String(value)}`];
 }
 return [];
 });
}

function toolCallArgumentLabels(args: Record<string, unknown>) {
 const skipped = new Set([
 "description",
 "query",
 "url",
 "path",
 "filePath",
 "pattern",
 "name",
 ]);
 return Object.entries(args)
 .filter(([key]) => !skipped.has(key))
 .flatMap(([key, value]) => {
 if (
 typeof value === "string" ||
 typeof value === "number" ||
 typeof value === "boolean"
 ) {
 return [`${key}=${String(value)}`];
 }
 return [];
 })
 .slice(0, 3);
}

function joinCommandParts(parts: string[]) {
 return parts.filter(Boolean).join(" ");
}

function StoppedResponse({ stoppedSeconds }: { stoppedSeconds: number }) {
 return (
 <Marker
 className="animate-in fade-in slide-in-from-bottom-2 text-sm  duration-300"
 role="status"
 variant="separator"
 >
 <MarkerContent>
 你在 {formatThinkingTime(stoppedSeconds)} 后停止了
 </MarkerContent>
 </Marker>
 );
}

function ThinkingResponse({
 actionHeading,
 showSkeleton,
}: {
 actionHeading?: string;
 showSkeleton: boolean;
}) {
 return (
 <div className="animate-in flex min-w-0 max-w-full flex-col items-stretch gap-3 fade-in slide-in-from-bottom-2 duration-300">
 <ThinkingStatus actionHeading={actionHeading} />
 {showSkeleton ? (
 <div className="flex min-w-0 max-w-full flex-col gap-3.5">
 <div className="h-4 w-full max-w-[540px] rounded-full bg-muted" />
 <div className="h-4 w-full max-w-[680px] rounded-full bg-muted" />
 <div className="h-4 w-full max-w-[460px] rounded-full bg-muted" />
 </div>
 ) : null}
 </div>
 );
}

function AssistantStatus({
 actionHeading,
 completed,
 responseSeconds,
}: {
 actionHeading?: string;
 completed: boolean;
 responseSeconds: number;
}) {
 if (!completed) {
 return <ThinkingStatus actionHeading={actionHeading} />;
 }

 return (
 <Marker
 className="animate-in fade-in slide-in-from-bottom-2 text-sm  duration-300"
 role="status"
 variant="separator"
 >
 <MarkerContent>
 已完成 {formatThinkingTime(responseSeconds)}
 </MarkerContent>
 </Marker>
 );
}

function ThinkingStatus({ actionHeading }: { actionHeading?: string }) {
 return (
 <div
 className="animate-in flex min-w-0 flex-col gap-1.5 fade-in slide-in-from-bottom-2 text-sm duration-300"
 role="status"
 >
 <ShimmerText text="正在思考" />
 {actionHeading ? (
 <div className="min-w-0 truncate text-muted-foreground">
 {actionHeading}
 </div>
 ) : null}
 </div>
 );
}

function ShimmerText({ text }: { text: string }) {
 return (
 <span
 aria-label={text}
 className="aivo-text-shimmer font-semibold text-muted-foreground"
 >
 <span data-slot="text-shimmer-char">
 <span aria-hidden="true" data-slot="text-shimmer-char-base">
 {text}
 </span>
 <span aria-hidden="true" data-slot="text-shimmer-char-shimmer">
 {text}
 </span>
 </span>
 </span>
 );
}

function AssistantResponse({
 actions,
 completedAt,
 responseText,
 turn,
}: {
 actions: ConversationTimelineActions;
 completedAt: Date | null;
 responseText: string;
 turn: ConversationTurn;
}) {
 return (
 <div className="group/assistant-response relative">
 <Markdown content={responseText} isFinished={Boolean(completedAt)} />
 <div className="relative h-6">
 <div
 className="pointer-events-none absolute left-0 top-0 z-10 flex items-center gap-2 opacity-0 transition-opacity duration-200 ease-out group-hover/assistant-response:pointer-events-auto group-hover/assistant-response:opacity-100 group-focus-within/assistant-response:pointer-events-auto group-focus-within/assistant-response:opacity-100"
 >
 <CopyTextButton ariaLabel="复制回复" text={responseText} />
 <Button aria-label="赞" size="icon-sm" type="button" variant="ghost">
 <ThumbsUp />
 </Button>
 <Button aria-label="踩" size="icon-sm" type="button" variant="ghost">
 <ThumbsDown />
 </Button>
 {turn.turnId && completedAt && actions.onRetryTurn ? (
 <Button
 aria-label="重试"
 onClick={() => actions.onRetryTurn?.(turn)}
 size="icon-sm"
 title="重试"
 type="button"
 variant="ghost"
 >
 <RotateCcw />
 </Button>
 ) : null}
 {turn.assistantEventId && actions.onDeleteAssistantMessage ? (
 <Button
 aria-label="删除回复"
 onClick={() => actions.onDeleteAssistantMessage?.(turn)}
 size="icon-sm"
 title="删除回复"
 type="button"
 variant="ghost"
 >
 <Trash2 />
 </Button>
 ) : null}
 <Button aria-label="展开" size="icon-sm" type="button" variant="ghost">
 <Expand />
 </Button>
 {completedAt && (
 <span className="text-sm ">
 {formatCompletionTime(completedAt)}
 </span>
 )}
 </div>
 </div>
 </div>
 );
}

function CopyTextButton({
 ariaLabel,
 text,
}: {
 ariaLabel: string;
 text: string;
}) {
 const [copied, setCopied] = useState(false);

 useEffect(() => {
 if (!copied) return;
 const timeoutId = window.setTimeout(() => setCopied(false), 1400);
 return () => window.clearTimeout(timeoutId);
 }, [copied]);

 async function copyText() {
 try {
 await navigator.clipboard.writeText(text);
 setCopied(true);
 } catch {
 setCopied(false);
 }
 }

 return (
 <Button
 aria-label={copied ? "已复制" : ariaLabel}
 onClick={copyText}
 size="icon-sm"
 title={copied ? "已复制" : ariaLabel}
 type="button"
 variant="ghost"
 >
 {copied ? <Check /> : <Copy />}
 </Button>
 );
}

function isExpandableUserMessage(text: string) {
 return text.length > 320 || text.split(/\r?\n/).length > 6;
}

function formatCompletionTime(date: Date) {
 return `${date.getHours()}:${date.getMinutes().toString().padStart(2, "0")}`;
}

function formatThinkingTime(totalSeconds: number) {
 if (totalSeconds < 60) return `${totalSeconds}s`;

 const minutes = Math.floor(totalSeconds / 60);
 const seconds = totalSeconds % 60;

 return seconds === 0 ? `${minutes}m` : `${minutes}m${seconds}s`;
}
