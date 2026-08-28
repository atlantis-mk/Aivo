import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultActiveBuiltinToolNames,
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
      name: "extension_notes_write",
      source: "extension",
      sourceId: "notes",
      enabled: true,
    },
    {
      name: "mcp_calendar_list",
      source: "extension",
      sourceId: "mcp_calendar",
      category: "mcp",
      selectionGroup: {
        id: "mcp_group_calendar",
        name: "Calendar",
        description: "Read and update calendars",
      },
      enabled: true,
    },
    {
      name: "mcp_calendar_update",
      source: "extension",
      sourceId: "mcp_calendar",
      category: "mcp",
      selectionGroup: {
        id: "mcp_group_calendar",
        name: "Calendar",
        description: "Read and update calendars",
      },
      enabled: true,
    },
    { name: "disabled_extension", source: "extension", enabled: false },
    { name: "tool_search", source: "bridge", enabled: true },
  ];
  const visible = tools.filter(isToggleableCatalogTool);

  assert.deepEqual(
    visible.map((tool) => tool.name),
    [
      "aivo_projects_list",
      "extension_notes",
      "extension_notes_write",
      "mcp_calendar_list",
      "mcp_calendar_update",
    ],
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
  assert.deepEqual(
    groups.map((group) => ({
      grouped: group.grouped,
      label: group.label,
      tools: group.tools.map((tool) => tool.name),
    })),
    [
      {
        grouped: false,
        label: "aivo_projects_list",
        tools: ["aivo_projects_list"],
      },
      {
        grouped: false,
        label: "extension_notes",
        tools: ["extension_notes"],
      },
      {
        grouped: false,
        label: "extension_notes_write",
        tools: ["extension_notes_write"],
      },
      {
        grouped: true,
        label: "Calendar",
        tools: ["mcp_calendar_list", "mcp_calendar_update"],
      },
    ],
  );
});

test("required core tools stay active even when legacy global state hides one", () => {
  const tools = [
    { name: "read", source: "builtin", enabled: true },
    { name: "bash", source: "builtin", enabled: true },
    { name: "edit", source: "builtin", enabled: false },
    { name: "write", source: "builtin", enabled: true },
    { name: "grep", source: "builtin", enabled: true },
    { name: "find", source: "builtin", enabled: true },
    { name: "ls", source: "builtin", enabled: true },
  ];

  assert.deepEqual(defaultActiveBuiltinToolNames(tools), [
    "read",
    "bash",
    "edit",
    "write",
  ]);
});
