import { access, readFile } from "node:fs/promises";
import { spawn } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const appDirectory = resolve(scriptDirectory, "..");
const serverUrlFile = resolve(appDirectory, "dist/.vite-dev-server-url");
const requiredBuildFiles = [
  resolve(appDirectory, "dist/main/main.cjs"),
  resolve(appDirectory, "dist/preload/preload.cjs"),
];

const serverUrl = await waitForDevelopmentServer();
const packageManager = process.platform === "win32" ? "pnpm.cmd" : "pnpm";
const electron = spawn(packageManager, ["exec", "electron", "."], {
  cwd: appDirectory,
  env: { ...process.env, VITE_DEV_SERVER_URL: serverUrl },
  stdio: "inherit",
});

electron.once("error", (error) => {
  console.error(`Could not start Electron: ${error.message}`);
  process.exitCode = 1;
});
electron.once("exit", (code, signal) => {
  if (signal) {
    process.exitCode = 1;
    return;
  }
  process.exitCode = code ?? 1;
});

async function waitForDevelopmentServer() {
  for (;;) {
    try {
      const [serverUrl] = await Promise.all([
        readFile(serverUrlFile, "utf8"),
        ...requiredBuildFiles.map((file) => access(file)),
      ]);
      const parsedUrl = new URL(serverUrl.trim());
      if (parsedUrl.protocol !== "http:" && parsedUrl.protocol !== "https:") {
        throw new Error("Development server URL must use HTTP or HTTPS.");
      }
      return parsedUrl.toString();
    } catch (error) {
      if (error instanceof Error && error.message.includes("must use HTTP")) {
        throw error;
      }
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
  }
}
