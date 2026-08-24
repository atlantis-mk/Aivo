import assert from "node:assert/strict";
import test from "node:test";

import { resolveSetupWorkspacePath } from "../src/features/setup/setup-workspace-path.ts";

test("fresh setup uses the whitespace-free default workspace path", () => {
  assert.equal(
    resolveSetupWorkspacePath({
      defaultInitialWorkspacePath: "/Users/aivo/Documents/Aivo-Workspaces",
    }),
    "/Users/aivo/Documents/Aivo-Workspaces",
  );
});

test("setup replaces the exact legacy built-in default for confirmation", () => {
  assert.equal(
    resolveSetupWorkspacePath({
      initialWorkspacePath: "/Users/aivo/Documents/Aivo Workspaces",
      defaultInitialWorkspacePath: "/Users/aivo/Documents/Aivo-Workspaces",
    }),
    "/Users/aivo/Documents/Aivo-Workspaces",
  );
  assert.equal(
    resolveSetupWorkspacePath({
      initialWorkspacePath: "C:\\Users\\aivo\\Documents\\Aivo Workspaces",
      defaultInitialWorkspacePath: "C:\\Users\\aivo\\Documents\\Aivo-Workspaces",
    }),
    "C:\\Users\\aivo\\Documents\\Aivo-Workspaces",
  );
});

test("setup preserves a user-selected custom workspace path", () => {
  assert.equal(
    resolveSetupWorkspacePath({
      initialWorkspacePath: "/Users/aivo/Projects/My Workspace",
      defaultInitialWorkspacePath: "/Users/aivo/Documents/Aivo-Workspaces",
    }),
    "/Users/aivo/Projects/My Workspace",
  );
});
