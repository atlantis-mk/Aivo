import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("plus and mention actions use the same local resource callback", async () => {
  const toolbar = await readFile(
    new URL(
      "../src/features/projects/project-prompt-composer-toolbar.tsx",
      import.meta.url,
    ),
    "utf8",
  );
  const textarea = await readFile(
    new URL(
      "../src/features/projects/project-prompt-composer-textarea.tsx",
      import.meta.url,
    ),
    "utf8",
  );

  assert.match(toolbar, /onClick=\{onSelectLocalResource\}/);
  assert.match(textarea, /await onSelectLocalResource\(\)/);
});

test("dropped files are resolved in preload and inspected by Electron main", async () => {
  const preload = await readFile(
    new URL("../electron/preload.cjs", import.meta.url),
    "utf8",
  );
  const main = await readFile(
    new URL("../electron/main.cjs", import.meta.url),
    "utf8",
  );
  const attachmentState = await readFile(
    new URL(
      "../src/features/projects/project-composer-attachment-state.ts",
      import.meta.url,
    ),
    "utf8",
  );

  assert.match(preload, /webUtils\.getPathForFile/);
  assert.match(preload, /aivo:inspect-dropped-composer-resources/);
  assert.match(main, /ipcMain\.handle\('aivo:inspect-dropped-composer-resources'/);
  assert.match(attachmentState, /inspectDroppedComposerResources\(files\)/);
  assert.match(attachmentState, /routeComposerLocalSelections/);
});
