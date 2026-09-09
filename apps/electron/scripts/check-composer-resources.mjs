// Read-only integration check against the same runtime and home used by Aivo.
// Does not start a thread, submit a turn or change resource configuration.
import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
import os from "node:os";
import { fileURLToPath } from "node:url";
import { buildAivoRuntimeEnvironment } from "../src/codex-runtime-environment.ts";
import {
  loadCodexSkills,
  loadCodexMcpServers,
} from "../src/codex-composer-resources.ts";

const binary =
  process.env.AIVO_CODEX_BIN ||
  fileURLToPath(
    new URL(
      "../../../vendor/codex/codex-rs/target/debug/codex",
      import.meta.url,
    ),
  );
const child = spawn(binary, ["app-server", "--stdio"], {
  env: buildAivoRuntimeEnvironment({
    homeDirectory: os.homedir(),
    inheritedEnvironment: process.env,
  }),
  stdio: ["pipe", "pipe", "pipe"],
});
let nextId = 0;
const pending = new Map();
child.stderr.resume();
const lines = createInterface({ input: child.stdout });
lines.on("line", (line) => {
  let message;
  try {
    message = JSON.parse(line);
  } catch {
    return;
  }
  const request = pending.get(message.id);
  if (!request) return;
  pending.delete(message.id);
  clearTimeout(request.timer);
  if (message.error) request.reject(new Error(message.error.message));
  else request.resolve(message.result);
});
const rejectPending = (error) => {
  for (const request of pending.values()) {
    clearTimeout(request.timer);
    request.reject(error);
  }
  pending.clear();
};
child.on("error", rejectPending);
child.on("exit", (code) => rejectPending(new Error(`Runtime exited: ${code}`)));
function request(method, params) {
  return new Promise((resolve, reject) => {
    const id = ++nextId;
    const timer = setTimeout(() => {
      pending.delete(id);
      reject(new Error(`${method} timed out`));
    }, 30000);
    pending.set(id, { resolve, reject, timer });
    child.stdin.write(JSON.stringify({ id, method, params }) + "\n");
  });
}
try {
  await request("initialize", {
    clientInfo: { name: "aivo_resource_check", version: "0.1.0" },
    capabilities: { experimentalApi: true },
  });
  child.stdin.write(
    JSON.stringify({ method: "initialized", params: {} }) + "\n",
  );
  const [skills, servers] = await Promise.all([
    loadCodexSkills(
      request,
      fileURLToPath(new URL("../../..", import.meta.url)),
    ),
    loadCodexMcpServers(request),
  ]);
  const output = process.argv.includes("--preview-json")
    ? { skills: { ...skills, skills: skills.skills.slice(0, 15) }, servers }
    : {
        enabledSkills: skills.skills.filter((skill) => skill.enabled).length,
        skillScanErrors: skills.errors.length,
        mcpServers: servers.length,
        mcpTools: servers.reduce(
          (sum, server) => sum + server.toolNames.length,
          0,
        ),
        mcpDiscoveryErrors: servers.filter((server) => server.toolsError)
          .length,
      };
  console.log(JSON.stringify(output, null, 2));
} finally {
  lines.close();
  child.stdin.end();
  child.kill("SIGTERM");
}
