import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { readFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { createInterface } from "node:readline";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  createExtensionServer,
  TOOL_NAME,
  TOOL_SCHEMA_HASH,
} from "./server.mjs";

const extensionRoot = dirname(fileURLToPath(import.meta.url));
const bearerToken = "test-bearer-token";

test("schema hash matches the declared manifest schema", async () => {
  const manifest = JSON.parse(
    await readFile(join(extensionRoot, ".aivo-extension", "extension.json"), "utf8"),
  );
  const declaration = manifest.contributes.tools.find((tool) => tool.name === TOOL_NAME);
  assert.ok(declaration);
  assert.equal(stableJSONStringify(declaration.schema), stableJSONStringify({
    additionalProperties: false,
    properties: {
      message: {
        description: "Message to display in the test panel.",
        type: "string",
      },
    },
    required: ["message"],
    type: "object",
  }));
  assert.match(TOOL_SCHEMA_HASH, /^[a-f0-9]{64}$/);
});

test("service authenticates requests and completes the tool/view flow", async (t) => {
  const server = createExtensionServer({ bearerToken });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  t.after(() => server.close());

  const address = server.address();
  const baseURL = `http://127.0.0.1:${address.port}`;
  const unauthorized = await fetch(`${baseURL}/ui`);
  assert.equal(unauthorized.status, 401);

  const initialized = await rpc(baseURL, "1", "extension/initialize", {
    apiVersion: "2",
    extensionId: "dev.aivo.ui-test",
    extensionVersion: "0.3.0",
  });
  assert.equal(initialized.result.apiVersion, "2");

  const catalog = await rpc(baseURL, "2", "catalog/list", {});
  assert.deepEqual(catalog.result.tools, [
    { name: TOOL_NAME, schemaHash: TOOL_SCHEMA_HASH },
  ]);

  const executed = await rpc(baseURL, "3", "tool/execute", {
    name: TOOL_NAME,
    arguments: { message: "Hello from the test" },
    operationId: "operation-1",
    sessionId: "session-1",
    turnId: "turn-1",
  });
  assert.equal(executed.result.ok, true);

  const stateResponse = await fetch(
    `${baseURL}/ui/state?operationId=operation-1`,
    { headers: authorizationHeaders() },
  );
  const state = await stateResponse.json();
  assert.equal(state.state.message, "Hello from the test");

  const action = await rpc(baseURL, "4", "ui/event", {
    viewId: "ui-test-detail",
    action: "test.record",
    data: { operationId: "operation-1", label: "Clicked" },
  });
  assert.equal(action.result.state.actionCount, 1);
  assert.equal(action.result.state.lastAction.label, "Clicked");

  const page = await fetch(`${baseURL}/ui`, { headers: authorizationHeaders() });
  assert.equal(page.status, 200);
  assert.match(await page.text(), /界面扩展测试台/);
});

test("runtime messaging provides bounded request and streamed Port flow", async (t) => {
  const server = createExtensionServer({ bearerToken });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  t.after(() => server.close());
  const address = server.address();
  const baseURL = `http://127.0.0.1:${address.port}`;

  const oneShot = await fetch(`${baseURL}/.well-known/aivo-runtime/messages`, {
    method: "POST",
    headers: { ...authorizationHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ message: { type: "ping" } }),
  });
  assert.equal(oneShot.status, 200);
  assert.deepEqual((await oneShot.json()).message.received, { type: "ping" });

  const portId = "port-test-1";
  const streamResponse = await fetch(
    `${baseURL}/.well-known/aivo-runtime/ports/${portId}`,
    {
      headers: {
        ...authorizationHeaders(),
        Accept: "application/x-ndjson",
        "X-Aivo-Port-Name": "state-stream",
        "X-Aivo-View-ID": "ui-test-detail",
      },
    },
  );
  assert.equal(streamResponse.status, 200);
  const reader = streamResponse.body.getReader();
  t.after(() => reader.cancel());
  assert.match(await readStreamLine(reader), /"type":"connected"/);

  await rpc(baseURL, "stream-tool", "tool/execute", {
    name: TOOL_NAME,
    arguments: { message: "Streamed state" },
    operationId: "stream-operation",
    sessionId: "session-1",
    turnId: "turn-1",
  });
  assert.match(await readStreamLine(reader), /"operationId":"stream-operation"/);

  const posted = await fetch(
    `${baseURL}/.well-known/aivo-runtime/ports/${portId}/messages`,
    {
      method: "POST",
      headers: { ...authorizationHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ message: { type: "select-operation", operationId: "stream-operation" } }),
    },
  );
  assert.equal(posted.status, 202);
  assert.match(await readStreamLine(reader), /"type":"tool-state"/);

  const closed = await fetch(
    `${baseURL}/.well-known/aivo-runtime/ports/${portId}`,
    { method: "DELETE", headers: authorizationHeaders() },
  );
  assert.equal(closed.status, 200);
});

test("executable binds an operating-system-assigned port and reports readiness", async (t) => {
  const child = spawn(process.execPath, [join(extensionRoot, "server.mjs")], {
    cwd: extensionRoot,
    env: { ...process.env, AIVO_EXTENSION_BEARER_TOKEN: bearerToken },
    stdio: ["ignore", "pipe", "pipe"],
  });
  t.after(() => child.kill("SIGTERM"));
  const lines = createInterface({ input: child.stdout });
  const [line] = await once(lines, "line");
  const readiness = JSON.parse(line);
  assert.equal(readiness.protocol, "aivo-extension-service/1");
  const endpoint = new URL(readiness.url);
  assert.equal(endpoint.hostname, "127.0.0.1");
  assert.ok(Number(endpoint.port) > 0);

  const initialized = await rpc(readiness.url, "dynamic-1", "extension/initialize", {
    apiVersion: "2",
    extensionId: "dev.aivo.ui-test",
    extensionVersion: "0.3.0",
  });
  assert.equal(initialized.result.apiVersion, "2");
});

async function readStreamLine(reader) {
  const decoder = new TextDecoder();
  let pending = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) throw new Error("stream ended before a line was received");
    pending += decoder.decode(value, { stream: true });
    const newline = pending.indexOf("\n");
    if (newline >= 0) return pending.slice(0, newline);
  }
}

async function rpc(baseURL, id, method, params) {
  const response = await fetch(baseURL, {
    method: "POST",
    headers: { ...authorizationHeaders(), "Content-Type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id, method, params }),
  });
  assert.equal(response.status, 200);
  const payload = await response.json();
  assert.equal(payload.id, id);
  assert.equal(payload.jsonrpc, "2.0");
  assert.equal(payload.error, undefined);
  return payload;
}

function authorizationHeaders() {
  return { Authorization: `Bearer ${bearerToken}` };
}

function stableJSONStringify(value) {
  if (Array.isArray(value)) return `[${value.map(stableJSONStringify).join(",")}]`;
  if (value && typeof value === "object") {
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${stableJSONStringify(value[key])}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}
