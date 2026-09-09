import { mkdir, rename, rm, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const appDirectory = resolve(scriptDirectory, "..");
const serverUrlFile = resolve(appDirectory, "dist/.vite-dev-server-url");

await mkdir(dirname(serverUrlFile), { recursive: true });
await rm(serverUrlFile, { force: true });

const server = await createServer({
  configFile: resolve(appDirectory, "vite.renderer.config.ts"),
  server: {
    host: "127.0.0.1",
    port: 5173,
  },
});

await server.listen();
const address = server.httpServer?.address();
if (!address || typeof address === "string") {
  throw new Error("Could not determine the Vite development server port.");
}

const serverUrl = `http://127.0.0.1:${address.port}`;
const temporaryUrlFile = `${serverUrlFile}.tmp`;
await writeFile(temporaryUrlFile, serverUrl, "utf8");
await rename(temporaryUrlFile, serverUrlFile);
server.printUrls();

let closing = false;
async function closeServer() {
  if (closing) return;
  closing = true;
  await rm(serverUrlFile, { force: true });
  await server.close();
}

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => {
    void closeServer().finally(() => process.exit(0));
  });
}
