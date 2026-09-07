export function stringArg(args: Record<string, unknown>, key: string) {
  const value = args[key];
  return typeof value === "string" ? value : "";
}

export function scalarArg(args: Record<string, unknown>, key: string) {
  const value = args[key];
  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return String(value);
  }
  return "";
}

export function optionalNumberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

export function stringValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

export function objectRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

export function arrayRecords(value: unknown): Record<string, unknown>[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => {
    const record = objectRecord(item);
    return record ? [record] : [];
  });
}

export function objectStringRecord(
  value: unknown,
): Record<string, string> | undefined {
  const record = objectRecord(value);
  if (!record) return undefined;
  return Object.fromEntries(
    Object.entries(record).filter(
      (entry): entry is [string, string] => typeof entry[1] === "string",
    ),
  );
}

export function truncateInline(value: string, max = 120) {
  const compact = value.trim().replace(/\s+/g, " ");
  if (compact.length <= max) return compact;
  return `${compact.slice(0, Math.max(0, max - 1)).trimEnd()}…`;
}

export function visibleToolArgs(
  args: Record<string, unknown>,
  skippedKeys: string[],
) {
  const skipped = new Set(skippedKeys);
  return Object.entries(args)
    .filter(([key]) => !skipped.has(key))
    .flatMap(([key, value]) => {
      if (
        typeof value === "string" ||
        typeof value === "number" ||
        typeof value === "boolean"
      ) {
        return [`${key}=${String(value)}`];
      }
      return [];
    });
}

export function toolCallArgumentLabels(args: Record<string, unknown>) {
  const skipped = new Set([
    "description",
    "query",
    "url",
    "path",
    "filePath",
    "pattern",
    "name",
  ]);
  return Object.entries(args)
    .filter(([key]) => !skipped.has(key))
    .flatMap(([key, value]) => {
      if (
        typeof value === "string" ||
        typeof value === "number" ||
        typeof value === "boolean"
      ) {
        return [`${key}=${String(value)}`];
      }
      return [];
    })
    .slice(0, 3);
}

export function joinCommandParts(parts: string[]) {
  return parts.filter(Boolean).join(" ");
}
