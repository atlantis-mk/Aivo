import { useCallback, useEffect, useState } from "react";
import { GitBranch, Loader2, RefreshCw, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  bindSessionToGitWorktree,
  createGitWorktree,
  listGitWorktrees,
  removeGitWorktree,
  resetGitWorktree,
  type GitWorktree,
} from "@/services/aivo/worktree-service";

export function ProjectWorktreeDialog({
  repositoryPath,
  sessionId,
}: {
  repositoryPath: string;
  sessionId?: string;
}) {
  const [open, setOpen] = useState(false);
  const [branch, setBranch] = useState("");
  const [detached, setDetached] = useState(false);
  const [startupCommand, setStartupCommand] = useState("");
  const [items, setItems] = useState<GitWorktree[]>([]);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const refresh = useCallback(async () => {
    if (!repositoryPath) return;
    setBusy("list");
    setError("");
    try {
      setItems(await listGitWorktrees(repositoryPath));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  }, [repositoryPath]);

  useEffect(() => {
    if (open) void refresh();
  }, [open, refresh]);

  const create = async () => {
    const command = startupCommand.trim();
    if (command && !window.confirm("创建后将在新 Worktree 中执行启动命令。确认继续？")) return;
    setBusy("create");
    setError("");
    setMessage("");
    try {
      const created = await createGitWorktree({
        repositoryPath,
        branch: detached ? undefined : branch.trim() || undefined,
        name: branch.trim() || undefined,
        detached,
        startupCommand: command || undefined,
        startupConfirmed: Boolean(command),
        sessionId: sessionId || undefined,
      });
      setBranch("");
      setStartupCommand("");
      setMessage(`已创建并使用 ${created.branch || created.path}`);
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  const bind = async (worktree: GitWorktree) => {
    if (!sessionId) return;
    setBusy(worktree.id);
    setError("");
    try {
      await bindSessionToGitWorktree(sessionId, worktree.id);
      setMessage(`当前会话已切换到 ${worktree.branch || worktree.path}`);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  const reset = async (worktree: GitWorktree) => {
    if (!window.confirm(`重置 ${worktree.branch || worktree.path} 的所有未提交修改？`))
      return;
    setBusy(worktree.id);
    setError("");
    try {
      await resetGitWorktree(worktree.id, { clean: true, confirmed: true });
      setMessage("Worktree 已重置");
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  const remove = async (worktree: GitWorktree) => {
    if (!window.confirm(`永久移除 worktree ${worktree.branch || worktree.path}？`))
      return;
    setBusy(worktree.id);
    setError("");
    try {
      await removeGitWorktree(worktree.id, { force: true, confirmed: true });
      setMessage("Worktree 已移除");
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  if (!repositoryPath) return null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button aria-label="管理 Git Worktree" size="icon-sm" variant="ghost">
          <GitBranch />
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Git Worktrees</DialogTitle>
          <DialogDescription>
            为并行任务创建隔离工作区。重置和移除始终需要再次确认。
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <div className="flex gap-2">
            <Input
              aria-label="新 Worktree 分支或名称"
              disabled={detached}
              onChange={(event) => setBranch(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") void create();
              }}
              placeholder="可选；留空自动创建 aivo/* 分支"
              value={branch}
            />
            <Button disabled={busy !== ""} onClick={() => void create()} size="sm">
            {busy === "create" ? <Loader2 className="animate-spin" /> : null}
            创建
            </Button>
            <Button aria-label="刷新 Worktree" disabled={busy !== ""} onClick={() => void refresh()} size="icon-sm" variant="outline">
              <RefreshCw className={busy === "list" ? "animate-spin" : ""} />
            </Button>
          </div>
          <Input aria-label="Worktree 启动命令" onChange={(event) => setStartupCommand(event.target.value)} placeholder="可选启动命令（执行前会确认）" value={startupCommand} />
          <label className="flex items-center gap-2 text-muted-foreground">
            <input checked={detached} onChange={(event) => setDetached(event.target.checked)} type="checkbox" />
            Detached Worktree
          </label>
        </div>
        {error ? <p className="rounded-md bg-destructive/10 px-3 py-2 text-destructive">{error}</p> : null}
        {message ? <p className="rounded-md bg-muted px-3 py-2 text-foreground">{message}</p> : null}
        <div className="max-h-80 space-y-2 overflow-y-auto">
          {busy === "list" && items.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">正在读取 Worktrees…</p>
          ) : null}
          {busy !== "list" && items.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">还没有 Aivo 管理的 Worktree</p>
          ) : null}
          {items.map((worktree) => (
            <div className="flex items-center gap-3 rounded-lg border border-border/70 p-3" key={worktree.id}>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate font-medium">{worktree.branch || "detached"}</span>
                  <Badge variant={worktree.status === "ready" ? "secondary" : "destructive"}>{worktree.status}</Badge>
                  {!worktree.managed ? <Badge variant="outline">external</Badge> : null}
                  {worktree.dirty ? <Badge variant="outline">dirty</Badge> : null}
                </div>
                <p className="truncate text-muted-foreground" title={worktree.path}>{worktree.path}</p>
                {worktree.activeSessions?.length ? (
                  <p className="text-amber-600">{worktree.activeSessions.length} 个活动会话正在使用</p>
                ) : null}
              </div>
              {sessionId ? (
                <Button disabled={busy !== "" || worktree.status !== "ready" || !worktree.managed} onClick={() => void bind(worktree)} size="sm" variant="outline">使用</Button>
              ) : null}
              <Button aria-label="重置 Worktree" disabled={busy !== "" || !worktree.managed || (worktree.status !== "ready" && worktree.status !== "error")} onClick={() => void reset(worktree)} size="icon-sm" variant="ghost">
                <RefreshCw />
              </Button>
              <Button aria-label="移除 Worktree" disabled={busy !== "" || !worktree.managed || Boolean(worktree.activeSessions?.length)} onClick={() => void remove(worktree)} size="icon-sm" variant="ghost">
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
