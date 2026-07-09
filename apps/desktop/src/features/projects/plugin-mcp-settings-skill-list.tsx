import { BookOpen, FileText, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/features/projects/plugin-mcp-settings-empty-state";
import type { SkillEntry, SkillImportCandidate } from "@/services/aivo";

export function SkillManagementGroup({
  candidates,
  loading,
  onDelete,
  onImport,
  onToggleEnabled,
  skills,
}: {
  candidates: SkillImportCandidate[];
  loading: boolean;
  onDelete: (skill: SkillEntry) => void;
  onImport: (candidate: SkillImportCandidate) => void;
  onToggleEnabled: (skill: SkillEntry, enabled: boolean) => void;
  skills: SkillEntry[];
}) {
  const visibleCandidates = candidates.filter(
    (candidate) => candidate.status !== "imported",
  );
  if (skills.length === 0 && visibleCandidates.length === 0) {
    return <EmptyState label="没有已导入技能或可导入候选" />;
  }
  return (
    <div className="flex flex-col gap-4">
      {skills.length > 0 ? (
        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium">Aivo 技能</div>
            <Badge variant="outline">{skills.length}</Badge>
          </div>
          <ItemGroup>
            {skills.map((skill) => (
              <Item key={skill.id}>
                <ItemMedia variant="icon">
                  <BookOpen />
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>{skill.name}</ItemTitle>
                  <ItemDescription>
                    {skill.description || skill.rootPath}
                  </ItemDescription>
                </ItemContent>
                <ItemActions>
                  <Badge
                    variant={skill.scope === "project" ? "secondary" : "outline"}
                  >
                    {skill.scope === "project" ? "项目" : "全局"}
                  </Badge>
                  <Switch
                    checked={skill.enabled}
                    disabled={loading}
                    onCheckedChange={(checked) =>
                      onToggleEnabled(skill, checked)
                    }
                    size="sm"
                  />
                  <Button
                    aria-label={`删除 ${skill.name}`}
                    disabled={loading}
                    onClick={() => onDelete(skill)}
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  >
                    <Trash2 />
                  </Button>
                </ItemActions>
              </Item>
            ))}
          </ItemGroup>
        </section>
      ) : null}

      {visibleCandidates.length > 0 ? (
        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium">兼容目录候选</div>
            <Badge variant="outline">{visibleCandidates.length}</Badge>
          </div>
          <ItemGroup>
            {visibleCandidates.map((candidate) => (
              <Item key={candidate.id}>
                <ItemMedia variant="icon">
                  <FileText />
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>{candidate.name}</ItemTitle>
                  <ItemDescription>
                    {candidate.description || candidate.skillPath}
                  </ItemDescription>
                </ItemContent>
                <ItemActions>
                  <Badge
                    variant={
                      candidate.status === "conflict"
                        ? "destructive"
                        : "outline"
                    }
                  >
                    {candidate.status === "conflict"
                      ? "冲突"
                      : candidate.source}
                  </Badge>
                  <Button
                    disabled={loading || candidate.status === "conflict"}
                    onClick={() => onImport(candidate)}
                    size="sm"
                    type="button"
                    variant="outline"
                  >
                    导入
                  </Button>
                </ItemActions>
              </Item>
            ))}
          </ItemGroup>
        </section>
      ) : null}
    </div>
  );
}
