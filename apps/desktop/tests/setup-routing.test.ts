import assert from "node:assert/strict";
import test from "node:test";

import {
  hasCompletedInitialization,
  startupRouteFor,
} from "../src/features/setup/setup-routing.ts";

test("fresh installs enter setup", () => {
  assert.equal(hasCompletedInitialization(null), false);
  assert.equal(
    hasCompletedInitialization({ initialized: false }),
    false,
  );
  assert.equal(startupRouteFor({ initialized: false }), "/setup");
});

test("completed initialization opens the chat workspace", () => {
  const config = {
    initialized: true,
    initialWorkspacePath: "/Users/aivo/Documents/Aivo-Workspaces",
  };

  assert.equal(hasCompletedInitialization(config), true);
  assert.equal(startupRouteFor(config), "/projects/chat");
});

test("legacy initialized state without a workspace still finishes setup", () => {
  assert.equal(
    hasCompletedInitialization({
      initialized: true,
      initialWorkspacePath: "  ",
    }),
    false,
  );
  assert.equal(
    startupRouteFor({ initialized: true }),
    "/setup",
  );
});
