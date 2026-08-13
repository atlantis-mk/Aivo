import assert from "node:assert/strict";
import test from "node:test";

import {
  groupToolCatalogEntries,
  isToggleableCatalogTool,
} from "../src/features/projects/project-tool-activation-model.ts";

test("tool activation shows globally enabled tools in their source categories", () => {
  const tools = [
    { name: "read", source: "builtin", enabled: true },
    {
      name: "aivo_projects_list",
      source: "extension",
      sourceId: "aivo.projects",
      enabled: true,
    },
    {
      name: "extension_notes",
      source: "extension",
      sourceId: "notes",
      enabled: true,
    },
    {
      name: "mcp_calendar_list",
      source: "extension",
      sourceId: "mcp_calendar",
      category: "mcp",
      enabled: true,
    },
    { name: "disabled_extension", source: "extension", enabled: false },
    { name: "tool_search", source: "bridge", enabled: true },
  ];
  const visible = tools.filter(isToggleableCatalogTool);

  assert.deepEqual(
    visible.map((tool) => tool.name),
    ["read", "aivo_projects_list", "extension_notes", "mcp_calendar_list"],
  );
  const groups = groupToolCatalogEntries(visible, {
    "mcp:mcp_calendar": {
      label: "Calendar",
      section: "mcp",
    },
  });
  assert.deepEqual(groups.map((group) => group.section), [
    "tools",
    "extensions",
    "extensions",
    "mcp",
  ]);
  assert.equal(groups.at(-1)?.label, "Calendar");
});
