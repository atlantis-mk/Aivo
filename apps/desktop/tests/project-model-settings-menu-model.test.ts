import assert from "node:assert/strict";
import test from "node:test";

import {
  reasoningEffortOptionsForModel,
  serviceTierOptionsForModel,
} from "../src/features/projects/project-model-settings-menu-model.ts";

test("model settings use dynamically declared Codex reasoning efforts", () => {
  assert.deepEqual(
    reasoningEffortOptionsForModel({
      id: "gpt-codex",
      providerId: "openai",
      name: "Codex",
      supportedReasoningEfforts: ["minimal", "medium", "xhigh"],
    }).map((option) => option.value),
    ["minimal", "medium", "xhigh"],
  );
});

test("model settings map declared fast and flex service tiers", () => {
  assert.deepEqual(
    serviceTierOptionsForModel({
      id: "gpt-codex",
      providerId: "openai",
      name: "Codex",
      serviceTiers: ["fast", "flex"],
    }).map((option) => option.value),
    ["default", "priority", "flex"],
  );
});
