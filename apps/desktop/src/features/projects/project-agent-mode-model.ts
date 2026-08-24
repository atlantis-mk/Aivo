import type { AgentModeDefinition, AgentModeId } from "@/services/aivo";

export const fallbackAgentModes: AgentModeDefinition[] = [
  {
    id: "assistant",
    displayName: "Assistant",
    description: "通用对话，必要时可编码",
    prompt: "",
  },
];

export function agentModeShortLabel(mode: AgentModeDefinition) {
  switch (mode.id) {
    case "assistant":
      return "助手";
    case "summary":
      return "总结";
    case "title":
      return "标题";
    default:
      return mode.displayName;
  }
}

export function agentModeLabel(mode: AgentModeId | string) {
  const definition =
    fallbackAgentModes.find((item) => item.id === mode) ??
    fallbackAgentModes[0];
  return agentModeShortLabel(definition);
}

export function normalizeAgentMode(
  mode: AgentModeId | string | undefined,
): AgentModeId {
  switch (mode) {
    case "assistant":
    case "summary":
    case "title":
    case "scheduler_worker":
      return mode;
    case "code":
    case "build":
    case "explore":
    case "plan":
    case "review":
    case "debug":
    case "planner":
      return "assistant";
    default:
      return "assistant";
  }
}
