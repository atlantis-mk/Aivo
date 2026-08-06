import assert from "node:assert/strict";
import test from "node:test";

import {
  groupToolCatalogEntries,
  isToggleableCatalogTool,
} from "../src/features/projects/project-tool-activation-model.ts";

test("tool activation lists built-in and extension tools but excludes MCP", () => {
  const tools = [
    { name: "read", source: "builtin", enabled: true },
    { name: "extension_notes", source: "extension", enabled: true },
    { name: "mcp_calendar", source: "mcp", enabled: true },
    { name: "tool_search", source: "bridge", enabled: true },
  ];
  const visible = tools.filter(isToggleableCatalogTool);

  assert.deepEqual(
    visible.map((tool) => tool.name),
    ["read", "extension_notes"],
  );
  assert.deepEqual(
    groupToolCatalogEntries(visible, {}).map((group) => group.section),
    ["tools", "extensions"],
  );
});
