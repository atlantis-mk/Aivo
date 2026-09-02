import {
  isRequiredCoreToolName,
  isStandaloneToolResource,
  type ToolInjectionResourceKind,
} from "./tool-injection-resource-model.ts";

export type PromptMentionRange = { query: string; start: number };

export type PromptMentionReference = {
  id: string;
  kind: "project" | ToolInjectionResourceKind;
  rootPath?: string;
  token: string;
};

export type PromptMentionItem = {
  detail?: string;
  id: string;
  label: string;
  reference: PromptMentionReference;
  token: string;
  type: "项目" | "技能" | "工具" | "扩展" | "MCP";
};

export type PromptMentionAction = {
  action: "compact-context" | "select-local";
  detail: string;
  id: string;
  label: string;
};

export type PromptMentionProject = {
  id?: string;
  name?: string;
  rootPath: string;
};

const promptMentionGroupOrder: PromptMentionItem["type"][] = [
  "项目",
  "技能",
  "扩展",
  "MCP",
  "工具",
];

const promptMentionActions: PromptMentionAction[] = [
  {
    action: "compact-context",
    detail: "@compact · 默认 80% 自动触发",
    id: "action:compact-context",
    label: "压缩上下文",
  },
  {
    action: "select-local",
    detail: "文件或文件夹",
    id: "action:select-local",
    label: "选择文件或文件夹",
  },
];

export function promptMentionRange(value: string, caret: number): PromptMentionRange | null {
  const beforeCaret = value.slice(0, caret);
  const start = beforeCaret.lastIndexOf("@");
  if (start < 0 || (start > 0 && !/\s/.test(value[start - 1]))) return null;
  const query = beforeCaret.slice(start + 1);
  return /\s/.test(query) ? null : { query, start };
}

export function fuzzyMatch(value: string, query: string) {
  const needle = query.trim().toLocaleLowerCase();
  if (!needle) return true;
  let index = 0;
  for (const character of value.toLocaleLowerCase()) {
    if (character === needle[index]) index += 1;
    if (index === needle.length) return true;
  }
  return false;
}

export function filterPromptMentionItems(items: PromptMentionItem[], query: string) {
  return items.filter((item) => fuzzyMatch(`${item.label} ${item.detail ?? ""} ${item.token}`, query));
}

export function filterPromptMentionActions(query: string) {
  return promptMentionActions.filter((item) =>
    fuzzyMatch(`${item.label} ${item.detail}`, query)
  );
}

export function groupPromptMentionItems(items: PromptMentionItem[]) {
  return promptMentionGroupOrder.flatMap((type) => {
    const groupItems = items.filter((item) => item.type === type);
    return groupItems.length ? [{ items: groupItems, type }] : [];
  });
}

export function isPromptMentionBuiltinTool(tool: {
  activationPolicy?: string;
  category?: string;
  enabled: boolean;
  name?: string;
  source: string;
  sourceId?: string;
  toolsets?: string[];
}) {
  return (
    tool.enabled &&
    !isRequiredCoreToolName(tool.name) &&
    isStandaloneToolResource(tool)
  );
}

export function promptMentionProjectItems(
  projects: PromptMentionProject[],
  currentProjectPath: string,
): PromptMentionItem[] {
  const currentPath = currentProjectPath.trim();
  const byPath = new Map<string, PromptMentionProject>();
  for (const project of projects) {
    const rootPath = project.rootPath.trim();
    if (rootPath && !byPath.has(rootPath)) byPath.set(rootPath, project);
  }
  return [...byPath.values()]
    .filter((project) => Boolean(project.id?.trim()))
    .toSorted((left, right) =>
      Number(right.rootPath === currentPath) - Number(left.rootPath === currentPath),
    )
    .map((project) => {
      const label = project.name?.trim() || projectNameFromPath(project.rootPath);
      const current = project.rootPath === currentPath;
      return {
        detail: current ? `${project.rootPath} · 当前项目` : project.rootPath,
        id: `project:${project.id || project.rootPath}`,
        label,
        reference: {
          id: project.id!,
          kind: "project",
          rootPath: project.rootPath,
          token: label,
        },
        token: label,
        type: "项目",
      };
    });
}

export function activePromptMentionReferences(
  references: PromptMentionReference[],
) {
  const seen = new Set<string>();
  return references.filter((reference) => {
    const key = `${reference.kind}\u0000${reference.id}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function addPromptMentionReference(
  references: PromptMentionReference[],
  reference: PromptMentionReference,
) {
  const withoutDuplicate = references.filter((item) =>
    item.kind !== reference.kind || item.id !== reference.id
  );
  if (reference.kind !== "project") {
    return [...withoutDuplicate, reference];
  }
  return [
    ...withoutDuplicate.filter((item) => item.kind !== "project"),
    reference,
  ];
}

export function removePromptMentionReference(
  references: PromptMentionReference[],
  reference: PromptMentionReference,
) {
  return references.filter((item) =>
    item.kind !== reference.kind || item.id !== reference.id
  );
}

function projectNameFromPath(rootPath: string) {
  const parts = rootPath.trim().replace(/[\\/]+$/, "").split(/[\\/]/).filter(Boolean);
  return parts.at(-1) || "Project";
}

export function consumePromptMentionQuery(
  value: string,
  caret: number,
  range: PromptMentionRange,
) {
  const prefix = value.slice(0, range.start);
  const suffix = value.slice(caret);
  if (!prefix) {
    const nextSuffix = suffix.replace(/^\s/, "");
    return { caret: 0, value: nextSuffix };
  }
  if (/\s$/.test(prefix) && /^\s/.test(suffix)) {
    return { caret: prefix.length, value: `${prefix}${suffix.slice(1)}` };
  }
  return { caret: prefix.length, value: `${prefix}${suffix}` };
}

export function promptComposerEnterAction(
  mentionOpen: boolean,
  shiftKey: boolean,
  isComposing: boolean,
) {
  if (isComposing) return "none" as const;
  if (shiftKey) return "newline" as const;
  return mentionOpen ? "mention" as const : "submit" as const;
}
