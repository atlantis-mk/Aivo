import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  getManagedSkillForEdit,
  updateManagedSkill,
  type SkillEditResult,
  type SkillEntry,
} from "@/services/aivo";

export function SkillEditorDialog({
  onOpenChange,
  onSaved,
  open,
  skill,
}: {
  onOpenChange: (open: boolean) => void;
  onSaved: (skill: SkillEntry) => Promise<void> | void;
  open: boolean;
  skill?: SkillEntry;
}) {
  const [editor, setEditor] = useState<SkillEditResult>();
  const [description, setDescription] = useState("");
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open || !skill) {
      setEditor(undefined);
      setDescription("");
      setContent("");
      setError("");
      return;
    }
    let active = true;
    setLoading(true);
    setError("");
    void getManagedSkillForEdit(skill.id)
      .then((result) => {
        if (!active) return;
        setEditor(result);
        setDescription(result.skill.description);
        setContent(result.content);
      })
      .catch((err) => {
        if (active) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [open, skill]);

  async function save() {
    if (!editor || !skill || !description.trim()) return;
    setSaving(true);
    setError("");
    try {
      const result = await updateManagedSkill({
        content,
        description: description.trim(),
        expectedContentHash: editor.skill.contentHash,
        skillId: skill.id,
      });
      await onSaved(result.skill);
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  const busy = loading || saving;

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="flex h-[min(86vh,46rem)] flex-col overflow-hidden sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>编辑技能</DialogTitle>
          <DialogDescription>
            保存后会更新 Aivo 管理的 SKILL.md；技能名称保持不变。
          </DialogDescription>
        </DialogHeader>

        <div className="grid min-h-0 flex-1 gap-4 overflow-y-auto pr-1">
          <div className="grid content-start gap-1.5">
            <Label htmlFor="skill-editor-name">技能名称</Label>
            <Input
              disabled
              id="skill-editor-name"
              value={editor?.skill.name ?? skill?.name ?? ""}
            />
          </div>
          <div className="grid content-start gap-1.5">
            <Label htmlFor="skill-editor-description">描述</Label>
            <Textarea
              aria-invalid={Boolean(editor && !description.trim())}
              className="min-h-20 resize-y"
              disabled={busy || !editor}
              id="skill-editor-description"
              maxLength={4096}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="说明该技能适合处理什么任务"
              value={description}
            />
          </div>
          <div className="grid min-h-0 content-stretch gap-1.5">
            <Label htmlFor="skill-editor-content">技能内容</Label>
            <Textarea
              className="min-h-72 resize-y font-mono"
              disabled={busy || !editor}
              id="skill-editor-content"
              onChange={(event) => setContent(event.target.value)}
              placeholder="输入技能执行说明"
              value={content}
            />
          </div>

          {error ? (
            <div
              className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive"
              role="alert"
            >
              {error}
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <Button
            disabled={saving}
            onClick={() => onOpenChange(false)}
            type="button"
            variant="outline"
          >
            取消
          </Button>
          <Button
            disabled={busy || !editor || !description.trim()}
            onClick={() => void save()}
            type="button"
          >
            {saving ? "保存中…" : loading ? "加载中…" : "保存"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
