import assert from "node:assert/strict";
import test from "node:test";

import {
  filterAgentModes,
  filterTools,
} from "../src/features/projects/extension-settings-search.ts";

test("tools tab shows only Aivo built-in tools", () => {
  const visible = filterTools(
    [
      { name: "read", source: "builtin", enabled: true },
      { name: "bash", source: "builtin", enabled: true },
      {
        name: "aivo_projects_query",
        source: "extension",
        sourceId: "aivo.projects",
        enabled: true,
      },
      {
        name: "third_party_notes",
        source: "extension",
        sourceId: "notes",
        enabled: true,
      },
      { name: "mcp_chrome_click", source: "extension", enabled: true },
      {
        name: "aivo_tools_register_mcp",
        source: "extension",
        sourceId: "aivo.tools",
        enabled: true,
      },
      { name: "calendar_list", source: "mcp", enabled: true },
    ],
    "",
  );

  assert.deepEqual(
    visible.map((tool) => tool.name),
    ["read", "bash", "aivo_projects_query"],
  );
});

test("agent mode search includes origin, prompt, model, and associations without toolsets", () => {
  const modes = [
    {
      id: "code",
      displayName: "Code",
      description: "Primary coding mode",
      prompt: "Implement requested changes",
      source: "builtin",
      builtIn: true,
      subagents: ["review"],
    },
    {
      id: "research",
      displayName: "Research",
      description: "Read-only investigator",
      prompt: "Inspect sources",
      source: "user",
      model: { providerId: "openai", modelId: "gpt-research" },
    },
  ];

  assert.deepEqual(
    filterAgentModes(modes, "gpt-research").map((mode) => mode.id),
    ["research"],
  );
  assert.deepEqual(filterAgentModes(modes, "git"), []);
  assert.deepEqual(
    filterAgentModes(modes, "review").map((mode) => mode.id),
    ["code"],
  );
});
