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
import type { ToolCatalogGroup } from "@/features/projects/project-tool-activation-model";
import type { SkillEntry } from "@/services/aivo";

export function ToolActivationDialogTabs({
  activeSkillSet,
  activeToolSet,
  disabled,
  groupedTools,
  loading,
  onToggleSkill,
  onToggleTool,
  skills,
  toggleableToolCount,
  usedToolSet,
}: {
  activeSkillSet: Set<string>;
  activeToolSet: Set<string>;
  disabled: boolean;
  groupedTools: ToolCatalogGroup[];
  loading: boolean;
  onToggleSkill: (id: string, enabled: boolean) => void;
  onToggleTool: (name: string, enabled: boolean) => void;
  skills: SkillEntry[];
  toggleableToolCount: number;
  usedToolSet: Set<string>;
}) {
  return (
    <Tabs defaultValue="tools" className="flex min-h-0 flex-1 flex-col gap-0">
      <ToolActivationDialogStatusBar toggleableToolCount={toggleableToolCount} />

      <Separator />

      <TabsContent className="min-h-0 flex-1 p-0" value="tools">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]]:overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!min-w-0">
          <div className="px-5 py-4">
            {loading ? (
              <ToolActivationDialogSkeleton />
            ) : (
              <ToolActivationToolList
                activeToolSet={activeToolSet}
                disabled={disabled}
                groupedTools={groupedTools}
                onToggleTool={onToggleTool}
                usedToolSet={usedToolSet}
              />
            )}
          </div>
        </ScrollArea>
      </TabsContent>

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
                skills={skills}
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
  toggleableToolCount,
}: {
  toggleableToolCount: number;
}) {
  return (
    <div className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <TabsList>
        <TabsTrigger value="tools">工具</TabsTrigger>
        <TabsTrigger value="skills">技能</TabsTrigger>
      </TabsList>
      <div className="text-xs text-muted-foreground">
        {toggleableToolCount} 个可配置工具
      </div>
    </div>
  );
}
