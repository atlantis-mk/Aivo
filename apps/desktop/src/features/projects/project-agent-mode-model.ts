import type { AgentModeDefinition, AgentModeId } from "@/services/aivo";

export const fallbackAgentModes: AgentModeDefinition[] = [
  {
    id: "code",
    displayName: "Code",
    description: "默认编码主代理，按权限使用完整工具",
    prompt: "",
    toolsets: [],
  },
  {
    id: "assistant",
    displayName: "Assistant",
    description: "通用对话，必要时可编码",
    prompt: "",
    toolsets: [],
  },
  {
    id: "build",
    displayName: "Build",
    description: "实现代码、编辑文件、运行验证",
    prompt: "",
    toolsets: [],
  },
  {
    id: "explore",
    displayName: "Explore",
    description: "只读探索代码与方案",
    prompt: "",
    toolsets: [],
  },
  {
    id: "plan",
    displayName: "Plan",
    description: "只规划，不修改",
    prompt: "",
    toolsets: [],
  },
  {
    id: "planner",
    displayName: "Planner",
    description: "Plan 兼容模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
  {
    id: "review",
    displayName: "Review",
    description: "只读审查代码风险",
    prompt: "",
    toolsets: [],
  },
  {
    id: "debug",
    displayName: "Debug",
    description: "诊断问题，可运行验证但不改文件",
    prompt: "",
    toolsets: [],
  },
  {
    id: "summary",
    displayName: "Summary",
    description: "隐藏总结模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
  {
    id: "title",
    displayName: "Title",
    description: "隐藏标题模式",
    prompt: "",
    toolsets: [],
    hidden: true,
  },
];

export function agentModeShortLabel(mode: AgentModeDefinition) {
  switch (mode.id) {
    case "code":
      return "代码";
    case "assistant":
      return "助手";
    case "build":
      return "构建";
    case "explore":
      return "探索";
    case "plan":
    case "planner":
      return "规划";
    case "review":
      return "审查";
    case "debug":
      return "调试";
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
    case "code":
    case "assistant":
    case "build":
    case "explore":
    case "plan":
    case "review":
    case "debug":
    case "summary":
    case "title":
    case "scheduler_worker":
      return mode;
    case "planner":
      return "plan";
    default:
      return "code";
  }
}
