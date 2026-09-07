export type ToolActivationSaveScope =
  | {
      kind: "pending";
      toolNames: string[];
    }
  | {
      kind: "session";
      sessionId: string;
      toolNames: string[];
    };

export const LEGACY_DEFAULT_ACTIVE_TOOL_NAMES_STORAGE_KEY =
  "aivo:default-active-tool-names:v1";

export function scopeToolActivationSave(
  sessionId: string,
  toolNames: string[],
): ToolActivationSaveScope {
  const normalized = normalizeScopedToolNames(toolNames);
  const normalizedSessionId = sessionId.trim();
  if (!normalizedSessionId) {
    return { kind: "pending", toolNames: normalized };
  }
  return {
    kind: "session",
    sessionId: normalizedSessionId,
    toolNames: normalized,
  };
}

export function consumePendingToolActivation(toolNames: string[]) {
  return {
    appliedToolNames: normalizeScopedToolNames(toolNames),
    remainingToolNames: [] as string[],
  };
}

export function discardLegacyDefaultActiveToolNames(storage?: {
  removeItem: (key: string) => void;
}) {
  try {
    storage?.removeItem(LEGACY_DEFAULT_ACTIVE_TOOL_NAMES_STORAGE_KEY);
  } catch {
    // Storage may be unavailable in restricted renderer contexts.
  }
  return [] as string[];
}

function normalizeScopedToolNames(toolNames: string[]) {
  return [...new Set(toolNames.map((name) => name.trim()).filter(Boolean))].toSorted();
}
