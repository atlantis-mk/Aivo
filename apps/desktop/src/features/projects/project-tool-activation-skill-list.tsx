import { BookOpen } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Switch } from "@/components/ui/switch";
import type { SkillEntry } from "@/services/aivo";

export function SkillActivationList({
  activeSkillSet,
  disabled,
  onToggleSkill,
  skills,
}: {
  activeSkillSet: Set<string>;
  disabled: boolean;
  onToggleSkill: (id: string, enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  if (skills.length === 0) {
    return (
      <Empty className="min-h-48 border">
        <EmptyMedia variant="icon">
          <BookOpen />
        </EmptyMedia>
        <EmptyHeader>
          <EmptyTitle>没有可用技能</EmptyTitle>
          <EmptyDescription>还没有已导入技能。</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <section className="flex min-w-0 flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1 truncate text-sm font-medium">技能</div>
        <Badge variant="outline">{skills.length}</Badge>
      </div>

      <div className="overflow-hidden rounded-md border">
        {skills.map((skill) => {
          const active = activeSkillSet.has(skill.id);
          const switchId = `skill-activation-${encodeURIComponent(skill.id)}`;
          return (
            <label
              className="grid w-full min-w-0 cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-3 overflow-hidden border-b p-3 transition-colors last:border-b-0 hover:bg-muted/50"
              htmlFor={switchId}
              key={skill.id}
            >
              <div className="min-w-0 flex-1 overflow-hidden">
                <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                  <div className="truncate text-sm font-medium">
                    {skill.name}
                  </div>
                  <Badge
                    variant={skill.scope === "project" ? "secondary" : "outline"}
                  >
                    {skill.scope === "project" ? "项目" : "全局"}
                  </Badge>
                </div>
                {skill.description ? (
                  <div
                    className="mt-1 truncate text-xs text-muted-foreground"
                    title={skill.description}
                  >
                    {skill.description}
                  </div>
                ) : null}
              </div>
              <Switch
                checked={active}
                className="shrink-0"
                disabled={disabled}
                id={switchId}
                onCheckedChange={(checked) => onToggleSkill(skill.id, checked)}
                size="sm"
              />
            </label>
          );
        })}
      </div>
    </section>
  );
}
