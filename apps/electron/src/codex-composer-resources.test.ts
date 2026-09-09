import assert from "node:assert/strict";
import { test } from "node:test";
import {
  loadCodexSkills,
  loadCodexMcpServers,
  codexResourceInputs,
} from "./codex-composer-resources.ts";
import {
  promptMentionCodexSkillItems,
  promptMentionCodexMcpItems,
  groupPromptMentionItems,
  filterPromptMentionItems,
} from "./features/projects/project-prompt-mention-model.ts";
import {
  parsePromptSubmission,
  mentionHref,
} from "./features/projects/project-prompt-editor-model.ts";

test("skills/list maps real metadata, retains scan errors and filters disabled skills", async () => {
  const result = await loadCodexSkills(
    async (method, params) => {
      assert.equal(method, "skills/list");
      assert.deepEqual(params, { cwds: ["/workspace"], forceReload: true });
      return {
        data: [
          {
            cwd: "/workspace",
            skills: [
              {
                name: "design",
                path: "/skills/design/SKILL.md",
                description: "Full description",
                enabled: true,
                interface: {
                  displayName: "设计",
                  shortDescription: "设计页面",
                },
              },
              {
                name: "disabled",
                path: "/skills/off/SKILL.md",
                enabled: false,
              },
            ],
            errors: [
              { path: "/skills/broken/SKILL.md", message: "invalid metadata" },
            ],
          },
        ],
      };
    },
    "/workspace",
    true,
  );
  const items = promptMentionCodexSkillItems(result.skills);
  assert.equal(items.length, 1);
  assert.equal(items[0].label, "设计");
  assert.equal(items[0].detail, "设计页面");
  assert.deepEqual(items[0].reference.input, {
    type: "skill",
    name: "design",
    path: "/skills/design/SKILL.md",
  });
  assert.match(result.errors[0], /invalid metadata/);
});

test("MCP reads every page before searching and keeps servers without discovered tools", async () => {
  let requests = 0;
  const servers = await loadCodexMcpServers(async (method, params) => {
    assert.equal(method, "mcpServerStatus/list");
    requests++;
    if (!params.cursor)
      return {
        data: Array.from({ length: 12 }, (_, i) => ({
          name: `server${i}`,
          tools: { ping: { name: "ping" } },
          runtimeStatus: "connected",
        })),
        nextCursor: "page2",
      };
    assert.equal(params.cursor, "page2");
    return {
      data: [
        {
          name: "older",
          tools: {},
          toolsError: "needs login",
          runtimeStatus: "authenticationRequired",
        },
        { name: "off", tools: {}, runtimeStatus: "disabled" },
      ],
      nextCursor: null,
    };
  });
  assert.equal(requests, 2);
  const items = promptMentionCodexMcpItems(servers);
  assert.equal(items.length, 13);
  assert.equal(groupPromptMentionItems(items)[0].items.length, 10);
  const found = groupPromptMentionItems(
    filterPromptMentionItems(items, "older"),
  )[0].items[0];
  assert.equal(found.reference.input?.path, "mcp://older");
  assert.match(found.detail!, /needs login/);
});

test("catalog failures propagate instead of silently becoming empty lists", async () => {
  await assert.rejects(
    loadCodexSkills(async () => {
      throw new Error("offline");
    }),
    /offline/,
  );
  await assert.rejects(
    loadCodexMcpServers(async () => ({})),
    /无效/,
  );
  await assert.rejects(
    loadCodexMcpServers(async () => ({ data: [], nextCursor: "same" })),
    /重复/,
  );
});

