import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  SkillActivationList,
  ToolActivationDialogSkeleton,
  ToolActivationToolList,
} from "@/features/projects/project-tool-activation-lists";
import type {
  SkillCatalogGroup,
  ToolCatalogGroup,
} from "@/features/projects/project-tool-activation-model";

export function ToolActivationDialogTabs({
  activeSkillSet,
  activeToolSet,
  disabled,
  groupedSkills,
  groupedTools,
  loading,
  onToggleSkill,
  onToggleToolGroup,
  toggleableToolCount,
  usedToolSet,
}: {
  activeSkillSet: Set<string>;
  activeToolSet: Set<string>;
  disabled: boolean;
  groupedSkills: SkillCatalogGroup[];
  groupedTools: ToolCatalogGroup[];
  loading: boolean;
  onToggleSkill: (ids: string[], enabled: boolean) => void;
  onToggleToolGroup: (names: string[], enabled: boolean) => void;
  toggleableToolCount: number;
  usedToolSet: Set<string>;
}) {
  return (
    <Tabs
      key={loading ? "loading" : "ready"}
      defaultValue={defaultActivationSection(groupedTools, groupedSkills.length)}
      className="flex min-h-0 flex-1 flex-col gap-0"
    >
      <ToolActivationDialogStatusBar
        groupedTools={groupedTools}
        skillCount={groupedSkills.length}
        toggleableToolCount={toggleableToolCount}
      />

      <Separator />

      {(["extensions", "mcp", "tools"] as const).map((section) => (
        <TabsContent
          className="min-h-0 flex-1 p-0"
          key={section}
          value={section}
        >
          <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
            <div className="px-5 py-4">
              {loading ? (
                <ToolActivationDialogSkeleton />
              ) : (
                <ToolActivationToolList
                  activeToolSet={activeToolSet}
                  disabled={disabled}
                  groupedTools={groupedTools.filter(
                    (group) => group.section === section,
                  )}
                  onToggleToolGroup={onToggleToolGroup}
                  usedToolSet={usedToolSet}
                />
              )}
            </div>
          </ScrollArea>
        </TabsContent>
      ))}

      <TabsContent className="min-h-0 flex-1 p-0" value="skills">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
          <div className="px-5 py-4">
            {loading ? (
              <ToolActivationDialogSkeleton />
            ) : (
              <SkillActivationList
                activeSkillSet={activeSkillSet}
                disabled={disabled}
                onToggleSkill={onToggleSkill}
                skills={groupedSkills.flatMap((group) => group.skills)}
              />
            )}
          </div>
        </ScrollArea>
      </TabsContent>
    </Tabs>
  );
}

export function ToolActivationDialogFooter({
  hasDraftChanges,
  loading,
  onSubmit,
  saving,
}: {
  hasDraftChanges: boolean;
  loading: boolean;
  onSubmit: () => void;
  saving: boolean;
}) {
  return (
    <DialogFooter className="border-t px-5 py-4 sm:items-center sm:justify-between">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        {loading || saving ? <Spinner /> : null}
        <span>
          {hasDraftChanges
            ? "更改将在提交后用于当前对话和新对话。"
            : "默认激活列表会用于新对话。"}
        </span>
      </div>
      <div className="flex flex-col-reverse gap-2 sm:flex-row">
        <DialogClose asChild>
          <Button disabled={saving} type="button" variant="outline">
            关闭
          </Button>
        </DialogClose>
        <Button
          disabled={loading || saving || !hasDraftChanges}
          onClick={onSubmit}
          type="button"
        >
          {saving ? "提交中" : "提交"}
        </Button>
      </div>
    </DialogFooter>
  );
}

function ToolActivationDialogStatusBar({
  groupedTools,
  skillCount,
  toggleableToolCount,
}: {
  groupedTools: ToolCatalogGroup[];
  skillCount: number;
  toggleableToolCount: number;
}) {
  const count = (section: ToolCatalogGroup["section"]) =>
    groupedTools.filter((group) => group.section === section).length;
  return (
    <div className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <TabsList>
        <TabsTrigger value="extensions">扩展 {count("extensions")}</TabsTrigger>
        <TabsTrigger value="mcp">MCP {count("mcp")}</TabsTrigger>
        <TabsTrigger value="skills">技能 {skillCount}</TabsTrigger>
        <TabsTrigger value="tools">工具 {count("tools")}</TabsTrigger>
      </TabsList>
      <div className="text-xs text-muted-foreground">
        {toggleableToolCount} 个可配置项
      </div>
    </div>
  );
}

function defaultActivationSection(
  groupedTools: ToolCatalogGroup[],
  skillCount: number,
) {
  for (const section of ["extensions", "mcp", "skills", "tools"] as const) {
    if (section === "skills") {
      if (skillCount > 0) return section;
      continue;
    }
    if (groupedTools.some((group) => group.section === section)) return section;
  }
  return "tools";
}
