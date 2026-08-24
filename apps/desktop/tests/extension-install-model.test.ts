import assert from "node:assert/strict";
import test from "node:test";

import {
  extensionCapabilityBadges,
  extensionRuntimeLabel,
  extensionStatusLabel,
} from "../src/features/projects/extension-install-model.ts";

test("presents extension runtime and lifecycle states in the install UI", () => {
  assert.equal(extensionRuntimeLabel("process"), "本地进程");
  assert.equal(extensionRuntimeLabel("static"), "静态界面");
  assert.equal(extensionStatusLabel("running", true), "运行中");
  assert.equal(extensionStatusLabel("running", false), "已停用");
  assert.equal(extensionStatusLabel("error", false), "需要处理");
});

test("summarizes only contributed extension capability groups", () => {
  assert.deepEqual(
    extensionCapabilityBadges({
      id: "com.example.ui",
      name: "Example UI",
      version: "1.0.0",
      apiVersion: "2",
      runtimeType: "static",
      executable: false,
      tools: ["render"],
      views: ["panel", "detail"],
      contexts: [],
    }),
    ["1 个工具", "2 个界面"],
  );
});
