import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("desktop exposes a native directory-only picker for extension installation", async () => {
  const preload = await readFile(
    new URL("../electron/preload.cjs", import.meta.url),
    "utf8",
  );
  const main = await readFile(
    new URL("../electron/main.cjs", import.meta.url),
    "utf8",
  );

  assert.match(
    preload,
    /selectExtensionDirectory:\s*\(\)\s*=>\s*ipcRenderer\.invoke\('aivo:select-extension-directory'\)/,
  );
  assert.match(
    main,
    /ipcMain\.handle\('aivo:select-extension-directory',[\s\S]*?properties:\s*\['openDirectory'\]/,
  );
  assert.doesNotMatch(
    main,
    /ipcMain\.handle\('aivo:select-extension-directory',[\s\S]*?properties:\s*\[[^\]]*openFile/,
  );
});
