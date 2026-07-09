export type PluginSettingsSection =
  | "plugins"
  | "apps"
  | "mcp"
  | "skills"
  | "tools";
export type AddToolMode = "plugin" | "mcp";
export type McpAddInputMode = "json" | "manual";

export type KeyValueRow = {
  key: string;
  value: string;
};
