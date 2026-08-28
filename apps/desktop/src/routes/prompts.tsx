/* eslint-disable react-refresh/only-export-components */

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, createFileRoute } from "@tanstack/react-router";
import {
  ArrowLeft,
  CheckCircle2,
  FileCode2,
  FolderOpen,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { WindowControls } from "@/components/app-top-bar-controls";
import { AgentSubagentSelect } from "@/features/agents/agent-subagent-select";
import { agentModeSubagentCandidates } from "@/features/projects/extension-settings-agent-mode-model";
import {
  createAgentPrompt,
  createQuickPrompt,
  deletePromptDocument,
  getPromptDirectory,
  listAgentModes,
  listPromptDocuments,
  listPromptToolDescriptions,
  reloadPromptCatalog,
  resetPromptDocument,
  savePromptDocument,
  setPromptDocumentEnabled,
  type PromptCategory,
  type PromptDocument,
  type PromptToolDescription,
  type AgentModeDefinition,
} from "@/services/aivo";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/prompts")({
  component: PromptManagementRoute,
});

const categoryLabels: Record<PromptCategory, string> = {
  agent: "Agent",
  protocol: "协议",
  auxiliary: "辅助任务",
  task: "内置任务",
  dynamic_context: "动态上下文",
  quick_prompt: "快捷提示",
};

type CatalogCategory = PromptCategory | "tool" | "all";
type Draft = Pick<PromptDocument, "id" | "category" | "title" | "body" | "enabled">;

