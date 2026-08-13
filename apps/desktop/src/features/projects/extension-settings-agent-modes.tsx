import { useEffect, useMemo, useState } from "react";
import { Bot, Pencil, RotateCcw, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import {
  agentModeModelsForProvider,
  agentModeSubagentCandidates,
  connectedAgentModeProviders,
} from "@/features/projects/extension-settings-agent-mode-model";
import { EmptyState } from "@/features/projects/extension-settings-empty-state";
import type { CatalogState } from "@/lib/provider-catalog";
import type { AgentModeDefinition } from "@/services/aivo";

type AgentModeDraft = {
  description: string;
  displayName: string;
  id: string;
  maxSteps: string;
  mode: "primary" | "subagent" | "all";
  modelId: string;
  options: string;
  permissionScope: string;
  prompt: string;
  providerId: string;
  subagents: string[];
  temperature: string;
  topP: string;
  variant: string;
};

const emptyAgentModeDraft: AgentModeDraft = {
  description: "",
  displayName: "",
  id: "",
  maxSteps: "",
  mode: "all",
  modelId: "",
  options: "{}",
  permissionScope: "",
  prompt: "",
  providerId: "",
  subagents: [],
  temperature: "",
  topP: "",
  variant: "",
};

export function AgentModeManagementGroup({
  disabled,
  modes,
  onEdit,
}: {
  disabled: boolean;
  modes: AgentModeDefinition[];
  onEdit: (mode: AgentModeDefinition) => void;
}) {
  if (modes.length === 0) {
    return <EmptyState label="没有匹配的 Agent 模式" />;
  }

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
      {modes.map((mode) => (
        <Card
          className="transition-colors hover:bg-muted/50 hover:ring-foreground/20"
          key={mode.id}
        >
          <CardHeader className="grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_auto] gap-x-3 gap-y-2">
            <div className="flex min-w-0 items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-muted text-foreground">
                <Bot className="size-5" />
              </div>
              <div className="min-w-0">
                <div className="flex min-w-0 items-center gap-2">
                  <CardTitle className="truncate text-base" title={mode.displayName}>
                    {mode.displayName}
                  </CardTitle>
                  <Badge className="shrink-0" variant="outline">
                    {mode.builtIn ? (mode.overridden ? "内置·已修改" : "内置") : "自定义"}
                  </Badge>
                </div>
                <div className="mt-1 truncate font-mono text-[11px] text-muted-foreground">
                  {mode.id}
                </div>
              </div>
            </div>
            <CardAction className="row-span-1 self-center">
              <Button
                aria-label={`编辑 ${mode.displayName}`}
                disabled={disabled}
                onClick={() => onEdit(mode)}
                size="icon-sm"
                type="button"
                variant="ghost"
              >
                <Pencil />
              </Button>
            </CardAction>
            <CardDescription
              className="col-span-2 line-clamp-2 min-h-10 text-sm"
              title={mode.description || mode.prompt}
            >
              {mode.description || mode.prompt}
            </CardDescription>
            <div className="col-span-2 flex flex-wrap gap-1.5">
              <Badge variant="secondary">{agentRoleLabel(mode.mode)}</Badge>
              {mode.permissionScope ? (
                <Badge variant="secondary">{mode.permissionScope}</Badge>
              ) : null}
              {mode.subagents?.length ? (
                <Badge variant="secondary">{mode.subagents.length} 个子 Agent</Badge>
              ) : null}
            </div>
          </CardHeader>
        </Card>
      ))}
    </div>
  );
}

