import { useEffect, useMemo, useRef, useState } from "react";

import { Folder01Icon, PaperclipIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { PromptMentionIcon } from "@/features/projects/project-prompt-mention-icon";
import {
  filterPromptMentionActions,
  filterPromptMentionItems,
  groupPromptMentionItems,
  promptMentionCodexSkillItems,
  promptMentionCodexMcpItems,
  promptMentionConversationItems,
  promptMentionProjectItems,
  PROMPT_MENTION_TYPE_LIMIT,
  type PromptMentionAction,
  type PromptMentionItem,
  type PromptMentionProject,
} from "@/features/projects/project-prompt-mention-model";
import { listCodexSessions } from "@/services/codex-thread-service";
import {
  listCodexSkills,
  listCodexMcpServers,
} from "@/services/codex-resource-service";

type ResourceState = {
  items: PromptMentionItem[];
  errors: string[];
  loading: boolean;
};

export function PromptMentionPicker({
  id,
  activeIndex,
  onActiveIndexChange,
  onSelect,
  onSelectAction,
  projectPath,
  projects,
  query,
}: {
  id: string;
  activeIndex: number;
  onActiveIndexChange: (index: number) => void;
  onSelect: (item: PromptMentionItem) => void;
  onSelectAction: (item: PromptMentionAction) => void;
  projectPath: string;
  projects: PromptMentionProject[];
  query: string;
}) {
  const [skills, setSkills] = useState<ResourceState>({
    items: [],
    errors: [],
    loading: true,
  });
  const [mcp, setMcp] = useState<ResourceState>({
    items: [],
    errors: [],
    loading: true,
  });
  const [reloadKey, setReloadKey] = useState(0);
  const [conversationResults, setConversationResults] = useState<{
    query: string;
    items: PromptMentionItem[];
  }>({ query: "", items: [] });
  const pickerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    setSkills({ items: [], errors: [], loading: true });
    setMcp({ items: [], errors: [], loading: true });
    const message = (error: unknown) =>
      error instanceof Error ? error.message : String(error);
    void listCodexSkills(projectPath || undefined, reloadKey > 0)
      .then((result) => {
        if (!cancelled)
          setSkills({
            items: promptMentionCodexSkillItems(result.skills),
            errors: result.errors,
            loading: false,
          });
      })
      .catch((error) => {
        if (!cancelled)
          setSkills({ items: [], errors: [message(error)], loading: false });
      });
    void listCodexMcpServers()
      .then((servers) => {
        if (!cancelled)
          setMcp({
            items: promptMentionCodexMcpItems(servers),
            errors: servers.flatMap((server) =>
              server.toolsError ? [`${server.name}: ${server.toolsError}`] : [],
            ),
            loading: false,
          });
      })
      .catch((error) => {
        if (!cancelled)
          setMcp({ items: [], errors: [message(error)], loading: false });
      });
    return () => {
      cancelled = true;
    };
  }, [projectPath, reloadKey]);

  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(
      () => {
        void listCodexSessions(
          PROMPT_MENTION_TYPE_LIMIT,
          query.trim() || undefined,
        )
          .then((conversations) => {
            if (!cancelled)
              setConversationResults({
                query,
                items: promptMentionConversationItems(conversations),
              });
          })
          .catch(() => {
            if (!cancelled) setConversationResults({ query, items: [] });
          });
      },
      query ? 150 : 0,
    );
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [query]);

  const items = useMemo(
    () =>
      filterPromptMentionItems(
        [
          ...promptMentionProjectItems(projects, projectPath),
          ...skills.items,
          ...mcp.items,
          ...(conversationResults.query === query
            ? conversationResults.items
            : []),
        ],
        query,
      ),
    [
      skills.items,
      mcp.items,
      conversationResults,
      projectPath,
      projects,
      query,
    ],
  );
  const groups = useMemo(() => groupPromptMentionItems(items), [items]);
  const actions = useMemo(
    () => filterPromptMentionActions(query).slice(0, PROMPT_MENTION_TYPE_LIMIT),
    [query],
  );
  const selectableActions = actions.filter((item) => !item.disabled);
  const orderedItems = useMemo(
    () => [...selectableActions, ...groups.flatMap((group) => group.items)],
    [groups, selectableActions],
  );
  const listHeight = Math.min(
    320,
    Math.max(
      72,
      24 +
        [skills, mcp].filter(
          (state) =>
            state.loading || state.errors.length || !state.items.length,
        ).length *
          24 +
        actions.length * 28 +
        groups.reduce(
          (height, group) => height + 32 + group.items.length * 28,
          0,
        ),
    ),
  );

  useEffect(() => {
    if (activeIndex >= orderedItems.length && activeIndex > 0) {
      onActiveIndexChange(Math.max(0, orderedItems.length - 1));
      return;
    }
    const option = pickerRef.current
      ?.querySelectorAll<HTMLElement>('[role="option"]:not([disabled])')
      .item(activeIndex);
    option?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, onActiveIndexChange, orderedItems]);

  return (
    <div
      id={id}
      aria-label="引用资源"
      className="absolute bottom-[calc(100%+1.25rem)] -left-5 z-30 max-h-[calc(100vh-5rem)] w-[calc(100%+2.5rem)] overflow-hidden rounded-xl border border-border/60 bg-popover p-1 shadow-sm"
      ref={pickerRef}
      role="listbox"
    >
      <ScrollArea
        type="always"
        className="min-w-0 pr-3 [&_[data-slot=scroll-area-scrollbar]]:w-2 [&_[data-slot=scroll-area-scrollbar]]:border-0 [&_[data-slot=scroll-area-thumb]]:bg-foreground/10 [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0 [&_[data-slot=scroll-area-viewport]>div]:!w-full"
        style={{ height: `min(${listHeight}px, calc(100vh - 12rem))` }}
      >
        <div className="flex h-[24px] items-center px-2 text-[10px] font-normal text-muted-foreground/80">
          {query ? `搜索 “${query}”` : "添加"}
        </div>
        {(
          [
            ["技能", skills],
            ["MCP", mcp],
          ] as const
        ).map(([name, state]) => {
          if (!state.loading && !state.errors.length && state.items.length)
            return null;
          return (
            <div
              key={name}
              role={state.errors.length ? "alert" : "status"}
              className="flex min-h-[24px] items-center gap-2 px-2 text-[10px] text-muted-foreground"
            >
              <span
                className="min-w-0 flex-1 truncate"
                title={state.errors.join("\n")}
              >
                {state.loading
                  ? `正在加载${name}…`
                  : state.errors.length
                    ? `${name}加载异常：${state.errors[0]}`
                    : name === "技能"
                      ? "当前项目未发现已启用的技能"
                      : "当前运行环境没有可用的 MCP 服务"}
              </span>
              {state.errors.length ? (
                <button
                  type="button"
                  className="shrink-0 underline"
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => setReloadKey((key) => key + 1)}
                >
                  重试
                </button>
              ) : null}
            </div>
          );
        })}
        {actions.map((item) => {
          const index = selectableActions.indexOf(item);
          return (
            <button
              aria-disabled={item.disabled || undefined}
              aria-selected={index >= 0 && activeIndex === index}
              className="flex h-[28px] w-full min-w-0 items-center gap-2 overflow-hidden rounded px-2 text-left text-[12px] font-normal outline-none hover:bg-foreground/5 aria-selected:bg-foreground/5 aria-disabled:cursor-not-allowed aria-disabled:opacity-45 aria-disabled:hover:bg-transparent"
              disabled={item.disabled}
              key={item.id}
              id={`${id}-option-${index}`}
              onMouseMove={() => {
                if (index >= 0) onActiveIndexChange(index);
              }}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => onSelectAction(item)}
              role="option"
              type="button"
            >
              <HugeiconsIcon
                aria-hidden
                className="size-4 shrink-0 text-muted-foreground"
                data-icon="inline-start"
                icon={
                  item.action === "select-local" ? PaperclipIcon : Folder01Icon
                }
                strokeWidth={1.8}
              />
              <span className="min-w-0 max-w-[60%] shrink-0 truncate">
                {item.label}
              </span>
              <span className="min-w-0 truncate text-[10px] text-muted-foreground/65">
                {item.detail}
              </span>
            </button>
          );
        })}
        {groups.map((group) => (
          <section
            aria-label={group.type}
            className="min-w-0 overflow-hidden pt-[8px]"
            key={group.type}
            role="group"
          >
            <div className="flex h-[24px] items-center px-2 text-[10px] font-normal text-muted-foreground/80">
              {group.type}
            </div>
            {group.items.map((item) => {
              const index =
                selectableActions.length +
                groups.flatMap((candidate) => candidate.items).indexOf(item);
              return (
                <button
                  id={`${id}-option-${index}`}
                  onMouseMove={() => onActiveIndexChange(index)}
                  aria-selected={activeIndex === index}
                  className="flex h-[28px] w-full min-w-0 items-center gap-2 overflow-hidden rounded px-2 text-left text-[12px] font-normal outline-none hover:bg-foreground/5 aria-selected:bg-foreground/5"
                  key={item.id}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => onSelect(item)}
                  role="option"
                  type="button"
                >
                  <PromptMentionIcon
                    className="size-4 text-muted-foreground"
                    kind={item.reference.kind}
                  />
                  <span className="min-w-0 max-w-[60%] shrink-0 truncate">
                    {item.label}
                  </span>
                  <span className="min-w-0 truncate text-[10px] text-muted-foreground/65">
                    {item.detail || item.type}
                  </span>
                </button>
              );
            })}
          </section>
        ))}
        {!actions.length &&
        !groups.length &&
        !skills.loading &&
        !mcp.loading ? (
          <div className="px-2 py-5 text-center text-sm text-muted-foreground">
            没有匹配的资源
          </div>
        ) : null}
      </ScrollArea>
    </div>
  );
}