test("codex_apps splits by real connector ID, not tool namespace, with native submission", async () => {
  const tool = (id: string, name: string) => ({
    _meta: {
      connector_id: id,
      connector_name: name,
      connector_description: `${name} tools`,
    },
  });
  const servers = await loadCodexMcpServers(async () => ({
    data: [
      {
        name: "codex_apps",
        tools: {
          "github.fetch": tool("connector-github", "GitHub"),
          "github.search": tool("connector-github", "GitHub"),
          "another_namespace.read": tool("connector-github", "GitHub"),
          "notion.fetch": tool("asdk-notion", "Notion"),
          "unknown.run": {},
        },
      },
    ],
  }));
  const items = promptMentionCodexMcpItems(servers);
  assert.deepEqual(
    items.map((item) => item.label),
    ["GitHub", "Notion"],
  );
  assert.equal(items[0].reference.toolNames?.length, 3);
  assert.ok(
    items.every((item) => item.reference.input?.path !== "mcp://codex_apps"),
  );
  const references = items.map((item) => item.reference);
  const markdown = references
    .map((ref) => `[${ref.token}](${mentionHref(ref)})`)
    .join(" 和 ");
  assert.deepEqual(
    codexResourceInputs(parsePromptSubmission(markdown, references).references),
    [
      { type: "mention", name: "GitHub", path: "app://connector-github" },
      { type: "mention", name: "Notion", path: "app://asdk-notion" },
    ],
  );
  assert.deepEqual(
    codexResourceInputs(parsePromptSubmission("正文", references).references),
    [],
  );
});

test("split apps share the ten-result MCP limit and search includes apps beyond ten", async () => {
  const servers = await loadCodexMcpServers(async () => ({
    data: [
      {
        name: "codex_apps",
        tools: Object.fromEntries(
          Array.from({ length: 14 }, (_, i) => [
            `app${i}.run`,
            {
              _meta: {
                connector_id: `id${i}`,
                connector_name: `App ${String(i).padStart(2, "0")}`,
              },
            },
          ]),
        ),
      },
    ],
  }));
  const items = promptMentionCodexMcpItems([
    { name: "local", displayName: "Local MCP", toolNames: [] },
    ...servers,
  ]);
  assert.equal(items.length, 15);
  const groups = groupPromptMentionItems(items);
  assert.equal(groups.length, 1);
  assert.equal(groups[0].items.length, 10);
  assert.ok(!groups[0].items.some((item) => item.label === "App 13"));
  const matches = groupPromptMentionItems(
    filterPromptMentionItems(items, "App 13"),
  );
  assert.equal(matches[0].items.length, 1);
  assert.ok(
    matches[0].items.some(
      (item) => item.reference.input?.path === "app://id13",
    ),
  );
});

test("disabled or unidentified aggregate apps never become a selectable server fallback", () => {
  assert.deepEqual(
    promptMentionCodexMcpItems([
      {
        name: "codex_apps",
        displayName: "codex_apps",
        toolNames: ["unknown.run"],
      },
      {
        name: "codex_apps",
        displayName: "codex_apps",
        runtimeStatus: "disabled",
        toolNames: [],
        apps: [{ id: "off", name: "Off", description: "", toolNames: [] }],
      },
    ]),
    [],
  );
});

test("native resource identity survives Markdown submission and deleted references are excluded", () => {
  const skill = promptMentionCodexSkillItems([
    {
      name: "design",
      displayName: "设计",
      path: "/skills/design/SKILL.md",
      description: "",
      enabled: true,
    },
  ])[0].reference;
  const mcp = promptMentionCodexMcpItems([
    { name: "docs", displayName: "文档", toolNames: [] },
  ])[0].reference;
  const submitted = parsePromptSubmission(
    `[设计](${mentionHref(skill)}) 和 [文档](${mentionHref(mcp)})`,
    [skill, mcp],
  );
  assert.deepEqual(codexResourceInputs(submitted.references), [
    { type: "skill", name: "design", path: "/skills/design/SKILL.md" },
    { type: "mention", name: "docs", path: "mcp://docs" },
  ]);
  assert.deepEqual(
    codexResourceInputs(parsePromptSubmission("正文", [skill, mcp]).references),
    [],
  );
  assert.equal(codexResourceInputs([skill, skill]).length, 1);
});
