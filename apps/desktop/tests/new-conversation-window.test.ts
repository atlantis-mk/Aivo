import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("new-window conversation opens an independent chat renderer", async () => {
  const [main, preload, topBar] = await Promise.all([
    readFile(new URL("../electron/main.cjs", import.meta.url), "utf8"),
    readFile(new URL("../electron/preload.cjs", import.meta.url), "utf8"),
    readFile(
      new URL(
        "../src/features/projects/project-workspace-top-bars.tsx",
        import.meta.url,
      ),
      "utf8",
    ),
  ]);

  assert.match(
    main,
    /const newConversationWindowOffset = 28/,
  );
  assert.match(
    main,
    /ipcMain\.handle\('aivo:new-conversation-window',[\s\S]*BrowserWindow\.fromWebContents\(event\.sender\)[\s\S]*x: sourcePosition\[0\] \+ newConversationWindowOffset,[\s\S]*y: sourcePosition\[1\] \+ newConversationWindowOffset,[\s\S]*createWindow/,
  );
  assert.match(
    main,
    /function createWindow\(initialRoute = '', position\)[\s\S]*x: position\.x, y: position\.y[\s\S]*loadURL\(rendererUrl\.toString\(\)\)[\s\S]*hash: initialRoute/,
  );
  assert.match(
    preload,
    /openNewConversationWindow:\s*\(\)\s*=>\s*ipcRenderer\.invoke\('aivo:new-conversation-window'\)/,
  );
  assert.match(topBar, /onClick=\{onNewConversationWindow\}/);
  assert.match(topBar, />新窗口对话<\/span>/);

  const newConversationIndex = topBar.indexOf(">新对话</span>");
  const newWindowIndex = topBar.indexOf(">新窗口对话</span>");
  assert.ok(newConversationIndex >= 0);
  assert.ok(newWindowIndex > newConversationIndex);
});
