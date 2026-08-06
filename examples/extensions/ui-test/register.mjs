#!/usr/bin/env node

import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const extensionRoot = dirname(fileURLToPath(import.meta.url));
const coreURL = readArgument("--core-url") || "http://127.0.0.1:43117";

const discovered = await callCore("DiscoverExtension", { path: extensionRoot });
let status = discovered;

if (!status.trusted) {
  status = await callCore("TrustExtension", {
    id: status.id,
    integrity: status.integrity,
  });
}

if (!status.enabled || (status.state !== "ready" && status.state !== "active")) {
  status = await callCore("EnableExtension", { id: status.id });
}

process.stdout.write(
  `${JSON.stringify(
    {
      extensionRoot: resolve(extensionRoot),
      id: status.id,
      integrity: status.integrity,
      state: status.state,
      trusted: status.trusted,
      enabled: status.enabled,
    },
    null,
    2,
  )}\n`,
);

async function callCore(method, input) {
  const response = await fetch(`${coreURL}/api/rpc/${encodeURIComponent(method)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ args: [input] }),
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(payload?.error || `${method} failed with HTTP ${response.status}`);
  }
  return payload;
}

function readArgument(name) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : "";
}
