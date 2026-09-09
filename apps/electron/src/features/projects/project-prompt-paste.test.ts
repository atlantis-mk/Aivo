import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  createPromptPaste,
  expandPromptPastes,
  isLargePromptPaste,
  promptPasteTextElements,
  promptWithPasteSummaries,
  promptPasteTarget,
  inferPromptPasteName,
} from "./project-prompt-paste.ts";

const largeText = "x".repeat(1001);

describe("prompt paste handling", () => {
  it("recognizes large pastes after line normalization", () => {
    assert.equal(isLargePromptPaste("x".repeat(1000)), false);
    assert.equal(isLargePromptPaste(largeText), true);
    assert.equal(isLargePromptPaste("a\r\n".repeat(501)), true);
  });

  it("creates a pending paste payload for large text", () => {
    const result = createPromptPaste({ pastedText: largeText });

    assert.equal(result?.paste.text, largeText);
    assert.equal(result?.paste.id.length, 36);
  });

  it("creates an openable pending paste from a clipboard file path", () => {
    const sourcePath = "/tmp/方舟 Agent Plan.md";
    const result = createPromptPaste({ pastedText: "", sourcePath });

    assert.deepEqual(result?.paste, {
      id: result?.paste.id,
      name: "方舟 Agent Plan.md",
      text: sourcePath,
      target: sourcePath,
    });
  });

  it("uses a clipboard file name while preserving pasted content", () => {
    const result = createPromptPaste({
      pastedText: largeText,
      sourcePath: "/tmp/方舟 Agent Plan.md",
    });

    assert.equal(result?.paste.name, "方舟 Agent Plan.md");
    assert.equal(result?.paste.text, largeText);
    assert.equal(result?.paste.target, "/tmp/方舟 Agent Plan.md");
  });

  it("infers a markdown file title from pasted content", () => {
    const result = createPromptPaste({
      pastedText: `# 方舟 Agent Plan\n\n${largeText}`,
    });

    assert.equal(result?.paste.name, "方舟 Agent Plan.md");
    assert.equal(inferPromptPasteName("---\ntitle: 方舟 Agent Plan.md\n---\nbody"), "方舟 Agent Plan.md");
    assert.equal(inferPromptPasteName(largeText), undefined);
  });

  it("summarizes pending pastes for display while expanding for submission", () => {
    const pendingPastes = [
      { id: "first", text: largeText },
      { id: "second", text: "second" },
    ];

    assert.equal(
      promptWithPasteSummaries("before", pendingPastes),
      "before\n\n[粘贴内容（1001 字符）]\n\n[粘贴内容（6 字符）]",
    );
    assert.equal(
      expandPromptPastes("before", pendingPastes),
      `before\n\n${largeText}\n\nsecond`,
    );
  });

  it("expands pastes without typed prompt text", () => {
    assert.equal(
      expandPromptPastes("", [{ id: "paste", text: largeText }]),
      largeText,
    );
  });

  it("marks pasted content with UTF-8 ranges for durable compact rendering", () => {
    assert.deepEqual(
      promptPasteTextElements("请处理：", [
        { id: "paste", text: "第一行\n第二行", name: "字幕.srt" },
      ]),
      [{
        byteRange: { start: 14, end: 33 },
        placeholder: "字幕.srt",
      }],
    );
  });

  it("detects openable path and URL targets", () => {
    assert.equal(promptPasteTarget("/tmp/plan.md"), "/tmp/plan.md");
    assert.equal(promptPasteTarget("/tmp/方舟 Agent Plan.md"), "/tmp/方舟 Agent Plan.md");
    assert.equal(promptPasteTarget('"/tmp/方舟 Agent Plan.md"'), "/tmp/方舟 Agent Plan.md");
    assert.equal(promptPasteTarget("https://example.com"), "https://example.com");
    assert.equal(promptPasteTarget("line one\nline two"), undefined);
    assert.equal(promptPasteTarget("plain text"), undefined);
  });
});
