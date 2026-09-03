import { createHash } from "node:crypto";
import { access, mkdir, readFile, rename, writeFile } from "node:fs/promises";
import { arch, platform } from "node:os";
import { basename, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const workspaceRoot = resolve(scriptDirectory, "../../..");
const cargoManifest = resolve(workspaceRoot, "codex-rs/Cargo.toml");
const release = process.argv.includes("--release");
const fastRelease = process.argv.includes("--fast-release");
const hostOnly = process.argv.includes("--host-only");
const debugCodexPath = resolve(workspaceRoot, "codex-rs/target/debug/codex");
const buildCodex = !hostOnly || release || !(await exists(debugCodexPath));
const v8Version = await resolvedV8Version();
const target = process.env.AIVO_RUST_TARGET ?? nativeRustTarget();
const profile = "ptrcomp_sandbox_release";
const releaseUrl = `https://github.com/openai/codex/releases/download/rusty-v8-v${v8Version}`;
const artifactDirectory = resolve(
  workspaceRoot,
  "codex-rs/target/aivo-rusty-v8",
  v8Version,
  target,
);
const archiveName = archiveFileName(profile, target);
const bindingName = `src_binding_${profile}_${target}.rs`;
const checksumsName = `rusty_v8_${profile}_${target}.sha256`;

await mkdir(artifactDirectory, { recursive: true });
const [archivePath, bindingPath, checksumsPath] = await Promise.all(
  [archiveName, bindingName, checksumsName].map((name) =>
    downloadArtifact(name, resolve(artifactDirectory, name)),
  ),
);
await verifyArtifacts(checksumsPath, [archivePath, bindingPath]);

const cargoArguments = [
  "build",
  ...(release ? ["--release"] : []),
  "--locked",
  "--manifest-path",
  cargoManifest,
  ...(buildCodex ? ["-p", "codex-cli", "--bin", "codex"] : []),
  "-p",
  "codex-code-mode-host",
  "--bin",
  "codex-code-mode-host",
];
const cargo = spawn("cargo", cargoArguments, {
  env: {
    ...process.env,
    ...(fastRelease
      ? {
          CARGO_PROFILE_RELEASE_CODEGEN_UNITS: "16",
          CARGO_PROFILE_RELEASE_LTO: "false",
        }
      : {}),
    RUSTY_V8_ARCHIVE: archivePath,
    RUSTY_V8_SRC_BINDING_PATH: bindingPath,
  },
  stdio: "inherit",
});

cargo.once("error", (error) => {
  console.error(`Could not start Cargo: ${error.message}`);
  process.exitCode = 1;
});
cargo.once("exit", (code, signal) => {
  if (signal) {
    console.error(`Cargo stopped by signal ${signal}.`);
    process.exitCode = 1;
    return;
  }
  process.exitCode = code ?? 1;
});

async function resolvedV8Version() {
  const cargoToml = await readFile(cargoManifest, "utf8");
  const match = cargoToml.match(/^v8\s*=\s*"=([^"]+)"$/m);
  if (!match) throw new Error("Could not resolve the pinned v8 crate version.");
  return match[1];
}

async function exists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

function nativeRustTarget() {
  const operatingSystem = platform();
  const architecture = arch();
  const targets = {
    "darwin:arm64": "aarch64-apple-darwin",
    "darwin:x64": "x86_64-apple-darwin",
    "linux:arm64": "aarch64-unknown-linux-gnu",
    "linux:x64": "x86_64-unknown-linux-gnu",
    "win32:arm64": "aarch64-pc-windows-msvc",
    "win32:x64": "x86_64-pc-windows-msvc",
  };
  const target = targets[`${operatingSystem}:${architecture}`];
  if (!target) {
    throw new Error(
      `No Codex V8 artifact is configured for ${operatingSystem}/${architecture}. Set AIVO_RUST_TARGET explicitly.`,
    );
  }
  return target;
}

function archiveFileName(currentProfile, currentTarget) {
  if (currentTarget.endsWith("-pc-windows-msvc")) {
    return `rusty_v8_${currentProfile}_${currentTarget}.lib.gz`;
  }
  return `librusty_v8_${currentProfile}_${currentTarget}.a.gz`;
}

async function downloadArtifact(name, destination) {
  try {
    await readFile(destination);
    return destination;
  } catch {
    const response = await fetch(`${releaseUrl}/${name}`);
    if (!response.ok) {
      throw new Error(
        `Could not download ${name}: ${response.status} ${response.statusText}`,
      );
    }
    const content = Buffer.from(await response.arrayBuffer());
    const temporaryDestination = `${destination}.tmp`;
    await writeFile(temporaryDestination, content);
    await rename(temporaryDestination, destination);
    return destination;
  }
}

async function verifyArtifacts(checksumsPath, paths) {
  const checksums = new Map(
    (await readFile(checksumsPath, "utf8"))
      .split(/\r?\n/)
      .filter(Boolean)
      .map((line) => {
        const [digest, name] = line.trim().split(/\s+/, 2);
        return [name, digest];
      }),
  );
  for (const path of paths) {
    const name = basename(path);
    const expectedDigest = name ? checksums.get(name) : undefined;
    if (!expectedDigest) throw new Error(`Checksum missing for ${path}.`);
    const actualDigest = createHash("sha256")
      .update(await readFile(path))
      .digest("hex");
    if (actualDigest !== expectedDigest) {
      throw new Error(`Checksum mismatch for ${path}.`);
    }
  }
}
