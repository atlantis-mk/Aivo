export const DEFAULT_APP_NAME = "Aivo";
export const MAX_APP_NAME_CHARACTERS = 40;

export function appNameFromConfig(config: unknown) {
  if (!config || typeof config !== "object") return DEFAULT_APP_NAME;
  const appName = (config as { appName?: unknown }).appName;
  return typeof appName === "string" && appName.trim()
    ? appName.trim()
    : DEFAULT_APP_NAME;
}

export function limitAppNameInput(value: string) {
  return Array.from(value).slice(0, MAX_APP_NAME_CHARACTERS).join("");
}

export function canSubmitAppName(value: string) {
  const normalized = value.trim();
  return (
    normalized.length > 0 &&
    Array.from(normalized).length <= MAX_APP_NAME_CHARACTERS &&
    !Array.from(normalized).some((character) => /\p{Cc}/u.test(character))
  );
}
