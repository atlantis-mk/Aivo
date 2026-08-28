import assert from "node:assert/strict";
import test from "node:test";

import { normalizePermissionMode } from "../src/features/projects/project-model-options.ts";

test("removed and unknown permission modes fall back to request approval", () => {
  assert.equal(normalizePermissionMode("auto_approve"), "request_approval");
  assert.equal(normalizePermissionMode("unknown"), "request_approval");
  assert.equal(normalizePermissionMode(undefined), "request_approval");
});

test("supported permission modes remain unchanged", () => {
  assert.equal(normalizePermissionMode("request_approval"), "request_approval");
  assert.equal(normalizePermissionMode("full_access"), "full_access");
});
