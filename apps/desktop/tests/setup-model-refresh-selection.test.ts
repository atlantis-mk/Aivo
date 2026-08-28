import assert from "node:assert/strict";
import test from "node:test";

import { modelSelectionAfterCatalogRefresh } from "../src/features/setup/setup-model-refresh-selection.ts";

test("provider catalog refresh retains the model chosen during setup", () => {
  assert.equal(
    modelSelectionAfterCatalogRefresh("gpt-5.5-codex", "gpt-5.5"),
    "gpt-5.5-codex",
  );
});

test("provider catalog refresh supplies a default only before selection", () => {
  assert.equal(modelSelectionAfterCatalogRefresh("  ", "gpt-5.5"), "gpt-5.5");
});