export function AgentModeEditorDialog({
  disabled,
  mode,
  modes,
  onDelete,
  onOpenChange,
  onSave,
  open,
  providerCatalog,
}: {
  disabled: boolean;
  mode?: AgentModeDefinition;
  modes: AgentModeDefinition[];
  onDelete: (mode: AgentModeDefinition) => Promise<void>;
  onOpenChange: (open: boolean) => void;
  onSave: (mode: AgentModeDefinition) => Promise<void>;
  open: boolean;
  providerCatalog: CatalogState | null;
}) {
  const [draft, setDraft] = useState<AgentModeDraft>(emptyAgentModeDraft);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setDraft(mode ? draftFromAgentMode(mode) : emptyAgentModeDraft);
    setError("");
  }, [mode, open]);

  const connectedProviders = useMemo(
    () => connectedAgentModeProviders(providerCatalog),
    [providerCatalog],
  );
  const modelOptions = useMemo(
    () => agentModeModelsForProvider(providerCatalog, draft.providerId),
    [draft.providerId, providerCatalog],
  );
  const subagentCandidates = useMemo(
    () => agentModeSubagentCandidates(modes, draft.id),
    [draft.id, modes],
  );
  const selectedProviderAvailable = connectedProviders.some(
    (provider) => provider.id === draft.providerId,
  );
  const selectedModelAvailable = modelOptions.some(
    (modelOption) => modelOption.id === draft.modelId,
  );
  const canSave = useMemo(
    () =>
      Boolean(
        draft.id.trim() &&
          draft.displayName.trim() &&
          draft.prompt.trim() &&
          (draft.providerId ? draft.modelId : !draft.modelId),
      ),
    [draft.displayName, draft.id, draft.modelId, draft.prompt, draft.providerId],
  );

  async function save() {
    setError("");
    try {
      await onSave(agentModeFromDraft(draft, mode));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function remove() {
    if (!mode) return;
    const action = mode.builtIn ? "重置为内置默认值" : "删除此 Agent 模式";
    if (!globalThis.confirm(`${action}？`)) return;
    setError("");
    try {
      await onDelete(mode);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const deleteLabel = mode?.builtIn ? "重置" : "删除";
  const canDelete = Boolean(mode && (!mode.builtIn || mode.overridden));

  return (
    <DialogContent className="flex h-[min(90vh,50rem)] flex-col overflow-hidden sm:max-w-3xl">
      <DialogHeader>
        <DialogTitle>{mode ? `编辑 ${mode.displayName}` : "添加 Agent 模式"}</DialogTitle>
        <DialogDescription>
          模式定义由 Core 校验并应用到后续会话。项目内的 .aivo 配置仍可覆盖全局定义。
        </DialogDescription>
      </DialogHeader>
      <div className="relative min-h-0 flex-1">
        <ScrollArea className="h-full pr-3">
          <div className="grid gap-4 pb-1">
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="标识符">
              <Input
                disabled={Boolean(mode)}
                onChange={(event) => updateDraft(setDraft, "id", event.target.value.toLowerCase())}
                placeholder="research"
                value={draft.id}
              />
            </Field>
            <Field label="显示名称">
              <Input
                onChange={(event) => updateDraft(setDraft, "displayName", event.target.value)}
                placeholder="Research"
                value={draft.displayName}
              />
            </Field>
          </div>

          <Field label="描述">
            <Input
              onChange={(event) => updateDraft(setDraft, "description", event.target.value)}
              placeholder="说明该模式适合什么任务"
              value={draft.description}
            />
          </Field>

          <Field label="系统提示词">
            <Textarea
              className="min-h-40 resize-y"
              onChange={(event) => updateDraft(setDraft, "prompt", event.target.value)}
              placeholder="定义 Agent 的工作方式和边界"
              value={draft.prompt}
            />
          </Field>

          <div className="grid gap-3 sm:grid-cols-3">
            <Field label="角色">
              <NativeSelect
                className="w-full"
                onChange={(event) =>
                  setDraft((current) => {
                    const nextMode = event.target.value as AgentModeDraft["mode"];
                    return {
                      ...current,
                      mode: nextMode,
                      subagents: nextMode === "subagent" ? [] : current.subagents,
                    };
                  })
                }
                value={draft.mode}
              >
                <NativeSelectOption value="all">主 Agent / 子 Agent</NativeSelectOption>
                <NativeSelectOption value="primary">仅主 Agent</NativeSelectOption>
                <NativeSelectOption disabled={Boolean(mode?.builtIn)} value="subagent">
                  仅子 Agent
                </NativeSelectOption>
              </NativeSelect>
            </Field>
            <Field label="权限范围">
              <NativeSelect
                className="w-full"
                onChange={(event) => updateDraft(setDraft, "permissionScope", event.target.value)}
                value={draft.permissionScope}
              >
                <NativeSelectOption value="">跟随会话</NativeSelectOption>
                <NativeSelectOption value="workspace">工作区</NativeSelectOption>
                <NativeSelectOption value="workspace_approval">工作区审批</NativeSelectOption>
                <NativeSelectOption value="read_only">只读</NativeSelectOption>
                <NativeSelectOption value="no_shell">禁止 Shell</NativeSelectOption>
              </NativeSelect>
            </Field>
            <Field label="最大步骤">
              <Input
                max={100}
                min={0}
                onChange={(event) => updateDraft(setDraft, "maxSteps", event.target.value)}
                placeholder="默认"
                type="number"
                value={draft.maxSteps}
              />
            </Field>
          </div>

          {draft.mode !== "subagent" ? (
            <div className="grid gap-1.5">
              <div>
                <Label>关联子 Agent</Label>
                <p className="mt-1 text-xs text-muted-foreground" id="agent-subagents-help">
                  模型会按任务自行判断是否委派，只能调用这里选中的子 Agent。
                </p>
              </div>
              <ScrollArea
                aria-describedby="agent-subagents-help"
                className="h-44 rounded-lg border bg-muted/20"
              >
                {subagentCandidates.length ? (
                  <div className="grid gap-1 p-2 pr-3">
                    {subagentCandidates.map((candidate, index) => {
                      const checkboxId = `agent-subagent-${index}`;
                      return (
                        <label
                          className="flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 transition-colors hover:bg-muted"
                          htmlFor={checkboxId}
                          key={candidate.id}
                        >
                          <Checkbox
                            checked={draft.subagents.includes(candidate.id)}
                            className="mt-0.5"
                            disabled={disabled}
                            id={checkboxId}
                            onCheckedChange={(checked) =>
                              setDraft((current) => ({
                                ...current,
                                subagents:
                                  checked === true
                                    ? [...current.subagents, candidate.id]
                                    : current.subagents.filter((id) => id !== candidate.id),
                              }))
                            }
                          />
                          <span className="min-w-0 flex-1">
                            <span className="flex items-center gap-2 text-sm font-medium">
                              <span className="truncate">{candidate.displayName}</span>
                              <span className="shrink-0 font-mono text-[11px] font-normal text-muted-foreground">
                                {candidate.id}
                              </span>
                            </span>
                            <span className="mt-0.5 line-clamp-2 block text-xs text-muted-foreground">
                              {candidate.description || candidate.prompt}
                            </span>
                          </span>
                        </label>
                      );
                    })}
                  </div>
                ) : (
                  <div className="flex h-full items-center justify-center px-4 text-center text-sm text-muted-foreground">
                    暂无可关联的子 Agent 模式
                  </div>
                )}
              </ScrollArea>
            </div>
          ) : (
            <p className="rounded-lg border bg-muted/20 p-3 text-sm text-muted-foreground">
              仅子 Agent 模式不能继续关联其他子 Agent。
            </p>
          )}

          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Provider">
              <NativeSelect
                aria-label="Provider"
                className="w-full"
                disabled={disabled}
                onChange={(event) => {
                  const providerId = event.target.value;
                  setDraft((current) => ({
                    ...current,
                    modelId: "",
                    providerId,
                  }));
                }}
                value={draft.providerId}
              >
                <NativeSelectOption value="">跟随会话</NativeSelectOption>
                {draft.providerId && !selectedProviderAvailable ? (
                  <NativeSelectOption disabled value={draft.providerId}>
                    {draft.providerId}（未连接）
                  </NativeSelectOption>
                ) : null}
                {connectedProviders.map((provider) => (
                  <NativeSelectOption key={provider.id} value={provider.id}>
                    {providerOptionLabel(provider.name, provider.id)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field label="模型">
              <NativeSelect
                aria-label="模型"
                className="w-full"
                disabled={disabled || !selectedProviderAvailable}
                onChange={(event) => updateDraft(setDraft, "modelId", event.target.value)}
                value={draft.modelId}
              >
                <NativeSelectOption value="">
                  {draft.providerId ? "选择模型" : "请先选择 Provider"}
                </NativeSelectOption>
                {draft.modelId && !selectedModelAvailable ? (
                  <NativeSelectOption disabled value={draft.modelId}>
                    {draft.modelId}（不可用）
                  </NativeSelectOption>
                ) : null}
                {modelOptions.map((modelOption) => (
                  <NativeSelectOption key={modelOption.id} value={modelOption.id}>
                    {modelOptionLabel(modelOption.name, modelOption.id)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <Field label="Temperature">
              <Input
                max={2}
                min={0}
                onChange={(event) => updateDraft(setDraft, "temperature", event.target.value)}
                placeholder="默认"
                step="0.1"
                type="number"
                value={draft.temperature}
              />
            </Field>
            <Field label="Top P">
              <Input
                max={1}
                min={0}
                onChange={(event) => updateDraft(setDraft, "topP", event.target.value)}
                placeholder="默认"
                step="0.1"
                type="number"
                value={draft.topP}
              />
            </Field>
            <Field label="模型变体">
              <Input
                onChange={(event) => updateDraft(setDraft, "variant", event.target.value)}
                placeholder="可选"
                value={draft.variant}
              />
            </Field>
          </div>

          <Field label="Provider 选项（JSON）">
            <Textarea
              className="min-h-24 resize-y font-mono"
              onChange={(event) => updateDraft(setDraft, "options", event.target.value)}
              value={draft.options}
            />
          </Field>

          {error ? (
            <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          ) : null}
          </div>
        </ScrollArea>
      </div>
      <DialogFooter className="sm:justify-between">
        <div>
          {canDelete ? (
            <Button disabled={disabled} onClick={() => void remove()} type="button" variant="destructive">
              {mode?.builtIn ? <RotateCcw /> : <Trash2 />}
              {deleteLabel}
            </Button>
          ) : null}
        </div>
        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Button disabled={disabled} onClick={() => onOpenChange(false)} type="button" variant="outline">
            取消
          </Button>
          <Button disabled={disabled || !canSave} onClick={() => void save()} type="button">
            {disabled ? "保存中" : "保存"}
          </Button>
        </div>
      </DialogFooter>
    </DialogContent>
  );
}

function Field({ children, label }: { children: React.ReactNode; label: string }) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function draftFromAgentMode(mode: AgentModeDefinition): AgentModeDraft {
  return {
    description: mode.description ?? "",
    displayName: mode.displayName,
    id: mode.id,
    maxSteps: mode.maxSteps ? String(mode.maxSteps) : "",
    mode: mode.mode ?? "all",
    modelId: mode.model?.modelId ?? "",
    options: JSON.stringify(mode.options ?? {}, null, 2),
    permissionScope: mode.permissionScope ?? "",
    prompt: mode.prompt,
    providerId: mode.model?.providerId ?? "",
    subagents: [...(mode.subagents ?? [])],
    temperature: mode.temperature == null ? "" : String(mode.temperature),
    topP: mode.topP == null ? "" : String(mode.topP),
    variant: mode.variant ?? "",
  };
}

function agentModeFromDraft(draft: AgentModeDraft, original?: AgentModeDefinition): AgentModeDefinition {
  let options: Record<string, unknown> | undefined;
  const optionsText = draft.options.trim();
  if (optionsText && optionsText !== "{}") {
    const parsed = JSON.parse(optionsText) as unknown;
    if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
      throw new Error("Provider 选项必须是 JSON 对象");
    }
    options = parsed as Record<string, unknown>;
  }
  const providerId = draft.providerId.trim();
  const modelId = draft.modelId.trim();
  if (Boolean(providerId) !== Boolean(modelId)) {
    throw new Error("Provider ID 和模型 ID 必须同时填写");
  }
  return {
    ...original,
    builtIn: undefined,
    description: draft.description.trim(),
    displayName: draft.displayName.trim(),
    hidden: false,
    id: draft.id.trim(),
    maxSteps: optionalNumber(draft.maxSteps),
    mode: draft.mode,
    model: providerId && modelId ? { providerId, modelId } : undefined,
    options,
    overridden: undefined,
    permissionScope: draft.permissionScope,
    prompt: draft.prompt.trim(),
    revision: undefined,
    source: undefined,
    subagents: draft.subagents,
    temperature: optionalNumber(draft.temperature),
    topP: optionalNumber(draft.topP),
    variant: draft.variant.trim(),
  };
}

function optionalNumber(value: string) {
  if (!value.trim()) return undefined;
  const number = Number(value);
  if (!Number.isFinite(number)) throw new Error("数值字段格式无效");
  return number;
}

function updateDraft<K extends keyof AgentModeDraft>(
  setDraft: React.Dispatch<React.SetStateAction<AgentModeDraft>>,
  key: K,
  value: AgentModeDraft[K],
) {
  setDraft((current) => ({ ...current, [key]: value }));
}

function agentRoleLabel(mode?: AgentModeDefinition["mode"]) {
  if (mode === "primary") return "仅主 Agent";
  if (mode === "subagent") return "仅子 Agent";
  return "主 / 子 Agent";
}

function providerOptionLabel(name: string, id: string) {
  return name && name !== id ? `${name} (${id})` : id;
}

function modelOptionLabel(name: string, id: string) {
  return name && name !== id ? `${name} (${id})` : id;
}
