import { useCallback, useEffect, useState } from "react";
import { Check, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import type {
  CodexApprovalAction,
  CodexApprovalRequest,
} from "@/codex-app-server";
import { cn } from "@/lib/utils";

type ApprovalActionState = "idle" | "approving" | "denying" | "approved" | "denied";

export function useCodexApprovalRequests() {
  const [requests, setRequests] = useState<CodexApprovalRequest[]>([]);

  useEffect(() => {
    let mounted = true;
    void window.aivoDesktop.codex
      .listApprovals()
      .then((initialRequests) => {
        if (mounted) setRequests(initialRequests);
      })
      .catch(() => undefined);

    const onRequest = (request: CodexApprovalRequest) => {
      setRequests((currentRequests) =>
        currentRequests.some((item) => item.id === request.id)
          ? currentRequests
          : [request, ...currentRequests],
      );
    };
    const onResolved = ({ requestId }: { requestId: string }) => {
      setRequests((currentRequests) =>
        currentRequests.filter((request) => request.id !== requestId),
      );
    };
    const offRequest = window.aivoDesktop.codex.onApprovalRequest(onRequest);
    const offResolved = window.aivoDesktop.codex.onApprovalResolved(onResolved);

    return () => {
      mounted = false;
      offRequest();
      offResolved();
    };
  }, []);

  const resolve = useCallback(
    async (request: CodexApprovalRequest, action: CodexApprovalAction) => {
      await window.aivoDesktop.codex.resolveApproval(request.id, action);
      setRequests((currentRequests) =>
        currentRequests.filter((item) => item.id !== request.id),
      );
    },
    [],
  );

  return { requests, resolve };
}

export function CodexApprovalDock({
  requests,
  resolve,
}: {
  requests: CodexApprovalRequest[];
  resolve: (
    request: CodexApprovalRequest,
    action: CodexApprovalAction,
  ) => Promise<void>;
}) {
  if (requests.length === 0) return null;

  return (
    <div className="absolute bottom-4 left-1/2 z-20 w-[calc(100%-2rem)] max-w-[760px] -translate-x-1/2 sm:bottom-6 sm:w-[calc(100%-48px)]">
      <div className="overflow-hidden rounded-2xl border border-border/80 bg-popover text-popover-foreground shadow-2xl shadow-foreground/15 ring-1 ring-foreground/5">
        <div className="flex items-center gap-3 border-b border-border/70 px-4 py-3">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold">Codex 需要批准</div>
            <div className="truncate text-xs text-muted-foreground">
              批准后操作会发送回 Codex app-server；拒绝会返回明确的拒绝决定。
            </div>
          </div>
          {requests.length > 1 ? (
            <span className="rounded-full bg-primary/10 px-2 py-1 text-xs text-primary">
              {requests.length} 项
            </span>
          ) : null}
        </div>
        <div className="max-h-[min(54vh,420px)] overflow-auto">
          {requests.map((request, index) => (
            <CodexApprovalCard
              index={index}
              key={request.id}
              request={request}
              resolve={resolve}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function CodexApprovalCard({
  index,
  request,
  resolve,
}: {
  index: number;
  request: CodexApprovalRequest;
  resolve: (
    request: CodexApprovalRequest,
    action: CodexApprovalAction,
  ) => Promise<void>;
}) {
  const [action, setAction] = useState<ApprovalActionState>("idle");
  const [remember, setRemember] = useState(false);

  async function submit(nextAction: "approve" | "deny") {
    if (action !== "idle") return;
    setAction(nextAction === "approve" ? "approving" : "denying");
    try {
      const decision: CodexApprovalAction =
        nextAction === "approve" && remember
          ? "approveForSession"
          : nextAction;
      await resolve(request, decision);
      setAction(nextAction === "approve" ? "approved" : "denied");
    } catch {
      setAction("idle");
    }
  }

  return (
    <section className="border-b border-border/70 p-4 text-popover-foreground last:border-b-0">
      <div className="flex min-w-0 items-start gap-3">
        <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
          {index + 1}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <h3 className="min-w-0 truncate text-sm font-semibold">{request.title}</h3>
            <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
              {request.kind}
            </span>
          </div>
          {request.reason ? (
            <div className="mt-1 text-xs text-muted-foreground">{request.reason}</div>
          ) : null}
          {request.command ? (
            <pre className="mt-3 max-h-24 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/70 p-3 font-mono text-[11px] leading-relaxed text-foreground">
              {request.command}
            </pre>
          ) : null}
          <div className="mt-2 min-w-0 truncate text-[11px] text-muted-foreground">
            cwd: {request.cwd || "unknown"} · item: {request.itemId}
          </div>
        </div>
      </div>
      <div className="mt-3 flex flex-col gap-2 border-t border-border/70 pt-3 sm:flex-row sm:items-center sm:justify-between">
        <label className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
          <input
            checked={remember}
            className="size-3.5 accent-primary"
            disabled={action !== "idle"}
            onChange={(event) => setRemember(event.target.checked)}
            type="checkbox"
          />
          <span className="truncate">本次会话内记住批准</span>
        </label>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            className="h-8 px-3 text-xs"
            disabled={action !== "idle"}
            onClick={() => void submit("deny")}
            size="sm"
            type="button"
            variant="outline"
          >
            {action === "denying" ? "拒绝中" : action === "denied" ? "已拒绝" : "拒绝"}
          </Button>
          <Button
            className={cn("h-8 gap-1.5 px-3 text-xs")}
            disabled={action !== "idle"}
            onClick={() => void submit("approve")}
            size="sm"
            type="button"
          >
            {action === "approving" ? null : <Check />}
            {action === "approving"
              ? "批准中"
              : action === "approved"
                ? "已批准"
                : "批准并继续"}
          </Button>
        </div>
      </div>
    </section>
  );
}
