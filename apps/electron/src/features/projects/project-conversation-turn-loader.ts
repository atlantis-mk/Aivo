import { useCallback } from "react";
import type { Dispatch, SetStateAction } from "react";

import {
  getTurnElapsedSeconds,
  type ConversationAssistantTextPart,
  type ConversationTurn,
  type ConversationUserAttachment,
  type ConversationUserPaste,
} from "@/features/projects/conversation-timeline-model";
import {
  applyPendingTurnMetadata,
  hasRunningTurn,
  mergePreservedTurnAttachments,
  mergeTurnPauseMetadata,
  toolCallsForTurn,
  turnsFromEvents,
} from "@/features/projects/project-conversation-events";
import { parseTime } from "@/features/projects/project-time-model";
import { codexToolCallFromItem } from "@/features/projects/project-codex-tool-calls";
import { runtimeMetricsFromEventPayload } from "@/features/projects/project-session-runtime-stats";
import {
  listSessionEvents,
  listSessionToolCalls,
  listSessionTurns,
} from "@/services/aivo";
import { listCodexThreadTurns } from "@/services/codex-thread-service";
import { hasCodexDesktopBridge } from "@/lib/app-config";
import type { domain } from "@/types/codex-domain";

export type LoadConversationTurnsOptions = {
  pendingTurnId?: string;
  pendingPrompt?: string;
  pendingAttachments?: ConversationUserAttachment[];
  pendingStartedAt?: number;
  fallbackAssistantEvent?: domain.SessionEvent;
  snapToBottomAfterLoad?: boolean;
};

export function useProjectConversationTurnLoader({
  activeSessionIdRef,
  prepareConversationReveal,
  setConversationRunning,
  setTurns,
  turns,
}: {
  activeSessionIdRef: { current: string };
  prepareConversationReveal: (turnCount: number) => void;
  setConversationRunning: (sessionId: string, running: boolean) => void;
  setTurns: Dispatch<SetStateAction<ConversationTurn[]>>;
  turns: ConversationTurn[];
}) {
  return useCallback(
    async function loadConversationTurns(
      sessionId: string,
      options: LoadConversationTurnsOptions = {},
    ) {
      if (hasCodexDesktopBridge()) {
        const nextTurns = await Promise.all(
          (await listCodexThreadTurns(sessionId)).map(async (turn) => {
            const conversationTurn = codexTurnToConversationTurn(
              turn,
              sessionId,
            );
            return {
              ...conversationTurn,
              sessionId,
              attachments: await hydrateConversationAttachments(
                conversationTurn.attachments,
              ),
            };
          }),
        );
        const hydratedTurns = mergeTurnPauseMetadata(
          applyPendingTurnMetadata(nextTurns, options),
          turns,
        );
        setConversationRunning(
          sessionId,
          hydratedTurns.some((turn) => !turn.responseCompletedAt && !turn.stopped),
        );
        if (activeSessionIdRef.current !== sessionId) return;
        if (options.snapToBottomAfterLoad) {
          prepareConversationReveal(hydratedTurns.length);
        }
        setTurns(hydratedTurns);
        return;
      }
      const [events, toolCalls, runtimeTurns] = await Promise.all([
        listSessionEvents(sessionId, false, 100),
        listSessionToolCalls(sessionId).catch(() => [] as domain.ToolCall[]),
        listSessionTurns(sessionId, 100).catch(() => [] as domain.Turn[]),
      ]);
      let nextTurns = turnsFromEvents(
        events ?? [],
        toolCalls ?? [],
        runtimeTurns ?? [],
      ).map((turn) => ({ ...turn, sessionId }));
      if (
        nextTurns.length === 0 &&
        options.fallbackAssistantEvent &&
        options.pendingPrompt
      ) {
        const submittedAt = new Date(options.pendingStartedAt ?? Date.now());
        const completedAt = parseTime(
          options.fallbackAssistantEvent.timeCreated,
        );
        nextTurns = [
          {
            id: options.pendingTurnId ?? options.fallbackAssistantEvent.id,
            activityVisible: (toolCalls ?? []).some(
              (toolCall) =>
                toolCall.turnId === options.fallbackAssistantEvent?.turnId,
            ),
            assistantPreambles: [],
            prompt: options.pendingPrompt,
            attachments:
              options.pendingAttachments ??
              mergePreservedTurnAttachments(options.pendingTurnId, turns),
            preToolText: "",
            responseCompletedAt: completedAt,
            responseText: options.fallbackAssistantEvent.content ?? "",
            responseVisible: true,
            runtimeMetrics: runtimeMetricsFromEventPayload(
              options.fallbackAssistantEvent.payload,
            ),
            startedAt: submittedAt.getTime(),
            stopped: false,
            submittedAt,
            thinkingSeconds: getTurnElapsedSeconds({
              startedAt: options.pendingStartedAt ?? submittedAt.getTime(),
            }),
            toolCalls: toolCallsForTurn(
              toolCalls ?? [],
              options.fallbackAssistantEvent.turnId,
            ),
            sessionId,
            turnId: options.fallbackAssistantEvent.turnId,
            assistantEventId: options.fallbackAssistantEvent.id,
          },
        ];
      }
      const hydratedTurns = mergeTurnPauseMetadata(
        applyPendingTurnMetadata(nextTurns, options),
        turns,
      );
      setConversationRunning(sessionId, hasRunningTurn(hydratedTurns));
      if (activeSessionIdRef.current !== sessionId) {
        return;
      }
      if (options.snapToBottomAfterLoad) {
        prepareConversationReveal(hydratedTurns.length);
      }
      setTurns(hydratedTurns);
    },
    [
      activeSessionIdRef,
      prepareConversationReveal,
      setConversationRunning,
      setTurns,
      turns,
    ],
  );
}

