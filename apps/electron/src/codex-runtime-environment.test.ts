import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, test } from "node:test";

import {
  buildAivoRuntimeEnvironment,
  ensureAivoCodexHome,
} from "./codex-runtime-environment.ts";

const temporaryDirectories: string[] = [];

function temporaryHome(): string {
  const directory = mkdtempSync(path.join(os.tmpdir(), "aivo-runtime-home-"));
  temporaryDirectories.push(directory);
  return directory;
}

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) {
    rmSync(directory, { force: true, recursive: true });
  }
});

test("creates the Aivo runtime home without changing existing Aivo data", () => {
  const home = temporaryHome();
  const aivoHome = path.join(home, ".aivo");
  const skills = path.join(aivoHome, "skills");
  mkdirSync(skills, { recursive: true });
  writeFileSync(path.join(aivoHome, "aivo.db"), "existing database");

  assert.equal(ensureAivoCodexHome(home), aivoHome);
  assert.equal(readFileSync(path.join(aivoHome, "aivo.db"), "utf8"), "existing database");
});

test("overrides inherited Codex storage paths only for the runtime environment", () => {
  const home = temporaryHome();
  const inheritedEnvironment = {
    CODEX_HOME: "/shared/codex-home",
    CODEX_SQLITE_HOME: "/shared/codex-sqlite",
    KEEP_ME: "kept",
  };

  const environment = buildAivoRuntimeEnvironment({
    homeDirectory: home,
    inheritedEnvironment,
    injectedEnvironment: { AIVO_PROVIDER_TEST_API_KEY: "secret" },
  });

  assert.equal(environment.CODEX_HOME, path.join(home, ".aivo"));
  assert.equal(environment.CODEX_SQLITE_HOME, path.join(home, ".aivo"));
  assert.equal(environment.KEEP_ME, "kept");
  assert.equal(environment.AIVO_PROVIDER_TEST_API_KEY, "secret");
  assert.equal(inheritedEnvironment.CODEX_HOME, "/shared/codex-home");
  assert.equal(inheritedEnvironment.CODEX_SQLITE_HOME, "/shared/codex-sqlite");
});

test("rejects an Aivo runtime home path occupied by a file", () => {
  const home = temporaryHome();
  writeFileSync(path.join(home, ".aivo"), "not a directory");

  assert.throws(
    () => ensureAivoCodexHome(home),
    /Aivo runtime home is unavailable.*\.aivo/,
  );
});
