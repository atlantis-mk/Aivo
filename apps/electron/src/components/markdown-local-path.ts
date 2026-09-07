const MAX_LOCAL_PATH_LENGTH = 4096;
const LOCAL_PATH_LINK_PREFIX = "#aivo-local-path=";
const RELATIVE_FILE_EXTENSIONS = new Set([
  "c",
  "cc",
  "conf",
  "cpp",
  "cs",
  "css",
  "csv",
  "cxx",
  "env",
  "go",
  "graphql",
  "h",
  "hpp",
  "htm",
  "html",
  "ini",
  "java",
  "js",
  "json",
  "jsonl",
  "jsx",
  "log",
  "md",
  "mdx",
  "mjs",
  "php",
  "plist",
  "properties",
  "proto",
  "py",
  "rb",
  "rs",
  "scss",
  "sh",
  "sql",
  "svelte",
  "swift",
  "toml",
  "ts",
  "tsx",
  "txt",
  "vue",
  "xml",
  "yaml",
  "yml",
  "zsh",
]);

export function localPathFromText(
  value: string,
  platform: string | undefined,
  workspaceRoot = "",
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
    if (isWindowsAbsolutePath(candidate)) return candidate;
  } else if (candidate.startsWith("/")) {
    return candidate;
  }

  if (!looksLikeRelativeFilePath(candidate)) return undefined;
  return resolvePathWithinWorkspace(candidate, workspaceRoot, platform);
}

export function markdownHrefForLocalPath(
  value: string,
  platform: string | undefined,
  workspaceRoot = "",
): string | undefined {
  const decoded = safelyDecodeURIComponent(value);
  const target = localPathFromText(decoded, platform, workspaceRoot);
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

function looksLikeRelativeFilePath(value: string) {
  if (
    !value ||
    /^[A-Za-z][A-Za-z\d+.-]*:/.test(value) ||
    value.startsWith("/") ||
    value.startsWith("\\\\")
  ) {
    return false;
  }

  const fileName = value.split(/[\\/]/).at(-1)?.trim() ?? "";
  const extension = fileName.includes(".")
    ? fileName.slice(fileName.lastIndexOf(".") + 1).toLowerCase()
    : "";
  return RELATIVE_FILE_EXTENSIONS.has(extension);
}

function resolvePathWithinWorkspace(
  relativePath: string,
  workspaceRoot: string,
  platform: string | undefined,
) {
  const root = workspaceRoot.trim();
  if (!root || root.length > MAX_LOCAL_PATH_LENGTH) return undefined;

  return platform === "win32"
    ? resolveWindowsPathWithinRoot(relativePath, root)
    : resolvePosixPathWithinRoot(relativePath, root);
}

function resolvePosixPathWithinRoot(relativePath: string, root: string) {
  if (!root.startsWith("/")) return undefined;

  const rootSegments = root.split("/").filter(Boolean);
  const resolved = resolveRelativeSegments(
    rootSegments,
    relativePath.split("/"),
  );
  return resolved ? `/${resolved.join("/")}` : undefined;
}

function resolveWindowsPathWithinRoot(relativePath: string, root: string) {
  const normalizedRoot = root.replaceAll("/", "\\");
  const rootMatch = normalizedRoot.match(
    /^([A-Za-z]:\\|\\\\[^\\]+\\[^\\]+)(.*)$/,
  );
  if (!rootMatch) return undefined;

  const prefix = rootMatch[1];
  const rootSegments = rootMatch[2].split("\\").filter(Boolean);
  const resolved = resolveRelativeSegments(
    rootSegments,
    relativePath.replaceAll("/", "\\").split("\\"),
  );
  if (!resolved) return undefined;

  const separator = prefix.endsWith("\\") ? "" : "\\";
  return `${prefix}${separator}${resolved.join("\\")}`;
}

function resolveRelativeSegments(rootSegments: string[], relativeSegments: string[]) {
  const resolved = [...rootSegments];
  const boundary = resolved.length;

  for (const segment of relativeSegments) {
    if (!segment || segment === ".") continue;
    if (segment === "..") {
      if (resolved.length === boundary) return undefined;
      resolved.pop();
      continue;
    }
    resolved.push(segment);
  }

  return resolved;
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
