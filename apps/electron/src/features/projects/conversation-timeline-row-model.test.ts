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

  it("keeps live activity in status, tool, then answer order", () => {
    const rows = constructConversationTimelineRows([
      createTurn({
        activityVisible: true,
        assistantPreambles: [
          {
            id: "preamble-1",
            text: "我先检查项目结构。",
            timeCreated: "2026-09-07T00:00:00.000Z",
          },
        ],
        toolCalls: [
          {
            id: "tool-1",
            name: "exec_command",
            status: "running",
            timeCreated: "2026-09-07T00:00:01.000Z",
          } as ConversationTurn["toolCalls"][number]],
      }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      ["user-message", "thinking", "assistant-preamble", "tool-group"],
    );
    const thinking = rows.find((row) => row.type === "thinking");
    assert.equal(thinking?.actionHeading, "正在运行 1 条命令");
    const preamble = rows.find((row) => row.type === "assistant-preamble");
    assert.equal(preamble?.isCompleted, false);
  });

  it("puts the completed summary before collapsed tool activity and the answer", () => {
    const rows = constructConversationTimelineRows([
      createTurn({
        responseCompletedAt: new Date("2026-09-07T00:00:02.000Z"),
        responseText: "完成了。",
        responseVisible: true,
        toolCalls: [
          {
            id: "tool-1",
            name: "read_file",
            status: "completed",
            timeCreated: "2026-09-07T00:00:01.000Z",
          } as ConversationTurn["toolCalls"][number]],
      }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      ["user-message", "assistant-status", "tool-group", "assistant-response"],
    );
    const toolGroup = rows.find((row) => row.type === "tool-group");
    assert.equal(toolGroup?.isCompleted, true);
  });

  it("puts a stopped status before a collapsed tool summary", () => {
    const rows = constructConversationTimelineRows([
      createTurn({
        assistantPreambles: [
          {
            id: "preamble-1",
            text: "我先检查项目结构。",
            timeCreated: "2026-09-07T00:00:00.000Z",
          },
        ],
        responseCompletedAt: new Date("2026-09-07T00:00:02.000Z"),
        responseText: "已经完成的部分回复。",
        responseVisible: true,
        stopped: true,
        toolCalls: [
          {
            id: "tool-1",
            name: "read_file",
            status: "completed",
            timeCreated: "2026-09-07T00:00:01.000Z",
          } as ConversationTurn["toolCalls"][number]],
      }),
    ]);

    assert.deepEqual(
      rows.map((row) => row.type),
      [
        "user-message",
        "stopped",
        "assistant-preamble",
        "tool-group",
        "assistant-response",
      ],
    );
    const toolGroup = rows.find((row) => row.type === "tool-group");
    assert.equal(toolGroup?.defaultCollapsed, true);
    assert.equal(toolGroup?.isCompleted, false);
  });
});
