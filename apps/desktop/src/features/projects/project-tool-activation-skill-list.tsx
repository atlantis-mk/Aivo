import { BookOpen } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Switch } from "@/components/ui/switch";
import type { SkillEntry, SkillImportCandidate } from "@/services/aivo";

export function SkillActivationList({
  activeSkillSet,
  candidates,
  disabled,
  onImportCandidate,
  onLoadSkill,
  onToggleSkill,
  skills,
}: {
  activeSkillSet: Set<string>;
  candidates: SkillImportCandidate[];
  disabled: boolean;
  onImportCandidate: (candidate: SkillImportCandidate) => void;
  onLoadSkill: (skill: SkillEntry, reload?: boolean) => void;
  onToggleSkill: (id: string, enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  const visibleCandidates = candidates.filter(
    (candidate) => candidate.status !== "imported",
  );
  if (skills.length === 0 && visibleCandidates.length === 0) {
    return (
      <Empty className="min-h-48 border">
        <EmptyMedia variant="icon">
          <BookOpen />
        </EmptyMedia>
        <EmptyHeader>
          <EmptyTitle>没有可用技能</EmptyTitle>
          <EmptyDescription>
            当前项目没有已导入技能或可迁移的兼容技能。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      {skills.length > 0 ? (
        <ImportedSkillSection
          activeSkillSet={activeSkillSet}
          disabled={disabled}
          onLoadSkill={onLoadSkill}
          onToggleSkill={onToggleSkill}
          skills={skills}
        />
      ) : null}

      {visibleCandidates.length > 0 ? (
        <SkillCandidateSection
          candidates={visibleCandidates}
          disabled={disabled}
          onImportCandidate={onImportCandidate}
        />
      ) : null}
    </div>
  );
}

function ImportedSkillSection({
  activeSkillSet,
  disabled,
  onLoadSkill,
  onToggleSkill,
  skills,
}: {
  activeSkillSet: Set<string>;
  disabled: boolean;
  onLoadSkill: (skill: SkillEntry, reload?: boolean) => void;
  onToggleSkill: (id: string, enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  return (
    <section className="flex min-w-0 flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1 truncate text-sm font-medium">
          已导入技能
        </div>
        <Badge variant="outline">{skills.length}</Badge>
      </div>
      <div className="overflow-hidden rounded-md border">
        {skills.map((skill) => {
          const active = activeSkillSet.has(skill.id);
          const switchId = `skill-activation-${encodeURIComponent(skill.id)}`;
          return (
            <div
              className="grid w-full min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 overflow-hidden border-b p-3 last:border-b-0"
              key={skill.id}
            >
              <label
                className="min-w-0 cursor-pointer overflow-hidden"
                htmlFor={switchId}
              >
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
                <div
                  className="mt-1 truncate text-xs text-muted-foreground"
                  title={skill.description}
                >
                  {skill.description}
                </div>
              </label>
              <div className="flex shrink-0 items-center gap-2">
                <Button
                  disabled={disabled}
                  onClick={() => onLoadSkill(skill, active)}
                  size="sm"
                  type="button"
                  variant="outline"
                >
                  {active ? "重载" : "加载"}
                </Button>
                <Switch
                  checked={active}
                  disabled={disabled}
                  id={switchId}
                  onCheckedChange={(checked) => onToggleSkill(skill.id, checked)}
                  size="sm"
                />
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function SkillCandidateSection({
  candidates,
  disabled,
  onImportCandidate,
}: {
  candidates: SkillImportCandidate[];
  disabled: boolean;
  onImportCandidate: (candidate: SkillImportCandidate) => void;
}) {
  return (
    <section className="flex min-w-0 flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1 truncate text-sm font-medium">
          可迁移技能
        </div>
        <Badge variant="outline">{candidates.length}</Badge>
      </div>
      <div className="overflow-hidden rounded-md border">
        {candidates.map((candidate) => (
          <div
            className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b p-3 last:border-b-0"
            key={candidate.id}
          >
            <div className="min-w-0 overflow-hidden">
              <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                <div className="truncate text-sm font-medium">
                  {candidate.name}
                </div>
                <Badge
                  variant={
                    candidate.status === "conflict" ? "destructive" : "outline"
                  }
                >
                  {candidate.status === "conflict"
                    ? "冲突"
                    : candidate.source}
                </Badge>
              </div>
              <div
                className="mt-1 truncate text-xs text-muted-foreground"
                title={candidate.skillPath}
              >
                {candidate.description || candidate.skillPath}
              </div>
            </div>
            <Button
              disabled={disabled || candidate.status === "conflict"}
              onClick={() => onImportCandidate(candidate)}
              size="sm"
              type="button"
              variant="outline"
            >
              导入
            </Button>
          </div>
        ))}
      </div>
    </section>
  );
}
