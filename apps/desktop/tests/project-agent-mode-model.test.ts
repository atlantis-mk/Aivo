import assert from "node:assert/strict";
import test from "node:test";

import {
  agentModeShortLabel,
  fallbackAgentModes,
  normalizeAgentMode,
} from "../src/features/projects/project-agent-mode-model.ts";

test("fresh composer fallback exposes Assistant only", () => {
  assert.deepEqual(
    fallbackAgentModes.map((mode) => mode.id),
    ["assistant"],
  );
});

test("retired built-in modes fall back to Assistant in the composer", () => {
  for (const mode of ["code", "build", "explore", "plan", "planner", "review", "debug"]) {
    assert.equal(normalizeAgentMode(mode), "assistant");
  }
  assert.equal(normalizeAgentMode(undefined), "assistant");
});

test("project modes reusing retired IDs keep their configured display name", () => {
  assert.equal(
    agentModeShortLabel({
      id: "code",
      displayName: "Project specialist",
      description: "Project-defined mode",
      prompt: "",
    }),
    "Project specialist",
  );
});