function PromptManagementRoute() {
  const [documents, setDocuments] = useState<PromptDocument[]>([]);
  const [tools, setTools] = useState<PromptToolDescription[]>([]);
  const [agentModes, setAgentModes] = useState<AgentModeDefinition[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [selectedTool, setSelectedTool] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState<CatalogCategory>("all");
  const [status, setStatus] = useState("all");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const selected = documents.find((item) => item.id === selectedID);
  const tool = tools.find((item) => item.name === selectedTool);
  const dirty = Boolean(
    selected &&
      draft &&
      (draft.title !== selected.title ||
        draft.body !== selected.body ||
        draft.enabled !== selected.enabled),
  );

  const load = async (reload = false) => {
    setLoading(true);
    try {
      const [nextDocuments, nextTools, nextAgentModes] = await Promise.all([
        reload ? reloadPromptCatalog() : listPromptDocuments(),
        listPromptToolDescriptions(),
        listAgentModes(),
      ]);
      setDocuments(nextDocuments);
      setTools(nextTools);
      setAgentModes(nextAgentModes);
      if (
        !selectedID &&
        nextDocuments.length > 0 &&
        !window.matchMedia("(max-width: 767px)").matches
      ) {
        selectDocument(nextDocuments[0]);
      }
      if (reload) toast.success("提示词目录已重新加载");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载提示词失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    // The initial catalog load should run once for this global route.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const guard = (event: BeforeUnloadEvent) => {
      if (!dirty) return;
      event.preventDefault();
    };
    window.addEventListener("beforeunload", guard);
    return () => window.removeEventListener("beforeunload", guard);
  }, [dirty]);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    return documents.filter((item) => {
      if (category !== "all" && category !== "tool" && item.category !== category) return false;
      if (category === "tool") return false;
      if (status === "error" && item.status !== "invalid") return false;
      if (status === "override" && item.origin !== "override") return false;
      if (status === "disabled" && item.enabled) return false;
      return !query || `${item.title} ${item.id} ${item.body}`.toLowerCase().includes(query);
    });
  }, [category, documents, search, status]);

  const filteredTools = useMemo(() => {
    if (category !== "tool") return [];
    const query = search.trim().toLowerCase();
    return tools.filter((item) => !query || `${item.name} ${item.description}`.toLowerCase().includes(query));
  }, [category, search, tools]);

  function canLeaveDraft() {
    return !dirty || window.confirm("当前提示词有未保存修改，确定离开吗？");
  }

  function selectDocument(document: PromptDocument) {
    if (!canLeaveDraft()) return;
    setSelectedTool("");
    setSelectedID(document.id);
    setDraft({
      id: document.id,
      category: document.category,
      title: document.title,
      body: document.body,
      enabled: document.enabled,
    });
  }

  async function save() {
    if (!draft) return;
    setSaving(true);
    try {
      const saved = await savePromptDocument(draft);
      setDocuments((items) => items.map((item) => (item.id === saved.id ? saved : item)));
      setDraft({ id: saved.id, category: saved.category, title: saved.title, body: saved.body, enabled: saved.enabled });
      toast[saved.status === "invalid" ? "warning" : "success"](
        saved.status === "invalid" ? "草稿已保存，校验未通过，运行时仍使用最近有效版本" : "提示词已保存并应用",
      );
      window.dispatchEvent(new CustomEvent("aivo:prompts-changed"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存失败");
    } finally {
      setSaving(false);
    }
  }

  async function reset() {
    if (!selected || !window.confirm(`重置“${selected.title}”到内置默认值？`)) return;
    try {
      const next = await resetPromptDocument(selected.id);
      setDocuments((items) => items.map((item) => (item.id === next.id ? next : item)));
      selectDocument(next);
      toast.success("已重置为内置默认值");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "重置失败");
    }
  }

  async function toggleEnabled() {
    if (!selected) return;
    try {
      const next = await setPromptDocumentEnabled(selected.id, !selected.enabled);
      setDocuments((items) => items.map((item) => (item.id === next.id ? next : item)));
      selectDocument(next);
      window.dispatchEvent(new CustomEvent("aivo:prompts-changed"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "更新状态失败");
    }
  }

  async function remove() {
    if (!selected || !window.confirm(`永久删除“${selected.title}”？`)) return;
    try {
      await deletePromptDocument(selected.id);
      const next = documents.filter((item) => item.id !== selected.id);
      setDocuments(next);
      setSelectedID("");
      setDraft(null);
      if (next.length > 0) selectDocument(next[0]);
      window.dispatchEvent(new CustomEvent("aivo:prompts-changed"));
      toast.success("提示词已删除");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除失败");
    }
  }

  return (
    <div className="flex h-dvh min-h-0 flex-col bg-background text-foreground">
      <header className="relative z-50 flex h-9 shrink-0 items-center gap-2 border-b px-3" data-app-drag>
        <WindowControls />
        <Button asChild size="sm" variant="ghost">
          <Link
            data-app-no-drag
            onClick={(event) => { if (!canLeaveDraft()) event.preventDefault(); }}
            to="/projects/chat"
          >
            <ArrowLeft data-icon="inline-start" />
            返回主页
          </Link>
        </Button>
        <div className="mx-auto flex items-center gap-2 text-sm font-semibold">
          <FileCode2 className="size-4" /> 提示词管理
        </div>
        <div className="flex items-center gap-1" data-app-no-drag>
          <NewPromptDialog agentModes={agentModes} onCreated={() => void load()} />
          <Button aria-label="重新加载提示词目录" onClick={() => void load(true)} size="icon" variant="ghost">
            <RefreshCw />
          </Button>
          <Button
            aria-label="打开提示词目录"
            onClick={() => void getPromptDirectory().then((path) => window.aivo.openPath(path)).catch((error) => toast.error(String(error)))}
            size="icon"
            variant="ghost"
          >
            <FolderOpen />
          </Button>
        </div>
      </header>

      <main className="grid min-h-0 flex-1 grid-cols-[220px_minmax(220px,340px)_minmax(0,1fr)] max-md:grid-cols-1">
        <aside className={cn("min-h-0 border-r p-3 max-md:border-r-0", (selected || tool) && "max-md:hidden")}>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input className="pl-8" onChange={(event) => setSearch(event.target.value)} placeholder="搜索提示词" value={search} />
          </div>
          <div className="mt-3 grid gap-2">
            <NativeSelect className="w-full" disabled={category === "tool"} onChange={(event) => setStatus(event.target.value)} value={status}>
              <NativeSelectOption value="all">全部状态</NativeSelectOption>
              <NativeSelectOption value="error">有错误</NativeSelectOption>
              <NativeSelectOption value="override">已覆盖</NativeSelectOption>
              <NativeSelectOption value="disabled">已禁用</NativeSelectOption>
            </NativeSelect>
          </div>
          <div className="mt-5 space-y-1 text-xs text-muted-foreground">
            {Object.entries(categoryLabels).map(([value, label]) => (
              <button className={cn("flex w-full justify-between rounded px-2 py-1.5 hover:bg-muted", category === value && "bg-muted text-foreground")} key={value} onClick={() => setCategory(value as PromptCategory)} type="button">
                <span>{label}</span><span>{documents.filter((item) => item.category === value).length}</span>
              </button>
            ))}
          </div>
        </aside>

        <ScrollArea className={cn("min-h-0 border-r max-md:border-r-0", (selected || tool) && "max-md:hidden")}>
          <div className="p-2">
            {loading ? <p className="p-4 text-sm text-muted-foreground">正在加载…</p> : null}
            {filtered.map((item) => (
              <button className={cn("mb-1 w-full rounded-lg border border-transparent p-3 text-left hover:bg-muted/70", item.id === selectedID && "border-border bg-muted")} key={item.id} onClick={() => selectDocument(item)} type="button">
                <div className="flex items-start justify-between gap-2">
                  <span className="truncate text-sm font-medium">{item.title}</span>
                  {item.status === "invalid" ? <TriangleAlert className="size-4 shrink-0 text-destructive" /> : <CheckCircle2 className="size-4 shrink-0 text-emerald-500" />}
                </div>
                <p className="mt-1 truncate font-mono text-[11px] text-muted-foreground">{item.id}</p>
                <div className="mt-2 flex flex-wrap gap-1">
                  <Badge variant="outline">{item.origin === "builtin" ? "内置" : "覆盖"}</Badge>
                  {!item.enabled ? <Badge variant="secondary">已禁用</Badge> : null}
                  {item.fallback ? <Badge variant="destructive">正在回退</Badge> : null}
                  {item.deletable ? <Badge variant="outline">可删除</Badge> : null}
                </div>
              </button>
            ))}
            {filteredTools.map((item) => (
              <button className="mb-1 w-full rounded-lg p-3 text-left hover:bg-muted" key={item.name} onClick={() => { setSelectedID(""); setDraft(null); setSelectedTool(item.name); }} type="button">
                <span className="text-sm font-medium">{item.name}</span>
                <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{item.description}</p>
              </button>
            ))}
          </div>
        </ScrollArea>

        <section className="min-h-0 min-w-0">
          {selected && draft ? (
            <ScrollArea className="h-full">
              <div className="mx-auto max-w-5xl p-4 lg:p-6">
                <Button className="mb-3 md:hidden" onClick={() => { if (canLeaveDraft()) { setSelectedID(""); setDraft(null); } }} size="sm" variant="ghost"><ArrowLeft /> 返回列表</Button>
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div><h1 className="text-xl font-semibold">{selected.title}</h1><p className="mt-1 font-mono text-xs text-muted-foreground">{selected.id}</p></div>
                  <div className="flex flex-wrap gap-2">
                    <Button disabled={saving || !dirty} onClick={() => void save()}>{saving ? "保存中…" : "保存并应用"}</Button>
                    {selected.origin !== "builtin" && !selected.deletable ? <Button onClick={() => void reset()} variant="outline"><RotateCcw />重置</Button> : null}
                    {selected.disableable ? <Button onClick={() => void toggleEnabled()} variant="outline">{selected.enabled ? "禁用" : "启用"}</Button> : null}
                    {selected.deletable ? <Button onClick={() => void remove()} variant="destructive"><Trash2 />删除</Button> : null}
                  </div>
                </div>

                <div className="mt-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
                  <div className="space-y-3">
                    <Input aria-label="提示词标题" onChange={(event) => setDraft({ ...draft, title: event.target.value })} value={draft.title} />
                    <Textarea aria-label="Markdown 提示词正文" className="min-h-[52vh] resize-y font-mono text-[13px] leading-6" onChange={(event) => setDraft({ ...draft, body: event.target.value })} spellCheck={false} value={draft.body} />
                    <details className="rounded-lg border p-3"><summary className="cursor-pointer text-sm font-medium">预览</summary><pre className="mt-3 whitespace-pre-wrap break-words text-sm text-muted-foreground">{draft.body}</pre></details>
                  </div>
                  <aside className="space-y-3">
                    <InfoCard title="变量契约">
                      {selected.variables?.length ? selected.variables.map((name) => <code className="mr-1 inline-block rounded bg-muted px-1.5 py-0.5 text-xs" key={name}>{`{{${name}}}`}</code>) : <span>无动态变量</span>}
                    </InfoCard>
                    <InfoCard title="生效状态">
                      <p>working: <Revision value={selected.workingRevision} /></p><p>active: <Revision value={selected.activeRevision} /></p>
                      {selected.fallback ? <p className="mt-2 text-destructive">当前草稿无效，运行时使用最近有效版本。</p> : null}
                    </InfoCard>
                    <InfoCard title="诊断">
                      {selected.diagnostics?.length ? selected.diagnostics.map((diagnostic) => <p className="mb-2 text-destructive" key={`${diagnostic.code}-${diagnostic.line}-${diagnostic.column}`}>{diagnostic.line ? `${diagnostic.line}:${diagnostic.column || 1} ` : ""}{diagnostic.message}</p>) : <p className="text-emerald-600">校验通过</p>}
                    </InfoCard>
                    <InfoCard title="操作策略"><p>{selected.required ? "必需项：可编辑、可重置，不可禁用或删除。" : selected.deletable ? "自定义项：允许删除。" : "可选内置项：可重置或禁用。"}</p></InfoCard>
                  </aside>
                </div>
              </div>
            </ScrollArea>
          ) : tool ? (
            <div className="mx-auto max-w-3xl p-6"><Button className="mb-3 md:hidden" onClick={() => setSelectedTool("")} size="sm" variant="ghost"><ArrowLeft />返回列表</Button><Badge variant="outline">只读工具说明</Badge><h1 className="mt-3 text-xl font-semibold">{tool.name}</h1><p className="mt-4 whitespace-pre-wrap text-sm leading-6 text-muted-foreground">{tool.description}</p><p className="mt-6 text-xs text-muted-foreground">工具 Schema、参数、权限与授权逻辑由 Core 代码拥有，提示词不能修改这些能力。</p></div>
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">从列表选择一个提示词</div>
          )}
        </section>
      </main>
    </div>
  );
}

function InfoCard({ children, title }: { children: ReactNode; title: string }) {
  return <div className="rounded-lg border bg-card p-3 text-xs leading-5 text-muted-foreground"><h2 className="mb-2 text-sm font-medium text-foreground">{title}</h2>{children}</div>;
}

function Revision({ value }: { value?: string }) {
  return <code title={value} className="font-mono">{value ? value.slice(0, 10) : "—"}</code>;
}

export function NewPromptDialog({ agentModes, onCreated }: { agentModes: AgentModeDefinition[]; onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<"agent" | "quick_prompt">("agent");
  const [mode, setMode] = useState<"primary" | "subagent" | "all">("all");
  const [subagents, setSubagents] = useState<string[]>([]);
  const [id, setID] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [saving, setSaving] = useState(false);
  const subagentCandidates = useMemo(
    () => agentModeSubagentCandidates(agentModes, id.trim().toLowerCase().replace(/^agent\./, "")),
    [agentModes, id],
  );

  async function create() {
    setSaving(true);
    try {
      if (kind === "agent") await createAgentPrompt({ id, title, body, permissionScope: "read_only", mode, subagents: mode === "subagent" ? [] : subagents.filter((subagent) => subagentCandidates.some((candidate) => candidate.id === subagent)) });
      else await createQuickPrompt({ id, title, body });
      setOpen(false); setID(""); setTitle(""); setBody(""); setMode("all"); setSubagents([]); onCreated();
      toast.success(kind === "agent" ? "Agent 提示词和安全默认模式已创建" : "快捷提示已创建");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "创建失败");
    } finally { setSaving(false); }
  }

  return (
    <Dialog onOpenChange={setOpen} open={open}>
      <DialogTrigger asChild><Button size="sm" variant="outline"><Plus />新建</Button></DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto">
        <DialogHeader><DialogTitle>新建提示词</DialogTitle><DialogDescription>只创建有明确消费者的 Agent 或首页快捷提示。</DialogDescription></DialogHeader>
        <div className="grid gap-3">
          <NativeSelect className="w-full" onChange={(event) => setKind(event.target.value as "agent" | "quick_prompt")} value={kind}><NativeSelectOption value="agent">Agent（同时创建安全默认模式）</NativeSelectOption><NativeSelectOption value="quick_prompt">快捷提示</NativeSelectOption></NativeSelect>
          <Input onChange={(event) => setID(event.target.value)} placeholder={kind === "agent" ? "research" : "daily_review"} value={id} />
          <Input onChange={(event) => setTitle(event.target.value)} placeholder="显示标题" value={title} />
          {kind === "agent" ? (
            <>
              <NativeSelect
                aria-label="Agent 角色"
                className="w-full"
                onChange={(event) => {
                  const nextMode = event.target.value as "primary" | "subagent" | "all";
                  setMode(nextMode);
                  if (nextMode === "subagent") setSubagents([]);
                }}
                value={mode}
              >
                <NativeSelectOption value="all">主 Agent / 子 Agent</NativeSelectOption>
                <NativeSelectOption value="primary">仅主 Agent</NativeSelectOption>
                <NativeSelectOption value="subagent">仅子 Agent</NativeSelectOption>
              </NativeSelect>
              {mode !== "subagent" ? (
                <div className="grid gap-1.5">
                  <p className="text-xs text-muted-foreground" id="new-agent-subagents-help">
                    最多可多选 16 个。运行时只会注入并允许委托这里关联的子 Agent。
                  </p>
                  <AgentSubagentSelect
                    candidates={subagentCandidates}
                    describedBy="new-agent-subagents-help"
                    disabled={saving}
                    onChange={setSubagents}
                    value={subagents}
                  />
                </div>
              ) : null}
            </>
          ) : null}
          <Textarea className="min-h-40 font-mono" onChange={(event) => setBody(event.target.value)} placeholder="Markdown 提示词正文" value={body} />
        </div>
        <DialogFooter><Button disabled={saving || !id.trim() || !title.trim() || !body.trim()} onClick={() => void create()}>{saving ? "创建中…" : "创建"}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
