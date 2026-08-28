import assert from "node:assert/strict";
import test from "node:test";

import {
  appNameFromConfig,
  canSubmitAppName,
  limitAppNameInput,
} from "../src/lib/app-identity.ts";

test("app name falls back to Aivo and trims persisted values", () => {
  assert.equal(appNameFromConfig(undefined), "Aivo");
  assert.equal(appNameFromConfig({ appName: "  小艾  " }), "小艾");
});

test("app name input is bounded by Unicode characters", () => {
  assert.equal(limitAppNameInput("😀".repeat(41)), "😀".repeat(40));
  assert.equal(canSubmitAppName("  小艾  "), true);
  assert.equal(canSubmitAppName("   "), false);
  assert.equal(canSubmitAppName("Aivo\nAgent"), false);
  assert.equal(canSubmitAppName("名".repeat(41)), false);
});
