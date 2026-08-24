import assert from "node:assert/strict";
import test from "node:test";

import {
  extensionToolViewContext,
  extensionToolViewIdentity,
  extensionToolViewRef,
  latestExtensionViewToolCallId,
  selectedExtensionViewToolCallId,
} from "../src/features/projects/extension-tool-view-model.ts";

function toolCall(id: string, view?: Record<string, unknown>) {
  return {
    id,
    result: view ? { details: { view } } : {},
  };
}

test("reads only bounded Host extension tool view identities", () => {
  assert.deepEqual(
    extensionToolViewRef(
      toolCall("call-1", {
        extensionId: "com.example.github",
        viewId: "issue-detail",
        surface: "tool-detail",
        title: "Issue detail",
      }),
    ),
    {
      extensionId: "com.example.github",
      viewId: "issue-detail",
      surface: "tool-detail",
      title: "Issue detail",
    },
  );
  assert.equal(
    extensionToolViewRef(
      toolCall("call-2", {
        extensionId: "https://example.com",
        viewId: "detail",
        surface: "tool-detail",
      }),
    ),
    null,
  );
  assert.equal(
    extensionToolViewRef(
      toolCall("call-3", {
        extensionId: "com.example.github",
        viewId: "detail",
        surface: "dialog",
      }),
    ),
    null,
  );
});

test("keeps view identity stable while tool-call context changes", () => {
  const firstView = extensionToolViewRef(
    toolCall("first", {
      extensionId: "com.example",
      viewId: "detail",
      surface: "tool-detail",
      title: "First title",
    }),
  );
  const nextView = extensionToolViewRef(
    toolCall("next", {
      extensionId: "com.example",
      viewId: "detail",
      surface: "tool-detail",
      title: "Updated title",
    }),
  );
  assert.ok(firstView);
  assert.ok(nextView);
  assert.equal(
    extensionToolViewIdentity(firstView),
    extensionToolViewIdentity(nextView),
  );
  assert.notDeepEqual(
    extensionToolViewContext({
      id: "first",
      sessionId: "session-1",
      turnId: "turn-1",
      name: "ui_test.render_panel",
    }),
    extensionToolViewContext({
      id: "next",
      sessionId: "session-1",
      turnId: "turn-2",
      name: "ui_test.render_panel",
    }),
  );
});

test("selects the latest tool call that has a custom view", () => {
  assert.equal(
    latestExtensionViewToolCallId([
      { toolCall: toolCall("first", { extensionId: "com.example", viewId: "one", surface: "page" }) },
      { toolCall: toolCall("native") },
      { toolCall: toolCall("last", { extensionId: "com.example", viewId: "two", surface: "tool-detail" }) },
    ]),
    "last",
  );
});

test("selects the next activity view atomically without an empty render", () => {
  assert.equal(
    selectedExtensionViewToolCallId({
      activityId: "activity-next",
      latestViewToolCallId: "call-next",
      selectedToolCallId: "call-previous",
      trackedActivityId: "activity-previous",
    }),
    "call-next",
  );
  assert.equal(
    selectedExtensionViewToolCallId({
      activityId: "activity-current",
      latestViewToolCallId: "call-auto",
      selectedToolCallId: "",
      trackedActivityId: "activity-current",
    }),
    "",
  );
});
