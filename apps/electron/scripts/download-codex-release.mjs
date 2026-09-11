import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import {
  chmod,
  mkdir,
  readFile,
  readdir,
  rename,
  rm,
  stat,
  writeFile,
} from "node:fs/promises";
import { arch, platform } from "node:os";
import { basename, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const appDirectory = resolve(scriptDirectory, "..");
const packageRuntime = process.argv.includes("--package");
const runtimeDirectory = resolve(
  appDirectory,
  ".aivo-runtime",
  ...(packageRuntime ? ["package"] : []),
);
const releaseApiUrl = "https://api.github.com/repos/openai/codex/releases/latest";
const target = nativeReleaseTarget();
const components = packageRuntime ? ["codex", "codex-code-mode-host"] : ["codex"];

const release = await latestStableRelease();
const assets = components.map((component) => releaseAsset(release, component));
if (await isCurrentRelease(release, assets)) {
  console.log(`Using OpenAI Codex ${release.tag_name} (${target}) from cache.`);
  process.exit(0);
}

await mkdir(runtimeDirectory, { recursive: true });
const temporaryDirectory = resolve(
  runtimeDirectory,
  `.download-${process.pid}-${Date.now()}`,
);

try {
  const installed = [];
  for (const asset of assets) {
    installed.push(await downloadAndExtract(asset));
  }

  for (const { binaryPath, component } of installed) {
    const destination = resolve(runtimeDirectory, installedBinaryName(component));
    await rm(destination, { force: true });
    await rename(binaryPath, destination);
  }
  await writeFile(
    resolve(runtimeDirectory, "codex-release.json"),
    `${JSON.stringify(
      {
        tag: release.tag_name,
        assets: assets.map(({ name, checksum }) => ({ name, sha256: checksum })),
        downloadedAt: new Date().toISOString(),
      },
      null,
      2,
    )}\n`,
  );
  console.log(
    `Installed OpenAI Codex ${release.tag_name} (${target}) for ${packageRuntime ? "packaging" : "development"}.`,
  );
} finally {
  await rm(temporaryDirectory, { recursive: true, force: true });
}

async function latestStableRelease() {
  const response = await fetch(releaseApiUrl, {
    headers: {
      Accept: "application/vnd.github+json",
      "User-Agent": "aivo-desktop-runtime",
    },
  });
  if (!response.ok) {
    throw new Error(
      `Could not find the latest OpenAI Codex release: ${response.status} ${response.statusText}`,
    );
  }
  const release = await response.json();
  if (
    !release ||
    typeof release.tag_name !== "string" ||
    release.prerelease ||
    release.draft ||
    !Array.isArray(release.assets)
  ) {
    throw new Error("OpenAI Codex latest release was not a published stable release.");
  }
  return release;
}

function releaseAsset(release, component) {
  const archiveName = `${component}-${target}${executableExtension()}.tar.gz`;
  const asset = release.assets.find((candidate) => candidate.name === archiveName);
  if (!asset) {
    throw new Error(
      `OpenAI Codex ${release.tag_name} has no ${archiveName} asset for ${platform()}/${arch()}.`,
    );
  }
  return { ...asset, checksum: checksumFor(asset), component };
}

async function isCurrentRelease(release, assets) {
  try {
    const [manifestText, ...binaries] = await Promise.all([
      readFile(resolve(runtimeDirectory, "codex-release.json"), "utf8"),
      ...components.map((component) =>
        stat(resolve(runtimeDirectory, installedBinaryName(component))),
      ),
    ]);
    const manifest = JSON.parse(manifestText);
    return (
      manifest.tag === release.tag_name &&
      binaries.every((binary) => binary.isFile()) &&
      sameAssets(manifest.assets, assets)
    );
  } catch {
    return false;
  }
}

function sameAssets(saved, assets) {
  if (!Array.isArray(saved) || saved.length !== assets.length) return false;
  return assets.every((asset) =>
    saved.some(
      (entry) => entry?.name === asset.name && entry?.sha256 === asset.checksum,
    ),
  );
}

async function downloadAndExtract(asset) {
  const componentDirectory = resolve(temporaryDirectory, asset.component);
  const archivePath = resolve(componentDirectory, asset.name);
  const extractedDirectory = resolve(componentDirectory, "extracted");
  await mkdir(extractedDirectory, { recursive: true });

  console.log(`Downloading ${asset.name}…`);
  const response = await fetch(asset.browser_download_url, {
    headers: {
      Accept: "application/octet-stream",
      "User-Agent": "aivo-desktop-runtime",
    },
  });
  if (!response.ok) {
    throw new Error(
      `Could not download ${asset.name}: ${response.status} ${response.statusText}`,
    );
  }

  const archive = Buffer.from(await response.arrayBuffer());
  const actualChecksum = createHash("sha256").update(archive).digest("hex");
  if (actualChecksum !== asset.checksum) {
    throw new Error(
      `Checksum mismatch for ${asset.name}: expected ${asset.checksum}, got ${actualChecksum}.`,
    );
  }
  await writeFile(archivePath, archive);
  await extractTarball(archivePath, extractedDirectory);

  const archiveBinaryName = `${asset.component}-${target}${executableExtension()}`;
  const binaryPath = resolve(componentDirectory, installedBinaryName(asset.component));
  await rename(await findBinary(extractedDirectory, archiveBinaryName), binaryPath);
  await chmod(binaryPath, 0o755);
  return { binaryPath, component: asset.component };
}

function nativeReleaseTarget() {
  const targets = {
    "darwin:arm64": "aarch64-apple-darwin",
    "darwin:x64": "x86_64-apple-darwin",
    "linux:arm64": "aarch64-unknown-linux-musl",
    "linux:x64": "x86_64-unknown-linux-musl",
    "win32:arm64": "aarch64-pc-windows-msvc",
    "win32:x64": "x86_64-pc-windows-msvc",
  };
  const resolvedTarget = targets[`${platform()}:${arch()}`];
  if (!resolvedTarget) {
    throw new Error(
      `No stable Codex release asset is configured for ${platform()}/${arch()}.`,
    );
  }
  return resolvedTarget;
}

function executableExtension() {
  return platform() === "win32" ? ".exe" : "";
}

function installedBinaryName(component) {
  return `${component}${executableExtension()}`;
}

function checksumFor(asset) {
  if (typeof asset.digest !== "string" || !asset.digest.startsWith("sha256:")) {
    throw new Error(`OpenAI did not publish a SHA-256 digest for ${asset.name}.`);
  }
  return asset.digest.slice("sha256:".length);
}

function extractTarball(archive, destination) {
  return new Promise((resolvePromise, reject) => {
    const tar = spawn("tar", ["-xzf", tarPath(archive), "-C", tarPath(destination)], {
      stdio: "inherit",
    });
    tar.once("error", (error) => {
      reject(new Error(`Could not extract ${basename(archive)}: ${error.message}`));
    });
    tar.once("exit", (code, signal) => {
      if (code === 0) {
        resolvePromise();
      } else {
        reject(
          new Error(
            `Could not extract ${basename(archive)}${signal ? ` (${signal})` : ""}.`,
          ),
        );
      }
    });
  });
}

function tarPath(filePath) {
  if (platform() !== "win32") return filePath;
  return filePath
    .replace(/^([A-Za-z]):/, (_, drive) => `/${drive.toLowerCase()}`)
    .replace(/\\/g, "/");
}

async function findBinary(directory, expectedName) {
  const candidates = [];
  await visit(directory);
  if (candidates.length !== 1) {
    throw new Error(
      `Expected exactly one ${expectedName} executable in the Codex archive, found ${candidates.length}.`,
    );
  }
  return candidates[0];

  async function visit(path) {
    for (const entry of await readdir(path, { withFileTypes: true })) {
      const entryPath = resolve(path, entry.name);
      if (entry.isDirectory()) {
        await visit(entryPath);
      } else if (entry.isFile() && entry.name === expectedName) {
        candidates.push(entryPath);
      }
    }
  }
}
