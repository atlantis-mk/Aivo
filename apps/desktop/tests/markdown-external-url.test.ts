import assert from "node:assert/strict";
import test from "node:test";

import { isExternalMarkdownUrl } from "../src/components/markdown-external-url.ts";

test("permits only absolute web and mail Markdown links", () => {
  for (const value of [
    "https://example.com/source",
    "http://127.0.0.1:43117/health",
    "mailto:hello@example.com",
  ]) {
    assert.equal(isExternalMarkdownUrl(value), true);
  }
});

test("refuses unsafe, relative, and malformed Markdown links", () => {
  for (const value of [
    "javascript:alert(1)",
    "file:///Users/aivo/private.txt",
    "../relative-path",
    "#fragment-only",
    "not a URL",
  ]) {
    assert.equal(isExternalMarkdownUrl(value), false);
  }
});
