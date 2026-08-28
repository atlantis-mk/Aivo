export const toolInjectionResourceKinds = [
  "extension",
  "mcp",
  "tool",
  "skill",
] as const;

export const requiredCoreToolNames = ["read", "bash", "edit", "write"] as const;

export function isRequiredCoreToolName(name?: string) {
  return requiredCoreToolNames.some((coreName) => coreName === name);
}

export type ToolInjectionResourceKind =
  (typeof toolInjectionResourceKinds)[number];

export const toolInjectionResourceLabels: Record<
  ToolInjectionResourceKind,
  string
> = {
  extension: "扩展",
  mcp: "MCP",
  tool: "工具",
  skill: "技能",
};

export function isToolInjectionResourceKind(
  value: unknown,
): value is ToolInjectionResourceKind {
  return toolInjectionResourceKinds.some((kind) => kind === value);
}

export function isStandaloneToolResource(tool: {
  activationPolicy?: string;
  source: string;
  sourceId?: string;
}) {
  if (tool.activationPolicy === "provider_declaration") return false;
  return (
    tool.source === "builtin" ||
    (tool.source === "extension" &&
      Boolean(tool.sourceId?.startsWith("aivo.")))
  );
}
