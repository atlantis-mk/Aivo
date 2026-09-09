import { useCallback, useEffect, useState } from "react";
import { ChevronDown, CornerDownLeft, Hand } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { Kbd } from "@/components/ui/kbd";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type {
  CodexApprovalAction,
  CodexApprovalRequest,
} from "@/codex-app-server";

type ApprovalActionState = "idle" | "approving" | "denying" | "approved" | "denied";
type ApprovalActionOption = Extract<
  CodexApprovalAction,
  "approve" | "approveForSession"
>;

export function useCodexApprovalRequests() {
  const [requests, setRequests] = useState<CodexApprovalRequest[]>([]);
  const [approvalAction, setApprovalAction] =
    useState<ApprovalActionOption>("approve");

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

  return { approvalAction, requests, resolve, setApprovalAction };
}

export function CodexApprovalDock({
  approvalAction,
  onApprovalActionSelect,
  requests,
  resolve,
}: {
  approvalAction: ApprovalActionOption;
  onApprovalActionSelect: (action: ApprovalActionOption) => void;
  requests: CodexApprovalRequest[];
  resolve: (
    request: CodexApprovalRequest,
    action: CodexApprovalAction,
  ) => Promise<void>;
}) {
  if (requests.length === 0) return null;

  return (
    <div className="absolute bottom-4 left-1/2 z-20 w-[calc(100%-2rem)] max-w-[736px] -translate-x-1/2 sm:w-[calc(100%-48px)]">
      <div className="max-h-[min(54vh,420px)] overflow-auto rounded-2xl border border-border bg-popover text-popover-foreground shadow-xl shadow-foreground/10">
          {requests.map((request, index) => (
            <CodexApprovalCard
              approvalAction={approvalAction}
              enableShortcuts={index === 0}
              key={request.id}
              onApprovalActionSelect={onApprovalActionSelect}
              request={request}
              resolve={resolve}
            />
          ))}
      </div>
    </div>
  );
}

function CodexApprovalCard({
  approvalAction,
  enableShortcuts,
  onApprovalActionSelect,
  request,
  resolve,
}: {
  approvalAction: ApprovalActionOption;
  enableShortcuts: boolean;
  onApprovalActionSelect: (action: ApprovalActionOption) => void;
  request: CodexApprovalRequest;
  resolve: (
    request: CodexApprovalRequest,
    action: CodexApprovalAction,
  ) => Promise<void>;
}) {
  const [action, setAction] = useState<ApprovalActionState>("idle");
  const presentation = approvalPresentation(request);

  const submit = useCallback(
    async (nextAction: ApprovalActionOption | "deny") => {
      if (action !== "idle") return;
      setAction(nextAction === "deny" ? "denying" : "approving");
      try {
        await resolve(request, nextAction);
        setAction(nextAction === "deny" ? "denied" : "approved");
      } catch {
        setAction("idle");
      }
    },
    [action, request, resolve],
  );

  useEffect(() => {
    if (!enableShortcuts) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || action !== "idle") return;
      if (event.key === "Escape") {
        event.preventDefault();
        void submit("deny");
      }
      if (event.key === "Enter" && !event.metaKey && !event.ctrlKey) {
        event.preventDefault();
        void submit(approvalAction);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [action, approvalAction, enableShortcuts, submit]);

  return (
    <section className="border-b border-border p-3 text-popover-foreground last:border-b-0">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Hand aria-hidden="true" className="size-4" strokeWidth={2} />
        <span>{presentation.category}</span>
      </div>
      <h3 className="mt-4 font-semibold tracking-tight">
        {presentation.prompt}
      </h3>
      <p className="mt-2 max-w-5xl text-xs leading-relaxed text-muted-foreground">
        {presentation.description}
      </p>
      <div className="mt-5 flex flex-wrap justify-end gap-3">
        <Button
          aria-keyshortcuts="Escape"
          disabled={action !== "idle"}
          onClick={() => void submit("deny")}
          type="button"
          variant="outline"
        >
          {action === "denying" ? "拒绝中…" : action === "denied" ? "已拒绝" : "拒绝"}
          <Kbd>Esc</Kbd>
        </Button>
        <ButtonGroup>
          <Button
            aria-keyshortcuts="Enter"
            disabled={action !== "idle"}
            onClick={() => void submit(approvalAction)}
            type="button"
            variant="default"
          >
            {action === "approving"
              ? "允许中…"
              : action === "approved"
                ? "已允许"
                : approvalAction === "approveForSession"
                  ? "允许此对话"
                  : "允许一次"}
            <Kbd className="bg-primary-foreground/15 text-primary-foreground">
              <CornerDownLeft aria-hidden="true" />
            </Kbd>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                aria-label="更多允许选项"
                disabled={action !== "idle"}
                size="icon"
                type="button"
                variant="default"
              >
                <ChevronDown aria-hidden="true" className="size-5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52" side="top">
              <DropdownMenuRadioGroup
                onValueChange={(value) =>
                  onApprovalActionSelect(value as ApprovalActionOption)
                }
                value={approvalAction}
              >
                <DropdownMenuRadioItem value="approve">
                  允许一次
                </DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="approveForSession">
                  允许此对话
                </DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </ButtonGroup>
      </div>
    </section>
  );
}

function approvalPresentation(request: CodexApprovalRequest) {
  const isInternetAccess =
    request.kind === "commandExecution" &&
    /\b(curl|wget|fetch)\b|https?:\/\//i.test(request.command ?? "");

  if (isInternetAccess) {
    return {
      category: "互联网访问",
      prompt: "允许 ChatGPT 连接互联网？",
      description:
        request.reason ?? "执行此网络请求前需要您的批准。",
    };
  }

  if (request.kind === "fileChange") {
    return {
      category: "文件修改",
      prompt: "允许 ChatGPT 修改文件？",
      description:
        request.reason ?? "修改文件前需要您的批准。",
    };
  }

  return {
    category: "命令执行",
    prompt: "允许 ChatGPT 执行此命令？",
    description: request.reason ?? "执行此命令前需要您的批准。",
  };
}
