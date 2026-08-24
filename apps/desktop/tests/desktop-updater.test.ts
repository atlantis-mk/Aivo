import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const {
  STABLE_MANIFEST_URL,
  artifactNameFor,
  compareStableVersions,
  createDesktopUpdater,
  validateGitHubRelease,
  validateManifest,
} = require("../electron/desktop-updater.cjs");

function fixture(version = "0.1.1", bytes = Buffer.from("verified update")) {
  const name = artifactNameFor(version, "darwin", "arm64");
  const sha256 = crypto.createHash("sha256").update(bytes).digest("hex");
  const artifact = {
    name,
    url: `https://pub-bf5092e77ab5409ba39fb34c4a76c1b1.r2.dev/aivo/releases/v${version}/${name}`,
    size: bytes.byteLength,
    sha256,
  };
  return {
    bytes,
    manifest: {
      version,
      tag: `v${version}`,
      publishedAt: "2026-08-25T00:00:00.000Z",
      releaseBaseUrl: `https://pub-bf5092e77ab5409ba39fb34c4a76c1b1.r2.dev/aivo/releases/v${version}`,
      artifacts: [artifact],
    },
    release: {
      tag_name: `v${version}`,
      draft: false,
      prerelease: false,
      assets: [{ name, size: bytes.byteLength, digest: `sha256:${sha256}` }],
    },
  };
}

test("desktop updater compares stable versions and derives only supported packages", () => {
  assert.equal(compareStableVersions("0.1.0", "0.1.1"), -1);
  assert.equal(compareStableVersions("2.0.0", "1.99.99"), 1);
  assert.equal(compareStableVersions("1.2.3", "1.2.3"), 0);
  assert.equal(artifactNameFor("1.2.3", "darwin", "arm64"), "Aivo_1.2.3_darwin-aarch64.dmg");
  assert.equal(artifactNameFor("1.2.3", "darwin", "x64"), "Aivo_1.2.3_darwin-x86_64.dmg");
  assert.equal(artifactNameFor("1.2.3", "win32", "x64"), "Aivo_1.2.3_windows-x86_64-setup.exe");
  assert.equal(artifactNameFor("1.2.3", "linux", "x64"), "Aivo_1.2.3_linux-x86_64.AppImage");
  assert.equal(artifactNameFor("1.2.3", "linux", "arm64"), null);
  assert.throws(() => compareStableVersions("1.2.3-beta.1", "1.2.3"), /无效/);
});

test("desktop updater refuses arbitrary R2 paths and cross-source digest mismatch", () => {
  const data = fixture();
  const unsafe = structuredClone(data.manifest);
  unsafe.artifacts[0].url = "https://example.com/update.dmg";
  assert.throws(() => validateManifest(unsafe, "darwin", "arm64"), /受信任/);

  const candidate = validateManifest(data.manifest, "darwin", "arm64");
  const mismatched = structuredClone(data.release);
  mismatched.assets[0].digest = `sha256:${"0".repeat(64)}`;
  assert.throws(() => validateGitHubRelease(mismatched, candidate), /不一致/);
});

test("desktop updater checks both sources, verifies bytes, and opens only the owned package", async (t) => {
  const data = fixture();
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "aivo-updater-test-"));
  t.after(() => fs.promises.rm(tempRoot, { recursive: true, force: true }));
  const opened: string[] = [];
  const requested: string[] = [];
  const updater = createDesktopUpdater({
    appVersion: "0.1.0",
    platform: "darwin",
    arch: "arm64",
    isPackaged: true,
    tempRoot,
    shell: {
      openPath: async (target: string) => {
        opened.push(target);
        return "";
      },
      showItemInFolder: () => {},
    },
    fetchImpl: async (url: string) => {
      requested.push(url);
      if (url === STABLE_MANIFEST_URL) return Response.json(data.manifest);
      if (url.includes("api.github.com")) return Response.json(data.release);
      if (url === data.manifest.artifacts[0].url) {
        return new Response(data.bytes, { headers: { "content-length": String(data.bytes.byteLength) } });
      }
      return new Response(null, { status: 404 });
    },
  });

  assert.equal((await updater.check()).phase, "available");
  assert.equal((await updater.download()).phase, "ready");
  assert.equal(opened.length, 0);
  assert.equal((await updater.install()).phase, "ready");
  assert.equal(opened.length, 1);
  assert.equal(path.basename(opened[0]), data.manifest.artifacts[0].name);
  assert.deepEqual(requested.slice(0, 2), [
    STABLE_MANIFEST_URL,
    "https://api.github.com/repos/atlantis-mk/Aivo/releases/tags/v0.1.1",
  ]);
});

test("desktop updater removes partial bytes when a download is cancelled", async (t) => {
  const data = fixture("0.1.1", Buffer.alloc(32, 7));
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "aivo-updater-cancel-"));
  t.after(() => fs.promises.rm(tempRoot, { recursive: true, force: true }));
  let downloadStarted!: () => void;
  const started = new Promise<void>((resolve) => { downloadStarted = resolve; });
  const updater = createDesktopUpdater({
    appVersion: "0.1.0",
    platform: "darwin",
    arch: "arm64",
    isPackaged: true,
    tempRoot,
    shell: { openPath: async () => "", showItemInFolder: () => {} },
    fetchImpl: async (url: string, options: { signal?: AbortSignal } = {}) => {
      if (url === STABLE_MANIFEST_URL) return Response.json(data.manifest);
      if (url.includes("api.github.com")) return Response.json(data.release);
      return new Response(new ReadableStream({
        start(controller) {
          controller.enqueue(data.bytes.subarray(0, 4));
          downloadStarted();
          options.signal?.addEventListener("abort", () => {
            controller.error(new DOMException("Cancelled", "AbortError"));
          }, { once: true });
        },
      }), { headers: { "content-length": String(data.bytes.byteLength) } });
    },
  });

  await updater.check();
  const downloading = updater.download();
  await started;
  updater.cancel();
  assert.equal((await downloading).phase, "available");
  const updateRoot = path.join(tempRoot, "aivo-updates");
  const files = fs.existsSync(updateRoot)
    ? (await fs.promises.readdir(updateRoot, { recursive: true })).filter((entry) => String(entry).endsWith(".part"))
    : [];
  assert.deepEqual(files, []);
});

test("preload exposes fixed updater capabilities without renderer-selected arguments", async () => {
  const preload = await fs.promises.readFile(
    new URL("../electron/preload.cjs", import.meta.url),
    "utf8",
  );
  assert.match(preload, /updates: Object\.freeze\(\{/);
  for (const operation of ["getState", "check", "download", "install", "cancel"]) {
    assert.match(preload, new RegExp(`${operation}: \\(\\) => ipcRenderer\\.invoke\\('aivo:update:`));
  }
  assert.doesNotMatch(preload, /aivo:update:(?:check|download|install)'\s*,\s*(?:target|url|path|input)/);
});
