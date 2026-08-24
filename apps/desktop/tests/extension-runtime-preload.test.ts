import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

test("extension preload exposes queued Chrome-style runtime Port messaging", async () => {
  const source = await readFile(
    new URL("../electron/extension-preload.cjs", import.meta.url),
    "utf8",
  );
  const calls: Array<{ channel: string; input: unknown }> = [];
  const handlers = new Map<string, (_event: unknown, payload: unknown) => void>();
  let exposed: Record<string, any> | null = null;
  let resolveOpen: ((value: { opened: boolean }) => void) | null = null;
  const openPromise = new Promise<{ opened: boolean }>((resolve) => {
    resolveOpen = resolve;
  });
  const ipcRenderer = {
    invoke(channel: string, input?: unknown) {
      calls.push({ channel, input });
      if (channel === "aivo:extension-runtime-open-port") return openPromise;
      if (channel === "aivo:extension-runtime-send-message") {
        return Promise.resolve({ message: { type: "pong" } });
      }
      return Promise.resolve({ closed: true, posted: true });
    },
    on(channel: string, handler: (_event: unknown, payload: unknown) => void) {
      handlers.set(channel, handler);
    },
    removeListener() {},
  };
  vm.runInNewContext(source, {
    console,
    Date,
    Map,
    Object,
    Promise,
    Set,
    require(id: string) {
      assert.equal(id, "electron");
      return {
        contextBridge: {
          exposeInMainWorld(_name: string, value: Record<string, any>) {
            exposed = value;
          },
        },
        ipcRenderer,
      };
    },
  });

  assert.ok(exposed);
  const runtime = exposed.aivoExtension?.runtime ?? exposed.runtime;
  assert.ok(runtime);
  assert.deepEqual(await runtime.sendMessage({ type: "ping" }), {
    message: { type: "pong" },
  });

  const port = runtime.connect({ name: "state-stream" });
  const messages: unknown[] = [];
  const disconnects: unknown[] = [];
  port.onMessage.addListener((message: unknown) => messages.push(message));
  port.onDisconnect.addListener((_port: unknown, event: unknown) =>
    disconnects.push(event),
  );
  port.postMessage({ queued: true });
  assert.equal(
    calls.filter(
      ({ channel }) => channel === "aivo:extension-runtime-post-port-message",
    ).length,
    0,
  );

  resolveOpen?.({ opened: true });
  await new Promise((resolve) => setImmediate(resolve));
  const post = calls.find(
    ({ channel }) => channel === "aivo:extension-runtime-post-port-message",
  );
  assert.deepEqual((post?.input as any)?.message, { queued: true });

  const clientPortId = (post?.input as any)?.clientPortId;
  handlers.get("aivo:extension-runtime-port-message")?.({}, {
    clientPortId,
    message: { type: "tool-state" },
  });
  assert.deepEqual(messages, [{ type: "tool-state" }]);

  port.disconnect();
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(disconnects.length, 1);
  assert.throws(() => port.postMessage({ after: "close" }), /disconnected/);
});
