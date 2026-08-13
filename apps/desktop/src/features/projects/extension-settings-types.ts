export type ExtensionSettingsSection =
  | "extensions"
  | "mcp"
  | "agents"
  | "skills"
  | "tools";
export type McpAddInputMode = "json" | "manual";

export type KeyValueRow = {
  key: string;
  value: string;
};
