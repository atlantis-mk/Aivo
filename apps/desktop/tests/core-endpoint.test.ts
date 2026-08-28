import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";

const require = createRequire(import.meta.url);
const {
  coreURLArgument,
  normalizePackagedCoreURL,
  packagedCoreURLFromArguments,
  parseCoreReadyLine,
} = require("../electron/core-endpoint.cjs");

test("CT-RELIABILITY-001 accepts one versioned dynamic loopback endpoint", () => {
  const url = parseCoreReadyLine(
    'AIVO_CORE_READY {"version":1,"url":"http://127.0.0.1:54321"}',
  );
  assert.equal(url, "http://127.0.0.1:54321");
  const argument = coreURLArgument(url);
  assert.equal(
    packagedCoreURLFromArguments(["electron", argument]),
    "http://127.0.0.1:54321",
  );
  assert.equal(parseCoreReadyLine("ordinary Core log output"), null);
});

test("CT-SECURITY-001 refuses unsafe or ambiguous packaged Core endpoints", () => {
  for (const url of [
    "https://127.0.0.1:54321",
    "http://localhost:54321",
    "http://0.0.0.0:54321",
    "http://127.0.0.1:0",
    "http://127.0.0.1:54321/rpc",
    "http://user:pass@127.0.0.1:54321",
    "http://127.0.0.1:54321?next=remote",
  ]) {
    assert.throws(() => normalizePackagedCoreURL(url), /loopback|missing|valid/);
  }
  assert.throws(
    () => parseCoreReadyLine('AIVO_CORE_READY {"version":2,"url":"http://127.0.0.1:54321"}'),
    /unsupported/,
  );
  assert.throws(
    () => packagedCoreURLFromArguments([
      "--aivo-core-url=http://127.0.0.1:54321",
      "--aivo-core-url=http://127.0.0.1:54322",
    ]),
    /exactly once/,
  );
});

test("packaged main owns a dynamic Core and injects the accepted endpoint", async () => {
  const { readFile } = await import("node:fs/promises");
  const main = await readFile(new URL("../electron/main.cjs", import.meta.url), "utf8");
  assert.match(main, /AIVO_CORE_ADDR: '127\.0\.0\.1:0'/);
  assert.match(main, /AIVO_CORE_READY_STDOUT: '1'/);
  assert.match(main, /additionalArguments:[\s\S]*coreURLArgument\(coreUrl\)/);
  assert.doesNotMatch(main, /using existing healthy core/);
});
