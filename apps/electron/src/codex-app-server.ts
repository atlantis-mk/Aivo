export type PermissionMode =
  | "request_approval"
  | "auto_review"
  | "full_access";

export type CodexApprovalKind = "commandExecution" | "fileChange";

export type CodexApprovalAction =
  | "approve"
  | "approveForSession"
  | "deny"
  | "cancel";

export type CodexApprovalRequest = {
  id: string;
  kind: CodexApprovalKind;
  threadId: string;
  turnId: string;
  itemId: string;
  title: string;
  command?: string;
  cwd?: string;
  reason?: string;
  startedAt: string;
};

export type CodexPermissionPolicyInput = {
  approvalPolicy?: "on-request" | "never";
  approvalsReviewer?: "user" | "auto_review";
  sandbox?: "workspace-write" | "danger-full-access";
  sandboxPolicy?: { type: "workspaceWrite"; networkAccess: boolean } | {
    type: "dangerFullAccess";
  };
};

const APPROVAL_METHODS = {
  "item/commandExecution/requestApproval": "commandExecution",
  "item/fileChange/requestApproval": "fileChange",
} as const;

export type CodexUserInputOption = {
  label: string;
  description?: string;
};

export type CodexUserInputQuestion = {
  id: string;
  header: string;
  question: string;
  options?: CodexUserInputOption[];
};

export type CodexUserInputRequest = {
  id: number | string;
  itemId: string;
  threadId: string;
  turnId: string;
  isBlocking: boolean;
  questions: CodexUserInputQuestion[];
};

export type CodexUserInputAnswers = Record<
  string,
  { answers: string[] }
>;

export function isCodexApprovalMethod(method: string) {
  return method in APPROVAL_METHODS;
}

export function isCodexUserInputMethod(method: string) {
  return method === "request_user_input";
}

export function codexPermissionPolicy(
  mode: PermissionMode,
): CodexPermissionPolicyInput {
  if (mode === "full_access") {
    return {
      approvalPolicy: "never",
      approvalsReviewer: "user",
      sandbox: "danger-full-access",
      sandboxPolicy: { type: "dangerFullAccess" },
    };
  }
  if (mode === "auto_review") {
    return {
      approvalPolicy: "on-request",
      approvalsReviewer: "auto_review",
      sandbox: "workspace-write",
      sandboxPolicy: { type: "workspaceWrite", networkAccess: false },
    };
  }
  return {
    approvalPolicy: "on-request",
    approvalsReviewer: "user",
    sandbox: "workspace-write",
    sandboxPolicy: { type: "workspaceWrite", networkAccess: false },
  };
}

export function normalizeCodexApprovalRequest(
  requestId: number | string,
  method: string,
  params: unknown,
): CodexApprovalRequest | null {
  const kind = APPROVAL_METHODS[method as keyof typeof APPROVAL_METHODS];
  const value = recordValue(params);
  const threadId = stringOrNull(value?.threadId);
  const turnId = stringOrNull(value?.turnId);
  const itemId = stringOrNull(value?.itemId);
  if (!kind || !threadId || !turnId || !itemId) return null;

  const command = stringOrNull(value?.command);
  return {
    id: String(requestId),
    kind,
    threadId,
    turnId,
    itemId,
    title: kind === "commandExecution" ? "执行命令" : "修改文件",
    command,
    cwd: stringOrNull(value?.cwd),
    reason: stringOrNull(value?.reason),
    startedAt: new Date(
      typeof value?.startedAtMs === "number" ? value.startedAtMs : Date.now(),
    ).toISOString(),
  };
}

export function codexApprovalResponse(
  kind: CodexApprovalKind,
  action: CodexApprovalAction,
) {
  void kind;
  const decision =
    action === "approve"
      ? "accept"
      : action === "approveForSession"
        ? "acceptForSession"
        : action === "deny"
          ? "decline"
          : "cancel";
  return { decision };
}

function recordValue(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null
    ? (value as Record<string, unknown>)
    : null;
}

function stringOrNull(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}
