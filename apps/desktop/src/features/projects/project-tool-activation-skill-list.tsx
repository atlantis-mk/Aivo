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
import {
  groupSkillCatalogEntries,
  isSkillCatalogGroupActive,
  isSkillCatalogGroupPartiallyActive,
  type SkillCatalogGroup,
} from "@/features/projects/project-tool-activation-model";
import { skillCanActivate } from "@/features/projects/skill-action-model";
import type { SkillEntry } from "@/services/aivo";

export function SkillActivationList({
  activeSkillSet,
  disabled,
  onToggleSkill,
  skills,
}: {
  activeSkillSet: Set<string>;
  disabled: boolean;
  onToggleSkill: (ids: string[], enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  const activatableSkills = skills.filter(skillCanActivate);
  const groupedSkills = groupSkillCatalogEntries(activatableSkills);

  if (groupedSkills.length === 0) {
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
        <Badge variant="outline">{groupedSkills.length}</Badge>
      </div>

      <div className="overflow-hidden rounded-md border">
        {groupedSkills.map((group) => (
          <SkillActivationRow
            activeSkillSet={activeSkillSet}
            disabled={disabled}
            group={group}
            key={group.id}
            onToggleSkill={onToggleSkill}
          />
        ))}
      </div>
    </section>
  );
}

function SkillActivationRow({
  activeSkillSet,
  disabled,
  group,
  onToggleSkill,
}: {
  activeSkillSet: Set<string>;
  disabled: boolean;
  group: SkillCatalogGroup;
  onToggleSkill: (ids: string[], enabled: boolean) => void;
}) {
  const active = isSkillCatalogGroupActive(group, activeSkillSet);
  const partial = isSkillCatalogGroupPartiallyActive(group, activeSkillSet);
  const switchId = `skill-activation-${encodeURIComponent(group.id)}`;
  const memberNames = group.skills.map((skill) => skill.name);
  const firstSkill = group.skills[0];
  const scopeBadge = group.grouped
    ? `${group.skills.length} 个技能`
    : firstSkill?.scope === "project"
      ? "项目"
      : "全局";

  return (
    <label
      className="grid w-full min-w-0 cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-3 overflow-hidden border-b p-3 transition-colors last:border-b-0 hover:bg-muted/50"
      htmlFor={switchId}
    >
      <div className="min-w-0 flex-1 overflow-hidden">
        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
          <div className="truncate text-sm font-medium" title={group.label}>
            {group.label}
          </div>
          <Badge variant={group.grouped ? "secondary" : "outline"}>
            {scopeBadge}
          </Badge>
        </div>
        {group.description ? (
          <div
            className="mt-1 truncate text-xs text-muted-foreground"
            title={group.description}
          >
            {group.description}
          </div>
        ) : null}
        {group.grouped ? (
          <div
            className="mt-2 flex min-w-0 flex-wrap gap-1"
            title={memberNames.join(", ")}
          >
            {memberNames.map((name) => (
              <Badge className="max-w-full truncate" key={name} variant="outline">
                {name}
              </Badge>
            ))}
          </div>
        ) : null}
      </div>
      <Switch
        aria-checked={partial ? "mixed" : active}
        checked={active}
        className="shrink-0"
        disabled={disabled}
        id={switchId}
        onCheckedChange={(checked) =>
          onToggleSkill(group.skills.map((skill) => skill.id), checked)
        }
        size="sm"
      />
    </label>
  );
}