function codexTurnToConversationTurn(
  turn: CodexThreadTurn,
  threadId: string,
): ConversationTurn {
  const startedAt = turn.startedAt ? Date.parse(turn.startedAt) : Date.now();
  const completedAt = turn.completedAt ? new Date(turn.completedAt) : null;
  const userMessage = userMessageContentFromItems(
    turn.items
      .filter(isCodexItemType("userMessage"))
      .map((item) => item.content),
    turn.id,
  );
  const attachments = turn.items
    .filter(isCodexItemType("userMessage"))
    .flatMap((item) => imageAttachmentsFromUserMessage(item.content, turn.id));
  const assistantText = splitCodexAssistantText(turn.items);
  const toolCalls = turn.items.flatMap((item) => {
    const toolCall = codexToolCallFromItem({
      item,
      threadId,
      timeCreated: turn.startedAt ?? undefined,
      timeUpdated: turn.completedAt ?? turn.startedAt ?? undefined,
      turnId: turn.id,
    });
    return toolCall ? [toolCall] : [];
  });

  return {
    activityVisible: turn.status === "inProgress" || toolCalls.length > 0,
    assistantPreambles: assistantText.preambles,
    attachments,
    id: turn.id,
    model: turn.model ?? undefined,
    modelProvider: turn.modelProvider ?? undefined,
    preToolText: assistantText.preambles.map((part) => part.text).join("\n"),
    pastes: userMessage.pastes,
    prompt: userMessage.prompt,
    responseCompletedAt:
      completedAt ?? (turn.status === "inProgress" ? null : new Date(startedAt)),
    responseText: assistantText.responseText || turn.error || "",
    responseVisible: Boolean(assistantText.responseText || turn.error),
    startedAt,
    stopped: turn.status === "interrupted",
    submittedAt: new Date(startedAt),
    thinkingSeconds:
      typeof turn.durationMs === "number" && Number.isFinite(turn.durationMs)
        ? Math.max(0, Math.floor(turn.durationMs / 1000))
        : getTurnElapsedSeconds({ startedAt }, completedAt?.getTime()),
    toolCalls,
    turnId: turn.id,
  };
}

