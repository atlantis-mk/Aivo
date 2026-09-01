import { useEffect, useMemo, useRef, useState } from "react";

import { ArrowShrink01Icon, Folder01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { PromptMentionIcon } from "@/features/projects/project-prompt-mention-icon";
import { filterPromptMentionActions, filterPromptMentionItems, groupPromptMentionItems, isPromptMentionBuiltinTool, promptMentionProjectItems, type PromptMentionAction, type PromptMentionItem, type PromptMentionProject } from "@/features/projects/project-prompt-mention-model";
import {
  groupSkillCatalogEntries,
  groupToolCatalogEntries,
} from "@/features/projects/project-tool-activation-model";
import { hasAppBridge } from "@/lib/app-config";
import { listMCPServers } from "@/services/aivo/mcp-service";
import { listExtensionInstalls } from "@/services/aivo/extension-service";
import { listSkills } from "@/services/aivo/skill-service";
import { listToolCatalog } from "@/services/aivo/tool-catalog-service";

export function PromptMentionPicker({ activeIndex, onSelect, onSelectAction, projectPath, projects, query }: {
  activeIndex: number;
  onSelect: (item: PromptMentionItem) => void;
  onSelectAction: (item: PromptMentionAction) => void;
  projectPath: string;
  projects: PromptMentionProject[];
  query: string;
}) {
  const [catalogItems, setCatalogItems] = useState<PromptMentionItem[]>([]);
  const pickerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hasAppBridge()) {
      setCatalogItems([]);
      return;
    }
    let cancelled = false;
    void Promise.allSettled([
      listToolCatalog(projectPath),
      listSkills({
        workspaceRoot: projectPath,
        includeDisabled: false,
      }),
      listMCPServers(false, false),
      listExtensionInstalls(),
    ] as const).then(([toolsResult, skillsResult, serversResult, extensionsResult]) => {
      if (cancelled) return;
      const tools = toolsResult.status === "fulfilled" ? toolsResult.value : [];
      const skills = skillsResult.status === "fulfilled"
        ? skillsResult.value.entries ?? []
        : [];
      const servers = serversResult.status === "fulfilled"
        ? serversResult.value
        : [];
      const extensions = extensionsResult.status === "fulfilled"
        ? extensionsResult.value
        : [];
      const toolResources = groupToolCatalogEntries(
        tools.filter(isPromptMentionBuiltinTool),
        {},
      );
      const skillResources = groupSkillCatalogEntries(
        skills.filter((skill) => skill.enabled),
      );
      setCatalogItems([
        ...skillResources.map((resource) => {
          const resourceID = resource.grouped
            ? `skill-group:${resource.skills[0]!.selectionGroup!.id}`
            : resource.skills[0]!.id;
          return {
            detail: resource.grouped
              ? [`${resource.skills.length} 个技能`, resource.description]
                  .filter(Boolean)
                  .join(" · ")
              : resource.description,
            id: `skill:${resourceID}`,
            label: resource.label,
            reference: {
              id: resourceID,
              kind: "skill" as const,
              token: resource.label,
            },
            token: resource.label,
            type: "技能" as const,
          };
        }),
        ...toolResources.map((resource) => {
          const firstTool = resource.tools[0]!;
          const resourceID = resource.grouped
            ? firstTool.selectionGroup!.id
            : firstTool.name;
          return {
            detail: resource.description,
            id: `tool:${resourceID}`,
            label: resource.label,
            reference: {
              id: resourceID,
              kind: "tool" as const,
              token: resource.label,
            },
            token: resource.label,
            type: "工具" as const,
          };
        }),
        ...extensions.filter((extension) => extension.enabled).map((extension) => {
          const token = extension.summary.name || extension.id;
          return { detail: extension.summary.description, id: `extension:${extension.id}`, label: token, reference: { id: extension.id, kind: "extension" as const, token }, token, type: "扩展" as const };
        }),
        ...servers.filter((item) => item.server.enabled).map((item) => {
          const label = item.server.displayName || item.server.name;
          return { detail: item.server.description, id: `mcp:${item.server.id}`, label, reference: { id: item.server.id, kind: "mcp" as const, token: label }, token: label, type: "MCP" as const };
        }),
      ]);
    });
    return () => { cancelled = true; };
  }, [projectPath]);

  const items = useMemo(() => filterPromptMentionItems([
    ...promptMentionProjectItems(projects, projectPath),
    ...catalogItems,
  ], query), [catalogItems, projectPath, projects, query]);
  const groups = useMemo(() => groupPromptMentionItems(items), [items]);
  const actions = useMemo(() => filterPromptMentionActions(query), [query]);
  const orderedItems = useMemo(
    () => [...actions, ...groups.flatMap((group) => group.items)],
    [actions, groups],
  );
  const listHeight = actions.length || groups.length
    ? Math.min(
        288,
        (actions.length ? 28 + actions.length * 40 : 0) + groups.reduce(
          (height, group) => height + 28 + group.items.length * 40,
          8,
        ),
      )
    : 64;

  useEffect(() => {
    const option = pickerRef.current
      ?.querySelectorAll<HTMLElement>('[role="option"]')
      .item(activeIndex);
    option?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, orderedItems]);

  return (
    <div aria-label="引用资源" className="absolute bottom-[calc(100%+0.25rem)] left-0 z-30 w-full overflow-hidden rounded-xl border bg-popover p-1" ref={pickerRef} role="listbox">
      <div className="px-2 py-1.5 text-xs text-muted-foreground">{query ? `模糊匹配 “${query}”` : "添加引用"}</div>
      <ScrollArea
        className="min-w-0 [&_[data-slot=scroll-area-scrollbar]]:my-1 [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0 [&_[data-slot=scroll-area-viewport]>div]:!w-full"
        style={{ height: listHeight }}
      >
        {actions.length ? (
          <section aria-label="本地" className="min-w-0 overflow-hidden border-t border-border/60 py-1 first:border-t-0" role="group">
            <div className="flex items-center justify-between px-2 py-1 text-[11px] font-medium text-muted-foreground">
              <span>本地</span>
              <span>{actions.length}</span>
            </div>
            {actions.map((item, index) => (
              <button aria-selected={activeIndex === index} className="flex w-full min-w-0 max-w-full items-center gap-2 overflow-hidden rounded-lg px-2 py-2 text-left text-sm outline-none hover:bg-muted aria-selected:bg-muted" key={item.id} onMouseDown={(event) => event.preventDefault()} onClick={() => onSelectAction(item)} role="option" type="button">
                <HugeiconsIcon
                  aria-hidden
                  className="size-4 shrink-0 text-muted-foreground"
                  data-icon="inline-start"
                  icon={item.action === "compact-context" ? ArrowShrink01Icon : Folder01Icon}
                  strokeWidth={1.8}
                />
                <span className="min-w-0 flex-1 truncate">{item.label}</span>
                <span className="min-w-0 max-w-[45%] shrink truncate text-xs text-muted-foreground">{item.detail}</span>
              </button>
            ))}
          </section>
        ) : null}
        {groups.map((group) => (
          <section aria-label={group.type} className="min-w-0 overflow-hidden border-t border-border/60 py-1 first:border-t-0" key={group.type} role="group">
            <div className="flex items-center justify-between px-2 py-1 text-[11px] font-medium text-muted-foreground">
              <span>{group.type}</span>
              <span>{group.items.length}</span>
            </div>
            {group.items.map((item) => {
              const index = actions.length + groups
                .flatMap((candidate) => candidate.items)
                .indexOf(item);
              return (
                <button aria-selected={activeIndex === index} className="flex w-full min-w-0 max-w-full items-center gap-2 overflow-hidden rounded-lg px-2 py-2 text-left text-sm outline-none hover:bg-muted aria-selected:bg-muted" key={item.id} onMouseDown={(event) => event.preventDefault()} onClick={() => onSelect(item)} role="option" type="button">
                  <PromptMentionIcon className="text-muted-foreground" kind={item.reference.kind} />
                  <span className="min-w-0 flex-1 truncate">{item.label}</span>
                  <span className="min-w-0 max-w-[45%] shrink truncate text-xs text-muted-foreground">{item.detail || item.type}</span>
                </button>
              );
            })}
          </section>
        ))}
        {!actions.length && !groups.length ? <div className="px-2 py-5 text-center text-sm text-muted-foreground">没有匹配的资源</div> : null}
      </ScrollArea>
    </div>
  );
}
