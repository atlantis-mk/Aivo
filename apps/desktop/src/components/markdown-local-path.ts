const MAX_LOCAL_PATH_LENGTH = 4096;
const LOCAL_PATH_LINK_PREFIX = "#aivo-local-path=";

export function localPathFromText(
  value: string,
  platform: string | undefined,
): string | undefined {
  const candidate = stripSourceLocation(value.trim());
  if (
    !candidate ||
    candidate.length > MAX_LOCAL_PATH_LENGTH ||
    candidate.includes("\0") ||
    candidate.includes("\n") ||
    candidate.includes("\r")
  ) {
    return undefined;
  }

  if (platform === "win32") {
    return isWindowsAbsolutePath(candidate) ? candidate : undefined;
  }

  return candidate.startsWith("/") ? candidate : undefined;
}

export function markdownHrefForLocalPath(
  value: string,
  platform: string | undefined,
): string | undefined {
  const decoded = safelyDecodeURIComponent(value);
  const target = localPathFromText(decoded, platform);
  return target
    ? `${LOCAL_PATH_LINK_PREFIX}${encodeURIComponent(target)}`
    : undefined;
}

export function localPathFromMarkdownHref(value: string): string | undefined {
  if (!value.startsWith(LOCAL_PATH_LINK_PREFIX)) return undefined;

  const encodedTarget = value.slice(LOCAL_PATH_LINK_PREFIX.length);
  const target = safelyDecodeURIComponent(encodedTarget);
  return target && target.length <= MAX_LOCAL_PATH_LENGTH ? target : undefined;
}

function isWindowsAbsolutePath(value: string) {
  return /^[A-Za-z]:[\\/]/.test(value) || /^\\\\[^\\/]+[\\/][^\\/]+/.test(value);
}

function stripSourceLocation(value: string) {
  return value
    .replace(/#L\d+(?:C\d+)?$/i, "")
    .replace(/:(\d+)(?::\d+)?$/, "");
}

function safelyDecodeURIComponent(value: string) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
