import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { hasRenderableMarkdown } from "./project-prompt-markdown.ts";

describe("prompt Markdown detection", () => {
  it("recognizes complete block and inline Markdown", () => {
    assert.equal(hasRenderableMarkdown("# 发布说明\n\n内容"), true);
    assert.equal(hasRenderableMarkdown("- 第一项\n- 第二项"), true);
    assert.equal(hasRenderableMarkdown("这是 **重点** 内容"), true);
    assert.equal(
      hasRenderableMarkdown("查看 [文档](https://example.com)"),
      true,
    );
  });

  it("keeps ordinary text in the editor", () => {
    assert.equal(hasRenderableMarkdown("请帮我整理这份说明"), false);
    assert.equal(hasRenderableMarkdown("尚未完成的 **格式"), false);
  });
});
