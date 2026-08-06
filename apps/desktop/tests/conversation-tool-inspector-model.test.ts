import assert from "node:assert/strict";
import test from "node:test";

import { toolTimelineDescription } from "../src/features/projects/conversation-tool-inspector-model.ts";

test("shows a shared invocation description only on its first timeline row", () => {
  assert.equal(
    toolTimelineDescription("检查运行环境", ""),
    "检查运行环境",
  );
  assert.equal(
    toolTimelineDescription(
      "检查运行环境",
      "检查运行环境",
    ),
    "",
  );
});

test("omits timeline titles when invocation descriptions are missing", () => {
  assert.equal(toolTimelineDescription("", ""), "");
  assert.equal(toolTimelineDescription(" 相同说明 ", "相同说明"), "");
  assert.equal(
    toolTimelineDescription("下一步注册", "检查运行环境"),
    "下一步注册",
  );
});
