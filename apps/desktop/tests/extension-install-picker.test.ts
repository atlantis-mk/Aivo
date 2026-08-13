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
  const extensionHandlerStart = main.indexOf(
    "ipcMain.handle('aivo:select-extension-directory'",
  );
  const nextHandlerStart = main.indexOf("\nipcMain.handle(", extensionHandlerStart + 1);
  const extensionHandler = main.slice(extensionHandlerStart, nextHandlerStart);

  assert.match(
    preload,
    /selectExtensionDirectory:\s*\(\)\s*=>\s*ipcRenderer\.invoke\('aivo:select-extension-directory'\)/,
  );
  assert.match(
    extensionHandler,
    /properties:\s*\['openDirectory'\]/,
  );
  assert.doesNotMatch(
    extensionHandler,
    /properties:\s*\[[^\]]*openFile/,
  );
});
