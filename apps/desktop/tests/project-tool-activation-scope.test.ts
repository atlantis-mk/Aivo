import assert from "node:assert/strict";
import test from "node:test";

import {
  consumePendingToolActivation,
  discardLegacyDefaultActiveToolNames,
  LEGACY_DEFAULT_ACTIVE_TOOL_NAMES_STORAGE_KEY,
  scopeToolActivationSave,
} from "../src/features/projects/project-tool-activation-scope.ts";

test("existing conversation activation remains session-scoped", () => {
  assert.deepEqual(
    scopeToolActivationSave("session-1", [
      " mcp_chrome_list_tabs ",
      "mcp_chrome_list_tabs",
    ]),
    {
      kind: "session",
      sessionId: "session-1",
      toolNames: ["mcp_chrome_list_tabs"],
    },
  );
});

test("activation without a conversation becomes a one-shot draft", () => {
  assert.deepEqual(scopeToolActivationSave("", ["extension_notes_write"]), {
    kind: "pending",
    toolNames: ["extension_notes_write"],
  });
});

test("new conversation consumption clears the draft", () => {
  assert.deepEqual(
    consumePendingToolActivation([
      "extension_notes_write",
      " mcp_chrome_list_tabs ",
      "extension_notes_write",
    ]),
    {
      appliedToolNames: ["extension_notes_write", "mcp_chrome_list_tabs"],
      remainingToolNames: [],
    },
  );
});

test("legacy global default is discarded instead of restored", () => {
  const removed: string[] = [];
  assert.deepEqual(
    discardLegacyDefaultActiveToolNames({
      removeItem: (key) => removed.push(key),
    }),
    [],
  );
  assert.deepEqual(removed, [LEGACY_DEFAULT_ACTIVE_TOOL_NAMES_STORAGE_KEY]);
});
