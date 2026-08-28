import {
  Add01Icon,
  BookOpen01Icon,
  Delete02Icon,
  File01Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Pencil } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/features/projects/extension-settings-empty-state";
import type { SkillEntry, SkillImportCandidate } from "@/services/aivo";

function SkillCardTitle({
  badge,
  icon,
  name,
}: {
  badge: string;
  icon: typeof BookOpen01Icon;
  name: string;
}) {
  return (
    <div className="flex min-w-0 items-center gap-3">
      <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-muted text-foreground">
        <HugeiconsIcon icon={icon} strokeWidth={1.8} />
      </div>
      <div className="flex min-w-0 items-center gap-2">
        <CardTitle className="truncate text-base" title={name}>
          {name}
        </CardTitle>
        <Badge className="shrink-0" variant="outline">
          {badge}
        </Badge>
      </div>
    </div>
  );
}

export function SkillManagementGroup({
  candidates,
  loading,
  onDelete,
  onEdit,
  onIgnore,
  onImport,
  onToggleEnabled,
  skills,
}: {
  candidates: SkillImportCandidate[];
  loading: boolean;
  onDelete: (skill: SkillEntry) => void;
  onEdit: (skill: SkillEntry) => void;
  onIgnore: (candidate: SkillImportCandidate) => void;
  onImport: (candidate: SkillImportCandidate) => void;
  onToggleEnabled: (skill: SkillEntry, enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  if (skills.length === 0 && candidates.length === 0) {
    return <EmptyState label="没有已导入技能或可导入候选" />;
  }

  return (
    <div className="flex flex-col gap-6">
      {skills.length > 0 ? (
        <section className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <div className="text-sm font-medium">Aivo 技能</div>
            <Badge variant="outline">{skills.length}</Badge>
          </div>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
            {skills.map((skill) => (
              <Card
                className="transition-colors hover:bg-muted/50 hover:ring-foreground/20"
                key={skill.id}
              >
                <CardHeader className="grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto] gap-x-3 gap-y-2">
                  <SkillCardTitle
                    badge={skill.scope === "project" ? "项目" : "全局"}
                    icon={BookOpen01Icon}
                    name={skill.name}
                  />
                  <CardAction className="row-span-1 flex items-center self-center gap-1">
                    <Switch
                      aria-label={`${skill.enabled ? "停用" : "启用"} ${skill.name}`}
                      checked={skill.enabled}
                      disabled={loading}
                      onCheckedChange={(checked) =>
                        onToggleEnabled(skill, checked)
                      }
                      size="sm"
                    />
                    <Button
                      aria-label={`编辑 ${skill.name}`}
                      disabled={loading}
                      onClick={() => onEdit(skill)}
                      size="icon-sm"
                      type="button"
                      variant="ghost"
                    >
                      <Pencil />
                    </Button>
                    <Button
                      aria-label={`删除 ${skill.name}`}
                      disabled={loading}
                      onClick={() => onDelete(skill)}
                      size="icon-sm"
                      type="button"
                      variant="ghost"
                    >
                      <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
                    </Button>
                  </CardAction>
                  <CardDescription
                    className="col-span-2 line-clamp-2 min-h-10 text-sm"
                    title={skill.description || skill.rootPath}
                  >
                    {skill.description || skill.rootPath}
                  </CardDescription>
                </CardHeader>
              </Card>
            ))}
          </div>
        </section>
      ) : null}

      {candidates.length > 0 ? (
        <section className="flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <div className="text-sm font-medium">兼容目录候选</div>
            <Badge variant="outline">{candidates.length}</Badge>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {candidates.map((candidate) => {
              const unavailable =
                candidate.status === "conflict" ||
                candidate.status === "ignored";
              const candidateBadge =
                candidate.status === "conflict"
                  ? "冲突"
                  : candidate.status === "ignored"
                    ? "已忽略"
                    : candidate.source;

              return (
                <Card
                  className="transition-colors hover:bg-muted/50 hover:ring-foreground/20 data-[ignored=true]:opacity-60"
                  data-ignored={candidate.status === "ignored"}
                  key={candidate.id}
                >
                  <CardHeader className="grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto] gap-x-3 gap-y-2">
                    <SkillCardTitle
                      badge={candidateBadge}
                      icon={File01Icon}
                      name={candidate.name}
                    />
                    <CardAction className="row-span-1 flex items-center self-center gap-1">
                      {candidate.status !== "ignored" ? (
                        <Button
                          disabled={loading}
                          onClick={() => onIgnore(candidate)}
                          size="sm"
                          type="button"
                          variant="ghost"
                        >
                          忽略
                        </Button>
                      ) : null}
                      <Button
                        aria-label={`导入 ${candidate.name}`}
                        disabled={loading || unavailable}
                        onClick={() => onImport(candidate)}
                        size="icon"
                        type="button"
                        variant="secondary"
                      >
                        <HugeiconsIcon icon={Add01Icon} strokeWidth={2} />
                      </Button>
                    </CardAction>
                    <CardDescription
                      className="col-span-2 line-clamp-2 min-h-10 text-sm"
                      title={candidate.description || candidate.skillPath}
                    >
                      {candidate.description || candidate.skillPath}
                    </CardDescription>
                  </CardHeader>
                </Card>
              );
            })}
          </div>
        </section>
      ) : null}
    </div>
  );
}
