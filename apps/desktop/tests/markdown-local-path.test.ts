import assert from "node:assert/strict";
import test from "node:test";

import {
  localPathFromMarkdownHref,
  localPathFromText,
  markdownHrefForLocalPath,
} from "../src/components/markdown-local-path.ts";

test("recognizes host-native absolute file and folder paths", () => {
  assert.equal(
    localPathFromText("/Users/aivo/Documents/AI 教程", "darwin"),
    "/Users/aivo/Documents/AI 教程",
  );
  assert.equal(
    localPathFromText("C:\\Users\\aivo\\notes.md", "win32"),
    "C:\\Users\\aivo\\notes.md",
  );
  assert.equal(
    localPathFromText("\\\\server\\share\\notes.md", "win32"),
    "\\\\server\\share\\notes.md",
  );
});

test("removes supported source locations before opening", () => {
  assert.equal(
    localPathFromText("/workspace/src/main.ts:42:7", "linux"),
    "/workspace/src/main.ts",
  );
  assert.equal(
    localPathFromText("/workspace/src/main.ts#L42C7", "linux"),
    "/workspace/src/main.ts",
  );
});

test("rejects relative and foreign-platform paths", () => {
  assert.equal(localPathFromText("src/main.ts", "linux"), undefined);
  assert.equal(localPathFromText("C:\\src\\main.ts", "darwin"), undefined);
  assert.equal(localPathFromText("/src/main.ts", "win32"), undefined);
  assert.equal(localPathFromText("/src/main.ts\nnext", "linux"), undefined);
});

test("round-trips absolute Markdown link targets through an inert fragment", () => {
  const href = markdownHrefForLocalPath(
    "/Users/aivo/Documents/AI%20%E6%95%99%E7%A8%8B/readme.md:12",
    "darwin",
  );

  assert.ok(href?.startsWith("#aivo-local-path="));
  assert.equal(
    localPathFromMarkdownHref(href!),
    "/Users/aivo/Documents/AI 教程/readme.md",
  );
});
