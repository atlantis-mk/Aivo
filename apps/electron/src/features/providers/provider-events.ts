export function normalizeProviderAuthUpdatedPayload<T = unknown>(
  payloads: unknown[],
) {
  const payload = normalizeRecordPayload(payloads);
  const status = payload?.status;
  if (!status || typeof status !== "object" || Array.isArray(status)) {
    return null;
  }
  const statusRecord = status as Record<string, unknown>;
  if (typeof statusRecord.providerId !== "string") return null;
  return statusRecord as T;
}

function normalizeRecordPayload(payloads: unknown[]) {
  const first = payloads[0];
  if (first && typeof first === "object" && !Array.isArray(first)) {
    const record = first as Record<string, unknown>;
    if (record.data && typeof record.data === "object" && !Array.isArray(record.data)) {
      return record.data as Record<string, unknown>;
    }
    if (
      record.properties &&
      typeof record.properties === "object" &&
      !Array.isArray(record.properties)
    ) {
      return record.properties as Record<string, unknown>;
    }
    return record;
  }
  return null;
}
