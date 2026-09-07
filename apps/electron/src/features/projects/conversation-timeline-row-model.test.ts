import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { constructConversationTimelineRows } from "./conversation-timeline-row-model.ts";
import type { ConversationTurn } from "./conversation-timeline-model.ts";

function createTurn(
  overrides: Partial<ConversationTurn> = {},
): ConversationTurn {
  return {
    activityVisible: false,
    assistantPreambles: [],
    attachments: [],
    id: "turn-1",
    prompt: "hello",
    preToolText: "",
    responseCompletedAt: null,
    responseText: "",
    responseVisible: false,
    startedAt: 0,
    stopped: false,
    submittedAt: new Date(0),
    thinkingSeconds: 0,
    toolCalls: [],
    ...overrides,
  };
}

describe("conversation timeline rows", () => {
  it("shows assistant content as soon as a normal turn starts responding", () => {
    const rows = constructConversationTimelineRows([
      createTurn({
        responseText: "partial reply",
        responseVisible: true,
      }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      ["user-message", "assistant-status", "assistant-response"],
    );
  });

  it("hides the original skeleton until a steered turn has assistant content", () => {
    const rows = constructConversationTimelineRows([
      createTurn({ hasSteeredInput: true }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      ["user-message"],
    );
  });

  it("shows steered assistant content with elapsed time and without another skeleton", () => {
    const rows = constructConversationTimelineRows([
      createTurn({
        responseText: "steered reply",
        responseVisible: true,
        steerTargetTurnId: "turn-1",
        steered: true,
      }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      ["user-message", "assistant-status", "assistant-response"],
    );
  });
});
