import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("desktop hides the Windows menu and localizes the macOS menu", async () => {
  const main = await readFile(
    new URL("../electron/main.cjs", import.meta.url),
    "utf8",
  );
  const menuStart = main.indexOf("function configureApplicationMenu() {");
  const menuEnd = main.indexOf("\n}\n", menuStart) + 3;
  const menuSource = main.slice(menuStart, menuEnd);

  assert.match(main, /app\.setName\('Aivo'\)/);
  assert.match(
    menuSource,
    /if \(!isMac\) \{\s*Menu\.setApplicationMenu\(null\)\s*return\s*\}/,
  );
  for (const label of [
    "Aivo",
    "关于 Aivo",
    "检查更新…",
    "服务",
    "隐藏 Aivo",
    "隐藏其他",
    "全部显示",
    "退出 Aivo",
    "文件",
    "关闭窗口",
    "编辑",
    "撤销",
    "重做",
    "剪切",
    "复制",
    "粘贴",
    "删除",
    "全选",
    "视图",
    "重新加载",
    "强制重新加载",
    "切换开发者工具",
    "实际大小",
    "放大",
    "缩小",
    "切换全屏",
    "窗口",
    "最小化",
    "缩放",
    "全部置于最前面",
  ]) {
    assert.match(menuSource, new RegExp(`label: '${label}'`));
  }
  assert.match(menuSource, /Menu\.setApplicationMenu\(menu\)/);
  assert.match(
    menuSource,
    /label: '检查更新…'[\s\S]*checkAndOfferUpdate\(window, true\)/,
  );
  assert.match(
    main,
    /state\.phase === 'up-to-date'[\s\S]*已是最新版本/,
  );
  assert.ok(
    main.indexOf("configureApplicationMenu()", main.indexOf("app.whenReady()")) <
      main.indexOf("createWindow()", main.indexOf("app.whenReady()")),
    "the localized menu should be installed before creating the first window",
  );
});
