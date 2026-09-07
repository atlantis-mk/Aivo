import assert from "node:assert/strict";
import { test } from "node:test";

import {
  codexApprovalResponse,
  codexPermissionPolicy,
  isCodexApprovalMethod,
  normalizeCodexApprovalRequest,
} from "./codex-app-server.ts";

test("maps Aivo permission modes to Codex thread policies", () => {
  assert.deepEqual(codexPermissionPolicy("request_approval"), {
    approvalPolicy: "on-request",
    approvalsReviewer: "user",
    sandbox: "workspace-write",
    sandboxPolicy: { type: "workspaceWrite", networkAccess: false },
  });
  assert.deepEqual(codexPermissionPolicy("auto_review"), {
    approvalPolicy: "on-request",
    approvalsReviewer: "auto_review",
    sandbox: "workspace-write",
    sandboxPolicy: { type: "workspaceWrite", networkAccess: false },
  });
  assert.deepEqual(codexPermissionPolicy("full_access"), {
    approvalPolicy: "never",
    approvalsReviewer: "user",
    sandbox: "danger-full-access",
    sandboxPolicy: { type: "dangerFullAccess" },
  });
});

test("normalizes command and file approval server requests", () => {
  const params = {
    threadId: "thread-1",
    turnId: "turn-1",
    itemId: "item-1",
    startedAtMs: 100,
    command: "cargo test",
    cwd: "/tmp/project",
    reason: "network access",
  };

  assert.equal(
    isCodexApprovalMethod("item/commandExecution/requestApproval"),
    true,
  );
  assert.deepEqual(
    normalizeCodexApprovalRequest(
      12,
      "item/commandExecution/requestApproval",
      params,
    ),
    {
      id: "12",
      kind: "commandExecution",
      threadId: "thread-1",
      turnId: "turn-1",
      itemId: "item-1",
      title: "执行命令",
      command: "cargo test",
      cwd: "/tmp/project",
      reason: "network access",
      startedAt: new Date(100).toISOString(),
    },
  );
  assert.deepEqual(
    normalizeCodexApprovalRequest(
      "rpc-1",
      "item/fileChange/requestApproval",
      params,
    )?.kind,
    "fileChange",
  );
});

test("maps approval actions to protocol decisions", () => {
  assert.deepEqual(codexApprovalResponse("commandExecution", "approve"), {
    decision: "accept",
  });
  assert.deepEqual(
    codexApprovalResponse("fileChange", "approveForSession"),
    { decision: "acceptForSession" },
  );
  assert.deepEqual(codexApprovalResponse("fileChange", "deny"), {
    decision: "decline",
  });
});
