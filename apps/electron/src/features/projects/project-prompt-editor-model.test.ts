import assert from "node:assert/strict";
import { test } from "node:test";
import { TextSelection } from "prosemirror-state";
import { undo, redo, closeHistory } from "prosemirror-history";
import {
  createPromptEditorState,
  editorMentionQuery,
  insertPromptMention,
  insertPromptParagraph,
  mentionHref,
  parsePromptDocument,
  parsePromptSubmission,
  promptDocumentReferences,
  serializePromptDocument,
} from "./project-prompt-editor-model.ts";
import type { PromptMentionReference } from "./project-prompt-mention-model.ts";

const project: PromptMentionReference = {
  id: "project/中文",
  kind: "project",
  rootPath: "/workspace/aivo",
  token: "Aivo",
};
const skill: PromptMentionReference = {
  id: "skill1",
  kind: "skill",
  token: "编写 [文档]",
};

test("insert a mention at the caret without moving surrounding text; round trip its identity", () => {
  let state = createPromptEditorState(parsePromptDocument("前文 @Ai 后文"));
  state = state.apply(
    state.tr.setSelection(TextSelection.create(state.doc, 7)),
  );
  assert.deepEqual(editorMentionQuery(state), { from: 4, to: 7, query: "Ai" });
  state = state.apply(insertPromptMention(state, 4, 7, project));
  assert.equal(state.doc.textContent, "前文 Aivo  后文");
  assert.equal(state.selection.from, 6);
  const markdown = serializePromptDocument(state.doc);
  assert.ok(markdown.startsWith("前文 [Aivo](aivo-mention://project/"));
  assert.deepEqual(
    parsePromptDocument(markdown, [project]).toJSON(),
    state.doc.toJSON(),
  );
  assert.deepEqual(parsePromptSubmission(markdown, [project]), {
    text: "前文 Aivo  后文",
    references: [project],
  });
});

test("deletion and undo/redo restore the same resource metadata", () => {
  let state = createPromptEditorState(parsePromptDocument("@"));
  state = state.apply(insertPromptMention(state, 1, 2, project));
  state = state.apply(closeHistory(state.tr).delete(1, 2));
  assert.deepEqual(promptDocumentReferences(state.doc), []);
  assert.ok(
    undo(state, (tr) => {
      state = state.apply(tr);
    }),
  );
  assert.deepEqual(promptDocumentReferences(state.doc), [project]);
  assert.ok(
    redo(state, (tr) => {
      state = state.apply(tr);
    }),
  );
  assert.deepEqual(promptDocumentReferences(state.doc), []);
});

test("Shift+Enter creates an editable paragraph and @ works there", () => {
  let state = createPromptEditorState(parsePromptDocument("第一行"));
  state = state.apply(state.tr.setSelection(TextSelection.atEnd(state.doc)));
  insertPromptParagraph(state, (tr) => {
    state = state.apply(tr);
  });
  state = state.apply(state.tr.insertText("@"));
  assert.equal(state.doc.childCount, 2);
  assert.equal(editorMentionQuery(state)?.query, "");
});

test("mention lookup ignores email, code and selected text", () => {
  for (const text of [
    "user@example",
    "`@code`",
    "```\n@code\n```",
    "[link](https://example.com)",
  ]) {
    let state = createPromptEditorState(parsePromptDocument(text));
    state = state.apply(state.tr.setSelection(TextSelection.atEnd(state.doc)));
    assert.equal(editorMentionQuery(state), null);
  }
  let state = createPromptEditorState(parsePromptDocument("@query"));
  state = state.apply(
    state.tr.setSelection(TextSelection.create(state.doc, 1, 7)),
  );
  assert.equal(editorMentionQuery(state), null);
});

test("submission resolves present IDs, ignores deleted references and literal code examples", () => {
  const markdown = `**请处理** [Aivo](${mentionHref(project)})，然后 [文档](${mentionHref(skill)})`;
  assert.deepEqual(
    parsePromptSubmission(markdown, [project, skill]).references,
    [project, skill],
  );
  assert.deepEqual(parsePromptSubmission("只是正文", [project, skill]), {
    text: "只是正文",
    references: [],
  });
  assert.deepEqual(
    parsePromptSubmission("`" + markdown + "`", [project, skill]).references,
    [],
  );
  assert.deepEqual(parsePromptSubmission(markdown, []).references, []);
});

test("Markdown formatting is preserved through parsing and serializing", () => {
  const markdown =
    "# 标题\n\n**粗体** *斜体* `代码` [链接](https://example.com)\n\n- 项目一\n- 项目二\n\n> 引用\n\n```ts\nconst a = 1\n```";
  const document = parsePromptDocument(markdown);
  assert.deepEqual(
    parsePromptDocument(serializePromptDocument(document)).toJSON(),
    document.toJSON(),
  );
});

test("switching project preserves other references and following text", () => {
  let state = createPromptEditorState(
    parsePromptDocument(`[Aivo](${mentionHref(project)}) 后文 @`, [project]),
  );
  state = state.apply(state.tr.setSelection(TextSelection.atEnd(state.doc)));
  const query = editorMentionQuery(state)!;
  const other = { ...project, id: "other", token: "Other" };
  state = state.apply(insertPromptMention(state, query.from, query.to, other));
  assert.deepEqual(promptDocumentReferences(state.doc), [other]);
  assert.ok(state.doc.textContent.includes("后文"));
});
