import assert from "node:assert/strict";
import test from "node:test";

import {
  filterAgentModes,
  filterTools,
  selectVisibleSkillCandidates,
} from "../src/features/projects/extension-settings-search.ts";

test("tools tab shows only manageable Aivo tools", () => {
  const visible = filterTools(
    [
      { name: "read", source: "builtin", enabled: true },
      { name: "bash", source: "builtin", enabled: true },
      { name: "grep", source: "builtin", enabled: true },
      { name: "find", source: "builtin", enabled: true },
      { name: "ls", source: "builtin", enabled: true },
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
    [
      "grep",
      "find",
      "ls",
      "aivo_projects_query",
      "aivo_tools_register_mcp",
    ],
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

test("skill candidates are unique by name and hidden while installed", () => {
  const candidates = [
    {
      id: "claude-review",
      name: "code-review",
      description: "Claude candidate",
      scope: "global",
      source: "claude",
      rootPath: "/skills/claude/code-review",
      skillPath: "/skills/claude/code-review/SKILL.md",
      contentHash: "claude-hash",
      status: "imported",
      lastSeenAt: "2026-08-26T10:00:00Z",
    },
    {
      id: "codex-review",
      name: "code-review",
      description: "Codex candidate",
      scope: "global",
      source: "codex",
      rootPath: "/skills/codex/code-review",
      skillPath: "/skills/codex/code-review/SKILL.md",
      contentHash: "codex-hash",
      status: "pending",
      lastSeenAt: "2026-08-26T09:00:00Z",
    },
    {
      id: "pdf",
      name: "pdf",
      description: "PDF candidate",
      scope: "global",
      source: "agents",
      rootPath: "/skills/agents/pdf",
      skillPath: "/skills/agents/pdf/SKILL.md",
      contentHash: "pdf-hash",
      status: "pending",
      lastSeenAt: "2026-08-26T08:00:00Z",
    },
  ];

  const installed = [
    {
      id: "installed-review",
      name: "Code-Review",
      description: "Installed review skill",
      scope: "global",
      source: "aivo",
      rootPath: "/skills/aivo/code-review",
      skillPath: "/skills/aivo/code-review/SKILL.md",
      contentHash: "installed-hash",
      enabled: true,
      timeCreated: "2026-08-26T10:00:00Z",
      timeUpdated: "2026-08-26T10:00:00Z",
    },
  ];

  assert.deepEqual(
    selectVisibleSkillCandidates(candidates, installed).map(
      (candidate) => candidate.id,
    ),
    ["pdf"],
  );
  assert.deepEqual(
    selectVisibleSkillCandidates(candidates, []).map(
      (candidate) => candidate.id,
    ),
    ["claude-review", "pdf"],
  );
});
