import { accessSync, constants, mkdirSync, statSync } from "node:fs";
import path from "node:path";

const AIVO_HOME_DIRECTORY = ".aivo";

interface AivoRuntimeEnvironmentOptions {
  homeDirectory: string;
  inheritedEnvironment: NodeJS.ProcessEnv;
  injectedEnvironment?: Record<string, string>;
}

export function ensureAivoCodexHome(homeDirectory: string): string {
  const aivoHome = path.join(homeDirectory, AIVO_HOME_DIRECTORY);

  try {
    mkdirSync(aivoHome, { recursive: true });
    if (!statSync(aivoHome).isDirectory()) {
      throw new Error("path is not a directory");
    }
    accessSync(aivoHome, constants.R_OK | constants.W_OK);
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error);
    throw new Error(`Aivo runtime home is unavailable at ${aivoHome}: ${detail}`);
  }

  return aivoHome;
}

export function buildAivoRuntimeEnvironment({
  homeDirectory,
  inheritedEnvironment,
  injectedEnvironment = {},
}: AivoRuntimeEnvironmentOptions): NodeJS.ProcessEnv {
  const aivoHome = ensureAivoCodexHome(homeDirectory);
  return {
    ...inheritedEnvironment,
    ...injectedEnvironment,
    CODEX_HOME: aivoHome,
    CODEX_SQLITE_HOME: aivoHome,
  };
}
