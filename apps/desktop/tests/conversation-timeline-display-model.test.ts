import assert from "node:assert/strict";
import test from "node:test";

import {
  COLLAPSED_USER_MESSAGE_HEIGHT,
  formatPendingAssistantStatus,
  shouldShowUserMessageDisclosure,
} from "../src/features/projects/conversation-timeline-display-model.ts";

test("user message disclosure is based on measured hidden content", () => {
  const longVisibleMessage = "使用 videospec 管理流程。".repeat(24);

  assert.equal(longVisibleMessage.length > 320, true);
  assert.equal(
    shouldShowUserMessageDisclosure(
      COLLAPSED_USER_MESSAGE_HEIGHT - 1,
    ),
    false,
  );
  assert.equal(
    shouldShowUserMessageDisclosure(COLLAPSED_USER_MESSAGE_HEIGHT),
    false,
  );
  assert.equal(
    shouldShowUserMessageDisclosure(COLLAPSED_USER_MESSAGE_HEIGHT + 1),
    true,
  );
});

test("user message disclosure waits for measurement", () => {
  assert.equal(shouldShowUserMessageDisclosure(null), false);
  assert.equal(
    shouldShowUserMessageDisclosure(COLLAPSED_USER_MESSAGE_HEIGHT + 1),
    true,
  );
});

test("pending assistant status distinguishes thinking from execution", () => {
  assert.equal(formatPendingAssistantStatus(false), "正在思考");
  assert.equal(formatPendingAssistantStatus(true), "正在执行");
});
