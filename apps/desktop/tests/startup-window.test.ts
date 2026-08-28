import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("desktop shows a loading surface until the main renderer is ready", async () => {
  const [main, html, startupHtml, setupLoading, indexRoute] = await Promise.all([
    readFile(new URL("../electron/main.cjs", import.meta.url), "utf8"),
    readFile(new URL("../index.html", import.meta.url), "utf8"),
    readFile(new URL("../electron/startup.html", import.meta.url), "utf8"),
    readFile(
      new URL(
        "../src/features/setup/setup-loading-skeleton.tsx",
        import.meta.url,
      ),
      "utf8",
    ),
    readFile(new URL("../src/routes/index.tsx", import.meta.url), "utf8"),
  ]);

  const createWindowStart = main.indexOf("function createWindow(");
  const createWindowEnd = main.indexOf("\n}\n", createWindowStart) + 3;
  const createWindowSource = main.slice(createWindowStart, createWindowEnd);

  assert.match(main, /const rendererBackgroundColor = '#ffffff'/);
  assert.match(createWindowSource, /show: false/);
  assert.match(
    createWindowSource,
    /backgroundColor: rendererBackgroundColor/,
  );
  assert.match(
    createWindowSource,
    /once\('ready-to-show',[\s\S]*mainWindow\.show\(\)/,
  );
  assert.match(html, /html,[\s\S]*body,[\s\S]*#root[\s\S]*background: #ffffff/);
  assert.match(html, /class="aivo-boot"[\s\S]*加载中/);
  assert.match(html, /\.aivo-boot[\s\S]*position: fixed;[\s\S]*inset: 0;/);
  assert.match(startupHtml, /role="status"[\s\S]*加载中/);
  assert.match(startupHtml, /main[\s\S]*position: fixed;[\s\S]*inset: 0;/);
  assert.match(setupLoading, /className="fixed inset-0 grid place-items-center/);

  const readyStart = main.indexOf("app.whenReady().then(async () => {");
  const readySource = main.slice(readyStart);
  assert.ok(
    readySource.indexOf("createStartupWindow()") <
      readySource.indexOf("await startPackagedCore()"),
    "the loading window should exist while the packaged Core starts",
  );
  assert.match(
    readySource,
    /startupWindow = createStartupWindow\(\)[\s\S]*await startPackagedCore\(\)[\s\S]*createWindow\(\)/,
  );
  assert.match(
    createWindowSource,
    /mainWindow\.setBounds\(startupWindow\.getBounds\(\)\)[\s\S]*mainWindow\.show\(\)[\s\S]*startupWindow\.destroy\(\)/,
  );
  assert.doesNotMatch(indexRoute, /<Navigate/);
  assert.match(
    indexRoute,
    /useEffect\([\s\S]*navigate\(\{ to: startupRouteFor\(config\), replace: true \}\)[\s\S]*return <SetupLoadingSkeleton/,
  );
});
