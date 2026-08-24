import {
  isToolInjectionResourceKind,
  type ToolInjectionResourceKind,
} from "./tool-injection-resource-model.ts";

const maxResourceCount = 256;
const maxResourceIdBytes = 512;
const maxResourceNameBytes = 512;

export type HostToolSelectionResource = {
  id: string;
  kind: ToolInjectionResourceKind;
  name: string;
  toolCount: number;
};

export type HostToolSelectionNote = {
  lifetime?: "conversation" | "request";
  resources: HostToolSelectionResource[];
  status: "completed" | "failed" | "running";
};

export function hostToolSelectionFromSystemNote(note: {
  payload?: Record<string, unknown>;
}): HostToolSelectionNote | null {
  const payload = note.payload;
  if (payload?.kind !== "host_tool_selection") return null;
  const status = payload.status ?? "completed";
  if (status !== "completed" && status !== "failed" && status !== "running") {
    return null;
  }
  const rawResources = payload.resources;
  if (status !== "completed") {
    if (!Array.isArray(rawResources) || rawResources.length !== 0) return null;
    return { resources: [], status };
  }
  const lifetime = payload.lifetime;
  if (lifetime !== "conversation" && lifetime !== "request") return null;
  if (!Array.isArray(rawResources) || rawResources.length > maxResourceCount) {
    return null;
  }

  const seen = new Set<string>();
  const resources: HostToolSelectionResource[] = [];
  for (const value of rawResources) {
    if (!value || typeof value !== "object" || Array.isArray(value)) return null;
    const resource = value as Record<string, unknown>;
    if (
      Object.keys(resource).some(
        (key) => !["id", "kind", "name", "toolCount"].includes(key),
      )
    ) {
      return null;
    }
    const { id, kind, name, toolCount } = resource;
    if (!isToolInjectionResourceKind(kind)) return null;
    if (
      typeof id !== "string" ||
      id !== id.trim() ||
      !id ||
      new TextEncoder().encode(id).length > maxResourceIdBytes ||
      typeof name !== "string" ||
      name !== name.trim() ||
      !name ||
      new TextEncoder().encode(name).length > maxResourceNameBytes ||
      !Number.isInteger(toolCount) ||
      (toolCount as number) < 0 ||
      (kind !== "skill" && (toolCount as number) === 0) ||
      (toolCount as number) > 10000
    ) {
      return null;
    }
    const key = `${kind}\0${id}`;
    if (seen.has(key)) return null;
    seen.add(key);
    resources.push({
      id,
      kind,
      name,
      toolCount: toolCount as number,
    });
  }
  return { lifetime, resources, status };
}
