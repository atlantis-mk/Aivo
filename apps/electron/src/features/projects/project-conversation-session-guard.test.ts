import assert from "node:assert/strict";
import { describe, it } from "node:test";

import type { ConversationTurn } from "./conversation-timeline-model.ts";
import { mergeSingleToolCall } from "./project-conversation-live-turns.ts";
import { mergeTurnPauseMetadata } from "./project-conversation-live-metadata.ts";

function createTurn(
  overrides: Partial<ConversationTurn> = {},
): ConversationTurn {
  return {
    activityVisible: false,
    assistantPreambles: [],
    attachments: [],
    id: "turn-1",
    prompt: "",
    preToolText: "",
    responseCompletedAt: null,
    responseText: "",
    responseVisible: false,
    sessionId: "session-a",
    startedAt: 0,
    stopped: false,
    submittedAt: new Date(0),
    thinkingSeconds: 0,
    toolCalls: [],
    ...overrides,
  };
}

describe("conversation session guards", () => {
  it("does not merge tool activity across sessions", () => {
    const turns = mergeSingleToolCall(
      [createTurn({ sessionId: "session-b" })],
      {
        id: "codex:a:item-1",
        name: "exec_command",
        sessionId: "session-a",
        status: "running",
        timeCreated: new Date().toISOString(),
        timeUpdated: new Date().toISOString(),
        turnId: "turn-a",
      } as never,
    );

    assert.equal(turns[0].toolCalls.length, 0);
  });

  it("does not preserve image attachments across sessions", () => {
    const attachment = {
      id: "image-1",
      kind: "image" as const,
      mimeType: "image/png",
      name: "a.png",
      previewUrl: "data:image/png;base64,a",
    };
    const turns = mergeTurnPauseMetadata(
      [createTurn({ sessionId: "session-b", turnId: "turn-b" })],
      [
        createTurn({
          attachments: [attachment],
          id: "pending-a",
          sessionId: "session-a",
          turnId: "turn-a",
        }),
      ],
    );

    assert.equal(turns[0].attachments?.length, 0);
  });
});
