import assert from "node:assert/strict";
import test from "node:test";

import { hostToolSelectionFromSystemNote } from "../src/features/projects/conversation-system-note-model.ts";
import { mergeSystemNoteEvent } from "../src/features/projects/project-conversation-system-notes.ts";

test("parses source-level initial injection resources and the empty state", () => {
  assert.deepEqual(
    hostToolSelectionFromSystemNote({
      payload: {
        kind: "host_tool_selection",
        lifetime: "conversation",
        resources: [
          {
            kind: "extension",
            id: "github",
            name: "GitHub",
            toolCount: 12,
          },
          { kind: "mcp", id: "docs", name: "Docs", toolCount: 68 },
          { kind: "tool", id: "read", name: "read", toolCount: 1 },
          { kind: "skill", id: "ui-review", name: "UI Review", toolCount: 0 },
        ],
      },
    }),
    {
      lifetime: "conversation",
      resources: [
        {
          kind: "extension",
          id: "github",
          name: "GitHub",
          toolCount: 12,
        },
        { kind: "mcp", id: "docs", name: "Docs", toolCount: 68 },
        { kind: "tool", id: "read", name: "read", toolCount: 1 },
        { kind: "skill", id: "ui-review", name: "UI Review", toolCount: 0 },
      ],
      status: "completed",
    },
  );
  assert.deepEqual(
    hostToolSelectionFromSystemNote({
      payload: {
        kind: "host_tool_selection",
        lifetime: "request",
        resources: [],
      },
    }),
    { lifetime: "request", resources: [], status: "completed" },
  );
});

test("parses live running and failed selection states without exposing a result", () => {
  assert.deepEqual(
    hostToolSelectionFromSystemNote({
      payload: {
        kind: "host_tool_selection",
        status: "running",
        resources: [],
      },
    }),
    { resources: [], status: "running" },
  );
  assert.deepEqual(
    hostToolSelectionFromSystemNote({
      payload: {
        kind: "host_tool_selection",
        status: "failed",
        resources: [],
      },
    }),
    { resources: [], status: "failed" },
  );
});

test("rejects malformed initial tool-selection payloads for safe fallback rendering", () => {
  for (const payload of [
    undefined,
    { kind: "other", resources: [] },
    { kind: "host_tool_selection" },
    { kind: "host_tool_selection", lifetime: "unknown", resources: [] },
    {
      kind: "host_tool_selection",
      status: "running",
      resources: [{ kind: "mcp", id: "docs", name: "Docs", toolCount: 1 }],
    },
    { kind: "host_tool_selection", status: "unknown", resources: [] },
    {
      kind: "host_tool_selection",
      lifetime: "conversation",
      resources: [
        { kind: "tool", id: " spaced ", name: "read", toolCount: 1 },
      ],
    },
    {
      kind: "host_tool_selection",
      lifetime: "conversation",
      resources: [{ kind: "tool", id: "read", name: "read", toolCount: 0 }],
    },
    {
      kind: "host_tool_selection",
      lifetime: "conversation",
      resources: [
        { kind: "agent", id: "code", name: "Code", toolCount: 1 },
      ],
    },
    {
      kind: "host_tool_selection",
      lifetime: "conversation",
      resources: [
        {
          kind: "mcp",
          id: "docs",
          name: "Docs",
          toolCount: 1,
          description: "unexpected",
        },
      ],
    },
    {
      kind: "host_tool_selection",
      lifetime: "conversation",
      resources: [
        { kind: "mcp", id: "docs", name: "Docs", toolCount: 1 },
        { kind: "mcp", id: "docs", name: "Docs", toolCount: 1 },
      ],
    },
  ]) {
    assert.equal(hostToolSelectionFromSystemNote({ payload }), null);
  }
});

test("merges live selection progress and result into one system-note row", () => {
  const turns = [
    {
      activityVisible: false,
      id: "user-1",
      prompt: "use docs",
      preToolText: "",
      responseText: "",
      toolCalls: [],
      turnId: "turn-1",
      startedAt: 1,
      submittedAt: new Date(1),
      thinkingSeconds: 0,
      responseCompletedAt: null,
      responseVisible: false,
      stopped: false,
    },
  ];
  const running = mergeSystemNoteEvent(turns, {
    id: "selection-1",
    sessionId: "session-1",
    turnId: "turn-1",
    type: "system_note",
    role: "system",
    visibility: "normal",
    content: "running",
    payload: {
      kind: "host_tool_selection",
      status: "running",
      resources: [],
    },
    timeCreated: new Date(2).toISOString(),
  });
  assert.equal(running[0].systemNotes?.length, 1);
  assert.equal(running[0].systemNotes?.[0]?.payload?.status, "running");

  const completed = mergeSystemNoteEvent(running, {
    id: "selection-1",
    sessionId: "session-1",
    turnId: "turn-1",
    type: "system_note",
    role: "system",
    visibility: "normal",
    content: "completed",
    payload: {
      kind: "host_tool_selection",
      status: "completed",
      lifetime: "conversation",
      resources: [{ kind: "mcp", id: "docs", name: "Docs", toolCount: 4 }],
    },
    timeCreated: new Date(2).toISOString(),
  });
  assert.equal(completed[0].systemNotes?.length, 1);
  assert.equal(completed[0].systemNotes?.[0]?.content, "completed");
  assert.deepEqual(completed[0].systemNotes?.[0]?.payload?.resources, [
    { kind: "mcp", id: "docs", name: "Docs", toolCount: 4 },
  ]);
});
