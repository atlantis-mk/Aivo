export type ExtensionToolViewRef = {
  extensionId: string;
  viewId: string;
  surface: "tool-detail" | "page";
  title?: string;
};

export type ExtensionToolViewContext = {
  operationId: string;
  sessionId: string;
  turnId: string;
  toolName: string;
};

type ToolCallContextSource = {
  id: string;
  sessionId?: string;
  turnId?: string;
  name?: string;
};

type ToolCallWithResult = {
  id: string;
  result?: Record<string, unknown>;
};

export function extensionToolViewRef(
  toolCall: ToolCallWithResult | null | undefined,
): ExtensionToolViewRef | null {
  const details = recordValue(toolCall?.result?.details);
  const view = recordValue(details?.view);
  const extensionId = stringValue(view?.extensionId);
  const viewId = stringValue(view?.viewId);
  const surface = stringValue(view?.surface);
  if (
    !/^[a-z0-9][a-z0-9._-]*$/.test(extensionId) ||
    !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(viewId) ||
    (surface !== "tool-detail" && surface !== "page")
  ) {
    return null;
  }
  const title = stringValue(view?.title).slice(0, 200);
  return { extensionId, viewId, surface, ...(title ? { title } : {}) };
}

export function latestExtensionViewToolCallId(
  entries: ReadonlyArray<{ toolCall: ToolCallWithResult }>,
) {
  return (
    entries.findLast(({ toolCall }) => Boolean(extensionToolViewRef(toolCall)))
      ?.toolCall.id ?? ""
  );
}

export function selectedExtensionViewToolCallId({
  activityId,
  latestViewToolCallId,
  selectedToolCallId,
  trackedActivityId,
}: {
  activityId: string;
  latestViewToolCallId: string;
  selectedToolCallId: string;
  trackedActivityId: string;
}) {
  if (!activityId) return "";
  return trackedActivityId === activityId
    ? selectedToolCallId
    : latestViewToolCallId;
}

export function extensionToolViewContext(
  toolCall: ToolCallContextSource,
): ExtensionToolViewContext {
  return {
    operationId: toolCall.id,
    sessionId: toolCall.sessionId ?? "",
    turnId: toolCall.turnId ?? "",
    toolName: toolCall.name ?? "",
  };
}

export function extensionToolViewIdentity(view: ExtensionToolViewRef) {
  return `${view.extensionId}\u0000${view.viewId}\u0000${view.surface}`;
}

function recordValue(value: unknown) {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}

function stringValue(value: unknown) {
  return typeof value === "string" ? value.trim() : "";
}
