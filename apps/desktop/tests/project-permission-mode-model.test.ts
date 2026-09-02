import assert from "node:assert/strict";
import test from "node:test";

import {
  normalizePermissionMode,
  normalizeWebSearchMode,
  webSearchModeLabel,
} from "../src/features/projects/project-model-options.ts";

test("removed and unknown permission modes fall back to request approval", () => {
  assert.equal(normalizePermissionMode("auto_approve"), "request_approval");
  assert.equal(normalizePermissionMode("unknown"), "request_approval");
  assert.equal(normalizePermissionMode(undefined), "request_approval");
});

test("supported permission modes remain unchanged", () => {
  assert.equal(normalizePermissionMode("request_approval"), "request_approval");
  assert.equal(normalizePermissionMode("full_access"), "full_access");
});

test("unknown web search modes fall back to live search", () => {
  assert.equal(normalizeWebSearchMode("unknown"), "live");
  assert.equal(normalizeWebSearchMode(undefined), "live");
});

test("supported web search modes remain unchanged", () => {
  assert.equal(normalizeWebSearchMode("disabled"), "disabled");
  assert.equal(normalizeWebSearchMode("cached"), "cached");
  assert.equal(normalizeWebSearchMode("indexed"), "indexed");
  assert.equal(normalizeWebSearchMode("live"), "live");
});

test("web search mode labels describe the active mode", () => {
  assert.equal(webSearchModeLabel("disabled"), "关闭");
  assert.equal(webSearchModeLabel("cached"), "缓存");
  assert.equal(webSearchModeLabel("indexed"), "索引");
  assert.equal(webSearchModeLabel("live"), "实时");
});