function splitCodexAssistantText(items: unknown[]) {
  const lastToolIndex = items.findLastIndex(isCodexToolItem);
  const agentMessages = items
    .map((item, index) => ({
      index,
      item,
      text: textFromAgentMessage(item),
    }))
    .filter((entry): entry is { index: number; item: unknown; text: string } =>
      Boolean(entry.text),
    );

  if (lastToolIndex < 0) {
    return {
      preambles: [],
      responseText: agentMessages.map((entry) => entry.text).join("\n"),
    };
  }

  const preambles: ConversationAssistantTextPart[] = [];
  let segmentTexts: string[] = [];
  for (let index = 0; index <= lastToolIndex; index += 1) {
    const item = items[index];
    const agentMessageText = textFromAgentMessage(item);
    if (agentMessageText) {
      segmentTexts.push(agentMessageText);
      continue;
    }
    if (!isCodexToolItem(item) || segmentTexts.length === 0) continue;

    const toolItemId = codexItemId(item);
    preambles.push({
      id: `codex-preamble:${toolItemId ?? index}`,
      text: segmentTexts.join("\n\n"),
      timeCreated: undefined,
    });
    segmentTexts = [];
  }

  return {
    preambles,
    responseText: agentMessages
      .filter((entry) => entry.index > lastToolIndex)
      .map((entry) => entry.text)
      .join("\n"),
  };
}

function codexItemId(item: unknown) {
  const value = (item as Record<string, unknown> | null)?.id;
  return typeof value === "string" && value ? value : null;
}

function isCodexToolItem(item: unknown) {
  const type = codexItemType(item);
  return (
    type === "commandExecution" ||
    type === "mcpToolCall" ||
    type === "webSearch"
  );
}

function textFromAgentMessage(item: unknown) {
  if (codexItemType(item) !== "agentMessage") return null;
  const text = (item as Record<string, unknown>).text;
  return typeof text === "string" && text.trim() ? text : null;
}

function codexItemType(item: unknown) {
  return typeof item === "object" && item !== null
    ? (item as Record<string, unknown>).type
    : undefined;
}

function isCodexItemType(type: string) {
  return (item: unknown): item is Record<string, unknown> =>
    typeof item === "object" &&
    item !== null &&
    (item as Record<string, unknown>).type === type;
}

function userMessageContentFromItems(contents: unknown[], turnId: string) {
  const prompts: string[] = [];
  const pastes: ConversationUserPaste[] = [];

  for (const content of contents) {
    if (!Array.isArray(content)) continue;
    for (const item of content) {
      if (typeof item !== "object" || item === null) continue;
      const record = item as Record<string, unknown>;
      if (record.type !== "text" || typeof record.text !== "string") continue;
      const parsed = textAndPastesFromUserMessage(
        record.text,
        record.textElements ?? record.text_elements,
        turnId,
        pastes.length,
      );
      if (parsed.prompt) prompts.push(parsed.prompt);
      pastes.push(...parsed.pastes);
    }
  }

  return { pastes, prompt: prompts.join("\n") };
}

function textAndPastesFromUserMessage(
  text: string,
  rawElements: unknown,
  turnId: string,
  pasteOffset: number,
) {
  const elements = textElementsFromUnknown(rawElements, text);
  if (elements.length === 0) return { pastes: [], prompt: text };

  let cursor = 0;
  const promptParts: string[] = [];
  const pastes: ConversationUserPaste[] = [];
  for (const element of elements) {
    const start = codeUnitOffsetForUtf8Byte(text, element.start);
    const end = codeUnitOffsetForUtf8Byte(text, element.end);
    if (start === null || end === null || start < cursor || start >= end) continue;
    promptParts.push(text.slice(cursor, start));
    const pastedText = text.slice(start, end);
    const name = element.placeholder || "粘贴内容";
    pastes.push({
      id: `codex-paste:${turnId}:${pasteOffset + pastes.length + 1}`,
      name,
      preview: pastedText.trim().split("\n").find(Boolean)?.trim() || name,
      text: pastedText,
    });
    cursor = end;
  }
  if (pastes.length === 0) return { pastes: [], prompt: text };
  promptParts.push(text.slice(cursor));
  return { pastes, prompt: promptParts.join("").trim() };
}

