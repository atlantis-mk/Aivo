#!/usr/bin/env node

import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { dirname, join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const extensionRoot = dirname(fileURLToPath(import.meta.url));
const MAX_BODY_BYTES = 1024 * 1024;
const MAX_STATES = 100;
const STATE_TTL_MS = 10 * 60 * 1000;

export const TOOL_NAME = "ui_test_show";
export const TOOL_SCHEMA = Object.freeze({
  type: "object",
  properties: {
    message: {
      type: "string",
      description: "Message to display in the test panel.",
    },
  },
  required: ["message"],
  additionalProperties: false,
});

export const TOOL_SCHEMA_HASH = createHash("sha256")
  .update(stableJSONStringify(TOOL_SCHEMA))
  .digest("hex");

const assetPaths = new Map([
  ["/ui", { path: join(extensionRoot, "ui", "index.html"), type: "text/html; charset=utf-8" }],
  ["/ui/", { path: join(extensionRoot, "ui", "index.html"), type: "text/html; charset=utf-8" }],
  ["/ui/app.js", { path: join(extensionRoot, "ui", "app.js"), type: "text/javascript; charset=utf-8" }],
  ["/ui/style.css", { path: join(extensionRoot, "ui", "style.css"), type: "text/css; charset=utf-8" }],
]);

export function createExtensionServer({ bearerToken, now = () => Date.now() } = {}) {
  const token = String(bearerToken ?? "").trim();
  if (!token) {
    throw new Error("AIVO_EXTENSION_BEARER_TOKEN is required");
  }

  const states = new Map();
  const runtimePorts = new Map();
  let initialized = false;
  let active = false;

  const server = createServer(async (request, response) => {
    try {
      if (!authorized(request, token)) {
        sendJSON(response, 401, { error: "unauthorized" });
        return;
      }

      const requestURL = new URL(request.url ?? "/", "http://127.0.0.1");
      pruneStates(states, now());

      if (
        request.method === "POST" &&
        requestURL.pathname === "/.well-known/aivo-runtime/messages"
      ) {
        const body = await readJSONBody(request);
        sendJSON(response, 200, {
          message: {
            type: "runtime.response",
            received: body.message ?? null,
            active,
            retainedOperations: states.size,
          },
        });
        return;
      }

      const runtimePortRoute = runtimePortPath(requestURL.pathname);
      if (runtimePortRoute) {
        const port = runtimePorts.get(runtimePortRoute.portId);
        if (request.method === "GET" && !runtimePortRoute.messages) {
          if (port) {
            sendJSON(response, 409, { error: "port_exists" });
            return;
          }
          const viewId = boundedText(request.headers["x-aivo-view-id"], 100);
          const name = boundedText(request.headers["x-aivo-port-name"], 100);
          if (viewId !== "ui-test-detail" || name !== "state-stream") {
            sendJSON(response, 400, { error: "invalid_port_identity" });
            return;
          }
          const nextPort = {
            id: runtimePortRoute.portId,
            name,
            response,
            selectedOperationId: "",
            viewId,
          };
          runtimePorts.set(nextPort.id, nextPort);
          response.writeHead(200, {
            "Cache-Control": "no-store",
            Connection: "keep-alive",
            "Content-Type": "application/x-ndjson; charset=utf-8",
            "X-Content-Type-Options": "nosniff",
          });
          response.flushHeaders();
          response.on("close", () => runtimePorts.delete(nextPort.id));
          sendRuntimePortEvent(nextPort, { type: "connected", name });
          return;
        }
        if (request.method === "POST" && runtimePortRoute.messages) {
          if (!port) {
            sendJSON(response, 404, { error: "port_not_found" });
            return;
          }
          const body = await readJSONBody(request);
          const message = body.message ?? null;
          if (message?.type === "select-operation") {
            port.selectedOperationId = boundedText(message.operationId, 200);
            const state = states.get(port.selectedOperationId);
            if (state) sendRuntimePortEvent(port, { type: "tool-state", state });
          } else {
            sendRuntimePortEvent(port, { type: "message", message });
          }
          sendJSON(response, 202, { ok: true });
          return;
        }
        if (request.method === "DELETE" && !runtimePortRoute.messages) {
          if (port) closeRuntimePort(runtimePorts, port);
          sendJSON(response, 200, { closed: Boolean(port) });
          return;
        }
      }

      if (request.method === "POST" && requestURL.pathname === "/") {
        const rpcRequest = await readJSONBody(request);
        const outcome = await handleRPC(rpcRequest, {
          states,
          runtimePorts,
          now,
          get initialized() {
            return initialized;
          },
          set initialized(value) {
            initialized = value;
          },
          get active() {
            return active;
          },
          set active(value) {
            active = value;
          },
        });
        sendJSON(response, 200, outcome.envelope);
        if (outcome.shutdown) {
          setImmediate(() => server.close());
        }
        return;
      }

      if (request.method === "GET" && requestURL.pathname === "/ui/state") {
        const operationId = boundedText(requestURL.searchParams.get("operationId"), 200);
        const state = states.get(operationId);
        if (!state) {
          sendJSON(response, 404, {
            ok: false,
            error: "No retained test state exists for this operation.",
          });
          return;
        }
        sendJSON(response, 200, { ok: true, state });
        return;
      }

      const asset = assetPaths.get(requestURL.pathname);
      if (request.method === "GET" && asset) {
        const body = await readFile(asset.path);
        response.writeHead(200, {
          "Cache-Control": "no-store",
          "Content-Length": body.byteLength,
          "Content-Type": asset.type,
          "X-Content-Type-Options": "nosniff",
        });
        response.end(body);
        return;
      }

      sendJSON(response, 404, { error: "not_found" });
    } catch (error) {
      sendJSON(response, error?.code === "body_too_large" ? 413 : 400, {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  });

  server.on("close", () => {
    for (const port of [...runtimePorts.values()]) closeRuntimePort(runtimePorts, port);
  });

  return server;
}

async function handleRPC(request, runtime) {
  const id = request?.id;
  if (request?.jsonrpc !== "2.0" || typeof id !== "string" || !request.method) {
    return rpcError(id ?? null, -32600, "Invalid JSON-RPC request");
  }

  const params = objectValue(request.params);
  switch (request.method) {
    case "extension/initialize": {
      if (
        params.apiVersion !== "2" ||
        params.extensionId !== "dev.aivo.ui-test" ||
        params.extensionVersion !== "0.3.0"
      ) {
        return rpcError(id, -32602, "Unsupported extension identity or API version");
      }
      runtime.initialized = true;
      return rpcResult(id, {
        apiVersion: "2",
        capabilities: { tools: true, views: true, actions: true, runtimeMessaging: true },
      });
    }
    case "catalog/list":
      if (!runtime.initialized) return rpcError(id, -32000, "Extension is not initialized");
      return rpcResult(id, {
        tools: [{ name: TOOL_NAME, schemaHash: TOOL_SCHEMA_HASH }],
      });
    case "extension/activate":
      runtime.active = true;
      return rpcResult(id, { active: true });
    case "extension/deactivate":
      runtime.active = false;
      return rpcResult(id, { active: false });
    case "health/check":
      return rpcResult(id, {
        ready: runtime.initialized,
        active: runtime.active,
        retainedOperations: runtime.states.size,
      });
    case "tool/execute":
      return executeTool(id, params, runtime);
    case "ui/event":
      return handleUIEvent(id, params, runtime);
    case "extension/shutdown":
      for (const port of [...runtime.runtimePorts.values()]) {
        closeRuntimePort(runtime.runtimePorts, port);
      }
      return { ...rpcResult(id, { stopped: true }), shutdown: true };
    default:
      return rpcError(id, -32601, `Unsupported method: ${request.method}`);
  }
}

function executeTool(id, params, runtime) {
  if (params.name !== TOOL_NAME) {
    return rpcError(id, -32602, `Unsupported tool: ${String(params.name ?? "")}`);
  }

  const operationId = boundedText(params.operationId, 200);
  const message = boundedText(objectValue(params.arguments).message, 2000);
  if (!operationId || !message) {
    return rpcError(id, -32602, "operationId and arguments.message are required");
  }

  const state = {
    operationId,
    sessionId: boundedText(params.sessionId, 200),
    turnId: boundedText(params.turnId, 200),
    toolName: TOOL_NAME,
    message,
    receivedAt: new Date(runtime.now()).toISOString(),
    actionCount: 0,
    lastAction: null,
  };
  runtime.states.set(operationId, state);
  trimOldestStates(runtime.states);
  for (const port of runtime.runtimePorts.values()) {
    if (!port.selectedOperationId || port.selectedOperationId === operationId) {
      sendRuntimePortEvent(port, { type: "tool-state", state });
    }
  }

  return rpcResult(id, {
    ok: true,
    content: `Opened the Aivo UI test panel with message: ${message}`,
    operationId,
    receivedAt: state.receivedAt,
  });
}

function handleUIEvent(id, params, runtime) {
  if (params.viewId !== "ui-test-detail") {
    return rpcError(id, -32602, "Unknown view");
  }
  if (params.action !== "test.record" && params.action !== "view.refresh") {
    return rpcError(id, -32602, "Undeclared action");
  }

  const data = objectValue(params.data);
  const operationId = boundedText(data.operationId, 200);
  const state = runtime.states.get(operationId);
  if (!state) {
    return rpcResult(id, { ok: false, error: "Operation state is unavailable" });
  }

  if (params.action === "test.record") {
    state.actionCount += 1;
    state.lastAction = {
      label: boundedText(data.label, 120) || "Interaction",
      time: new Date(runtime.now()).toISOString(),
    };
  }
  for (const port of runtime.runtimePorts.values()) {
    if (port.selectedOperationId === operationId) {
      sendRuntimePortEvent(port, { type: "tool-state", state });
    }
  }
  return rpcResult(id, { ok: true, state });
}

function runtimePortPath(pathname) {
  const match = /^\/\.well-known\/aivo-runtime\/ports\/([A-Za-z0-9-]{1,100})(\/messages)?$/.exec(pathname);
  return match ? { portId: match[1], messages: Boolean(match[2]) } : null;
}

function sendRuntimePortEvent(port, event) {
  if (port.response.destroyed || port.response.writableEnded) return;
  port.response.write(`${JSON.stringify(event)}\n`);
}

function closeRuntimePort(ports, port) {
  ports.delete(port.id);
  if (!port.response.writableEnded) port.response.end();
}

function rpcResult(id, result) {
  return { envelope: { jsonrpc: "2.0", id, result }, shutdown: false };
}

function rpcError(id, code, message) {
  return {
    envelope: { jsonrpc: "2.0", id, error: { code, message } },
    shutdown: false,
  };
}

async function readJSONBody(request) {
  const chunks = [];
  let size = 0;
  for await (const chunk of request) {
    size += chunk.byteLength;
    if (size > MAX_BODY_BYTES) {
      const error = new Error("Request body exceeds 1 MiB");
      error.code = "body_too_large";
      throw error;
    }
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8"));
}

function authorized(request, token) {
  return request.headers.authorization === `Bearer ${token}`;
}

function objectValue(value) {
  return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

function boundedText(value, maxLength) {
  return typeof value === "string" ? value.trim().slice(0, maxLength) : "";
}

function pruneStates(states, currentTime) {
  for (const [operationId, state] of states) {
    if (currentTime - Date.parse(state.receivedAt) > STATE_TTL_MS) {
      states.delete(operationId);
    }
  }
}

function trimOldestStates(states) {
  while (states.size > MAX_STATES) {
    states.delete(states.keys().next().value);
  }
}

function sendJSON(response, status, value) {
  if (response.headersSent) return;
  const body = Buffer.from(JSON.stringify(value));
  response.writeHead(status, {
    "Cache-Control": "no-store",
    "Content-Length": body.byteLength,
    "Content-Type": "application/json; charset=utf-8",
    "X-Content-Type-Options": "nosniff",
  });
  response.end(body);
}

function stableJSONStringify(value) {
  if (Array.isArray(value)) {
    return `[${value.map(stableJSONStringify).join(",")}]`;
  }
  if (value && typeof value === "object") {
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${stableJSONStringify(value[key])}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}

function readPort(argv) {
  const index = argv.indexOf("--port");
  const parsed = index >= 0 ? Number.parseInt(argv[index + 1] ?? "", 10) : 0;
  if (!Number.isInteger(parsed) || parsed < 0 || parsed > 65535) {
    throw new Error("--port must be an integer between 0 and 65535");
  }
  return parsed;
}

function isMainModule() {
  return process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
}

if (isMainModule()) {
  const bearerToken = process.env.AIVO_EXTENSION_BEARER_TOKEN;
  const port = readPort(process.argv.slice(2));
  const server = createExtensionServer({ bearerToken });
  server.listen(port, "127.0.0.1");
  server.on("listening", () => {
    const address = server.address();
    if (!address || typeof address === "string" || address.port < 1) {
      server.close();
      process.exitCode = 1;
      return;
    }
    process.stdout.write(
      `${JSON.stringify({
        protocol: "aivo-extension-service/1",
        url: `http://127.0.0.1:${address.port}`,
      })}\n`,
    );
  });
  server.on("error", (error) => {
    process.stderr.write(`Aivo UI test extension failed: ${error.message}\n`);
    process.exitCode = 1;
  });
}
