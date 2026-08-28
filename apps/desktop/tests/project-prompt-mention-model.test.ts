import assert from "node:assert/strict";
import test from "node:test";

import {
  activePromptMentionReferences,
  addPromptMentionReference,
  consumePromptMentionQuery,
  filterPromptMentionActions,
  filterPromptMentionItems,
  groupPromptMentionItems,
  isPromptMentionBuiltinTool,
  promptComposerEnterAction,
  promptMentionProjectItems,
  promptMentionRange,
  removePromptMentionReference,
} from "../src/features/projects/project-prompt-mention-model.ts";

test("composer mentions expose context compaction and local resource actions", () => {
  assert.deepEqual(
    filterPromptMentionActions("").map(({ action, label }) => ({ action, label })),
    [
      { action: "compact-context", label: "压缩上下文" },
      { action: "select-local", label: "选择文件或文件夹" },
    ],
  );
  assert.deepEqual(
    filterPromptMentionActions("文件夹").map((item) => item.action),
    ["select-local"],
  );
  assert.deepEqual(
    filterPromptMentionActions("compact").map((item) => item.action),
    ["compact-context"],
  );
  assert.deepEqual(
    filterPromptMentionActions("压缩").map((item) => item.action),
    ["compact-context"],
  );
});

test("composer mentions fuzzy-match catalog labels and descriptions", () => {
  const items = filterPromptMentionItems([
    { id: "skill:pdf", label: "PDF", reference: { id: "pdf", kind: "skill", token: "PDF" }, token: "PDF", type: "技能", detail: "Read and verify PDF files" },
    { id: "tool:write", label: "write", reference: { id: "write", kind: "tool", token: "write" }, token: "write", type: "工具" },
  ], "rvp");

  assert.deepEqual(items.map((item) => item.id), ["skill:pdf"]);
});

test("composer project mentions include existing projects with the current project first", () => {
  const items = promptMentionProjectItems([
    { id: "a", name: "Alpha", rootPath: "/work/alpha" },
    { id: "b", name: "Beta", rootPath: "/work/beta" },
  ], "/work/beta");

  assert.deepEqual(items.map((item) => item.label), ["Beta", "Alpha"]);
  assert.equal(items[0]?.detail, "/work/beta · 当前项目");
  assert.equal(items[0]?.token, "Beta");
  assert.deepEqual(items[0]?.reference, {
    id: "b",
    kind: "project",
    rootPath: "/work/beta",
    token: "Beta",
  });
});

test("composer tool mentions omit required core tools and include optional enabled tools", () => {
  assert.equal(
    isPromptMentionBuiltinTool({ enabled: true, name: "read", source: "builtin" }),
    false,
  );
  assert.equal(
    isPromptMentionBuiltinTool({ enabled: true, name: "grep", source: "builtin" }),
    true,
  );
  assert.equal(isPromptMentionBuiltinTool({ enabled: false, source: "builtin" }), false);
  assert.equal(
    isPromptMentionBuiltinTool({
      activationPolicy: "provider_declaration",
      enabled: true,
      source: "builtin",
    }),
    false,
  );
  assert.equal(isPromptMentionBuiltinTool({ enabled: true, source: "extension" }), false);
  assert.equal(
    isPromptMentionBuiltinTool({
      enabled: true,
      name: "aivo_projects_query",
      source: "extension",
      sourceId: "aivo.projects",
    }),
    true,
  );
  assert.equal(isPromptMentionBuiltinTool({ enabled: true, source: "mcp" }), false);
});

test("composer mention results are grouped in a stable resource order", () => {
  const groups = groupPromptMentionItems([
    { id: "tool:write", label: "write", reference: { id: "write", kind: "tool", token: "write" }, token: "write", type: "工具" },
    { id: "mcp:calendar", label: "Calendar", reference: { id: "calendar", kind: "mcp", token: "calendar" }, token: "calendar", type: "MCP" },
    { id: "skill:pdf", label: "PDF", reference: { id: "pdf", kind: "skill", token: "PDF" }, token: "PDF", type: "技能" },
    { id: "extension:notes", label: "Notes", reference: { id: "notes", kind: "extension", token: "Notes" }, token: "Notes", type: "扩展" },
    { id: "project:current", label: "当前项目", reference: { id: "current", kind: "project", rootPath: "/work/current", token: "当前项目" }, token: "当前项目", type: "项目" },
  ]);

  assert.deepEqual(groups.map((group) => group.type), [
    "项目",
    "技能",
    "扩展",
    "MCP",
    "工具",
  ]);
  assert.deepEqual(
    groups.flatMap((group) => group.items.map((item) => item.id)),
    [
      "project:current",
      "skill:pdf",
      "extension:notes",
      "mcp:calendar",
      "tool:write",
    ],
  );
});

test("composer submits each chooser-selected reference once", () => {
  const references = activePromptMentionReferences([
    { id: "pdf", kind: "skill", token: "PDF" },
    { id: "pdf", kind: "skill", token: "PDF" },
    { id: "write", kind: "tool", token: "write" },
    { id: "notes", kind: "extension", token: "Notes" },
  ]);

  assert.deepEqual(references.map(({ id, kind }) => ({ id, kind })), [
    { id: "pdf", kind: "skill" },
    { id: "write", kind: "tool" },
    { id: "notes", kind: "extension" },
  ]);
});

test("composer reference tags deduplicate resources and keep one project", () => {
  const pdf = { id: "pdf", kind: "skill" as const, token: "PDF" };
  const alpha = { id: "alpha", kind: "project" as const, rootPath: "/alpha", token: "Alpha" };
  const beta = { id: "beta", kind: "project" as const, rootPath: "/beta", token: "Beta" };
  const selected = addPromptMentionReference(
    addPromptMentionReference(
      addPromptMentionReference([pdf], alpha),
      pdf,
    ),
    beta,
  );

  assert.deepEqual(selected, [pdf, beta]);
  assert.deepEqual(removePromptMentionReference(selected, pdf), [beta]);
});

test("composer enter behavior keeps newline, submit, and mention selection distinct", () => {
  assert.equal(promptComposerEnterAction(false, false, false), "submit");
  assert.equal(promptComposerEnterAction(false, true, false), "newline");
  assert.equal(promptComposerEnterAction(true, false, false), "mention");
  assert.equal(promptComposerEnterAction(true, true, false), "newline");
  assert.equal(promptComposerEnterAction(true, false, true), "none");
});

test("composer consumes the active @ query before rendering its reference tag", () => {
  const value = "请使用 @pd 完成";
  const range = promptMentionRange(value, 7);
  assert.deepEqual(range, { start: 4, query: "pd" });
  assert.deepEqual(consumePromptMentionQuery(value, 7, range!), {
    caret: 4,
    value: "请使用 完成",
  });
  assert.deepEqual(consumePromptMentionQuery("@pd 后续", 3, { start: 0, query: "pd" }), {
    caret: 0,
    value: "后续",
  });
  assert.equal(promptMentionRange("email@test", 10), null);
});