function textElementsFromUnknown(rawElements: unknown, text: string) {
  if (!Array.isArray(rawElements)) return [];
  return rawElements.flatMap((rawElement) => {
    if (typeof rawElement !== "object" || rawElement === null) return [];
    const element = rawElement as Record<string, unknown>;
    const range = element.byteRange ?? element.byte_range;
    if (typeof range !== "object" || range === null) return [];
    const { start, end } = range as Record<string, unknown>;
    if (
      typeof start !== "number" ||
      typeof end !== "number" ||
      !Number.isInteger(start) ||
      !Number.isInteger(end) ||
      start < 0 ||
      end > utf8ByteLength(text) ||
      start >= end
    ) {
      return [];
    }
    return [{
      end,
      placeholder: typeof element.placeholder === "string" ? element.placeholder : "",
      start,
    }];
  }).sort((a, b) => a.start - b.start);
}

function codeUnitOffsetForUtf8Byte(text: string, byteOffset: number) {
  if (byteOffset === 0) return 0;
  let bytes = 0;
  for (let index = 0; index < text.length;) {
    const codePoint = text.codePointAt(index)!;
    const nextIndex = index + (codePoint > 0xffff ? 2 : 1);
    bytes += codePoint <= 0x7f ? 1 : codePoint <= 0x7ff ? 2 : codePoint <= 0xffff ? 3 : 4;
    if (bytes === byteOffset) return nextIndex;
    if (bytes > byteOffset) return null;
    index = nextIndex;
  }
  return bytes === byteOffset ? text.length : null;
}

function utf8ByteLength(value: string) {
  return new TextEncoder().encode(value).length;
}

function imageAttachmentsFromUserMessage(
  content: unknown,
  turnId: string,
): ConversationUserAttachment[] {
  if (!Array.isArray(content)) return [];
  let imageIndex = 0;
  return content.flatMap((item): ConversationUserAttachment[] => {
    if (typeof item !== "object" || item === null) return [];
    const record = item as Record<string, unknown>;
    if (record.type === "image") {
      if (typeof record.url !== "string") return [];
      const mimeType = mimeTypeFromDataUrl(record.url);
      if (!mimeType) return [];
      imageIndex += 1;
      return [{
        id: `codex-image:${turnId}:${imageIndex}`,
        kind: "image" as const,
        mimeType,
        name: `图片 ${imageIndex}`,
        previewUrl: record.url,
      }];
    }

    if (record.type !== "localImage" || typeof record.path !== "string") return [];
    const mimeType = mimeTypeFromImagePath(record.path);
    if (!mimeType) return [];
    imageIndex += 1;
    return [{
      id: `codex-image:${turnId}:${imageIndex}`,
      kind: "image" as const,
      mimeType,
      name: `图片 ${imageIndex}`,
      path: record.path,
    }];
  });
}

function mimeTypeFromDataUrl(url: string) {
  return /^data:([^;,]+);base64,/i.exec(url)?.[1] ?? null;
}

function mimeTypeFromImagePath(path: string) {
  const extension = path.split(".").at(-1)?.toLowerCase();
  if (extension === "png") return "image/png";
  if (extension === "jpg" || extension === "jpeg") return "image/jpeg";
  if (extension === "webp") return "image/webp";
  if (extension === "gif") return "image/gif";
  return null;
}

async function hydrateConversationAttachments(
  attachments: ConversationUserAttachment[] | undefined,
) {
  if (!attachments?.length) return attachments;
  return Promise.all(
    attachments.map(async (attachment) => {
      if (
        attachment.kind !== "image" ||
        attachment.previewUrl ||
        !attachment.path ||
        !hasCodexDesktopBridge()
      ) {
        return attachment;
      }
      const previewUrl =
        await window.aivoDesktop.file.readDataUrl(attachment.path);
      return previewUrl ? { ...attachment, previewUrl } : attachment;
    }),
  );
}
