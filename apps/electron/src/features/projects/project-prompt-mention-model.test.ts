import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  consumePromptMentionQuery,
  filterPromptMentionItems,
  filterPromptMentionActions,
  groupPromptMentionItems,
  promptMentionCatalogItems,
  promptMentionConversationItems,
  promptMentionRange,
  type PromptMentionItem,
} from "./project-prompt-mention-model.ts";

describe("prompt mention keyword matching", () => {
  function item(label: string, detail = "", token = label): PromptMentionItem {
    return { id: label, label, detail, token, type: "技能",
      reference: { id: label, kind: "skill", token } };
  }

  it("rejects scattered letters and cross-field matches for @hyper", () => {
    const items = [
      item("arkcli-auth", "h xxx y xxx p xxx e xxx r"),
      item("Build Report", "happy report"),
      item("hy", "per"),
      item("hyp", "", "er"),
      item("HyperFrames", "视频生成"),
    ];
    assert.deepEqual(filterPromptMentionItems(items, " HyPeR ").map(i => i.label), ["HyperFrames"]);
    assert.deepEqual(filterPromptMentionItems([item("hy", "per")], "hy per"), []);
  });

  it("ranks exact names, prefixes and contained names ahead of descriptions, without mutating input", () => {
    const items = [
      item("Video", "Use HyperFrames"),
      item("my-hyper-tool"),
      item("HyperFrames"),
      item("超帧", "", "hyper-render"),
      item("hyper"),
    ];
    const original = [...items];
    assert.deepEqual(filterPromptMentionItems(items, "hyper").map(i => i.label), [
      "hyper", "HyperFrames", "超帧", "my-hyper-tool", "Video",
    ]);
    assert.deepEqual(items, original);
    assert.deepEqual(filterPromptMentionItems(items, "  "), original);
  });

  it("matches Chinese and paths continuously and applies the same rule to actions", () => {
    assert.equal(filterPromptMentionItems([item("项目", "/workspace/hyperframes")], "hyper").length, 1);
    assert.equal(filterPromptMentionItems([item("视频生成")], "视频").length, 1);
    assert.equal(filterPromptMentionItems([item("视频生成")], "视成").length, 0);
    assert.equal(filterPromptMentionActions("文件").length, 1);
    assert.equal(filterPromptMentionActions("选夹").length, 0);
    assert.equal(filterPromptMentionActions("夹 添加").length, 0);
  });

  it("ranks the full catalog before applying the ten-result limit", () => {
    const items = Array.from({ length: 15 }, (_, i) => item(`Video ${i}`, "HyperFrames integration"));
    items.push(item("HyperFrames"));
    const results = groupPromptMentionItems(filterPromptMentionItems(items, "hyper"))[0].items;
    assert.equal(results.length, 10);
    assert.equal(results[0].label, "HyperFrames");
    assert.deepEqual(results.slice(1).map(i => i.label), items.slice(0, 9).map(i => i.label));
  });
});

describe("prompt mention result limits", () => {
  const types: PromptMentionItem["type"][] = [
    "项目",
    "对话",
    "技能",
    "工具",
    "扩展",
    "MCP",
  ];
  const items: PromptMentionItem[] = types.flatMap((type) =>
    Array.from({ length: 15 }, (_, index) => ({
      id: `${type}-${index}`,
      label: index >= 10 ? `${type}目标${index}` : `${type}${index}`,
      token: `${type}${index}`,
      type,
      reference: {
        id: `${type}-${index}`,
        kind: "tool" as const,
        token: `${type}${index}`,
      },
    })),
  );

  it("shows at most ten entries per type, preserving their order", () => {
    const groups = groupPromptMentionItems(filterPromptMentionItems(items, ""));
    assert.equal(groups.length, types.length);
    for (const group of groups) {
      assert.equal(group.items.length, 10);
      assert.deepEqual(
        group.items.map((item) => item.id),
        Array.from({ length: 10 }, (_, index) => `${group.type}-${index}`),
      );
    }
  });

  it("filters the full catalog before limiting each type", () => {
    const groups = groupPromptMentionItems(
      filterPromptMentionItems(items, "目标"),
    );
    assert.equal(groups.length, types.length);
    for (const group of groups) {
      assert.deepEqual(
        group.items.map((item) => item.id),
        Array.from({ length: 5 }, (_, index) => `${group.type}-${index + 10}`),
      );
    }
    assert.deepEqual(
      groupPromptMentionItems(filterPromptMentionItems(items, "不存在的资源")),
      [],
    );
  });
});

describe("prompt mention query", () => {
  it("opens for a bare @ and keeps the live query at the caret", () => {
    assert.deepEqual(promptMentionRange("@", 1), { query: "", start: 0 });
    assert.deepEqual(promptMentionRange("before @pro after", 11), {
      query: "pro",
      start: 7,
    });
  });

  it("consumes only the @ query when an item is selected", () => {
    assert.deepEqual(
      consumePromptMentionQuery("before @pro after", 11, {
        query: "pro",
        start: 7,
      }),
      { caret: 7, value: "before after" },
    );
  });
});

describe("prompt mention catalog", () => {
  it("maps enabled desktop resources to selectable mention items", () => {
    const items = promptMentionCatalogItems({
      extensions: [
        {
          enabled: true,
          id: "hyperframes",
          summary: {
            description: "创建和渲染视频",
            name: "HyperFrames",
            tools: ["render_video"],
          },
        },
      ],
      mcpServers: [
        {
          server: {
            description: "项目数据",
            enabled: true,
            id: "analytics",
            name: "Analytics",
          },
          tools: [{ name: "query_metrics" }],
        },
      ],
      skills: [
        {
          description: "分析产品指标",
          enabled: true,
          id: "metrics",
          name: "Metric diagnostics",
        },
      ],
      tools: [
        {
          description: "渲染视频",
          enabled: true,
          name: "render_video",
          source: "extension",
          sourceId: "hyperframes",
        },
        {
          description: "查询指标",
          enabled: true,
          name: "query_metrics",
          source: "mcp",
          sourceId: "analytics",
        },
        {
          enabled: true,
          name: "read",
          source: "builtin",
        },
      ],
    });

    assert.deepEqual(
      items.map((item) => [item.type, item.label]),
      [
        ["扩展", "HyperFrames"],
        ["技能", "Metric diagnostics"],
        ["MCP", "Analytics"],
      ],
    );
    assert.deepEqual(items[0]?.reference.toolNames, ["render_video"]);
    assert.deepEqual(items[2]?.reference.toolNames, ["query_metrics"]);
  });

  it("keeps recent local conversations as a separate resource type", () => {
    const items = promptMentionConversationItems([
      {
        id: "thread-1",
        projectPath: "/workspace/aivo",
        title: "完善桌面端引用功能",
      },
    ]);

    assert.deepEqual(items[0], {
      detail: "/workspace/aivo",
      id: "conversation:thread-1",
      label: "完善桌面端引用功能",
      reference: {
        id: "thread-1",
        kind: "conversation",
        rootPath: "/workspace/aivo",
        token: "完善桌面端引用功能",
      },
      token: "完善桌面端引用功能",
      type: "对话",
    });
  });
});
