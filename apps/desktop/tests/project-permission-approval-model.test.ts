import assert from "node:assert/strict";
import test from "node:test";

import {
  permissionApprovalTarget,
  permissionApprovalTitle,
  permissionCommand,
  permissionMCPRegistration,
  permissionProject,
  permissionRememberLabel,
} from "../src/features/projects/project-permission-approval-model.ts";
import type { PermissionRequest } from "../src/services/aivo/interaction-service.ts";

function permission(
  toolName: string,
  args: Record<string, unknown>,
): PermissionRequest {
  return {
    id: "permission-1",
    toolName,
    action: "write",
    arguments: args,
    status: "pending",
    timeCreated: "2026-08-04T00:00:00Z",
    timeUpdated: "2026-08-04T00:00:00Z",
  };
}

test("project add approval exposes the concrete root and bounded remember scope", () => {
  const request = permission("aivo_projects_add", {
    projectOperation: "add",
    projectRoot: "/Users/example/a/very/long/path/to/an/existing/project",
    rememberScope: "exact_project",
  });
  const command = permissionCommand(request);

  assert.equal(permissionApprovalTitle(request, command), "批准添加项目");
  assert.equal(
    permissionApprovalTarget(request, command),
    "/Users/example/a/very/long/path/to/an/existing/project",
  );
  assert.deepEqual(permissionProject(request), {
    operation: "add",
    rootPath: "/Users/example/a/very/long/path/to/an/existing/project",
    immutableAssociation: false,
  });
  assert.equal(permissionRememberLabel(request, command), "记住此项目操作");
});

test("project association approval carries the permanent-binding warning state", () => {
  const request = permission("aivo_projects_associate", {
    projectOperation: "associate",
    projectName: "Aivo",
    projectRoot: "/Users/example/Aivo",
    immutableAssociation: true,
    rememberScope: "exact_project",
  });
  const command = permissionCommand(request);

  assert.equal(permissionApprovalTitle(request, command), "批准关联项目");
  assert.deepEqual(permissionProject(request), {
    operation: "associate",
    name: "Aivo",
    rootPath: "/Users/example/Aivo",
    immutableAssociation: true,
  });
  assert.equal(permissionRememberLabel(request, command), "记住此项目操作");
});

test("MCP registration approval exposes the exact global source without a reusable grant", () => {
  const request = permission("aivo_tools_register_mcp", {
    registrationKind: "mcp",
    registrationServerId: "github_mcp",
    registrationName: "GitHub MCP",
    registrationTransport: "streamable_http",
    registrationTarget: "https://mcp.example.test/v1",
    registrationRoots: ["/Users/example/Aivo"],
    registrationAuth: "bearer",
    registrationBearerTokenEnv: "GITHUB_TOKEN",
    registrationGlobal: true,
    rememberScope: "never",
  });
  const command = permissionCommand(request);

  assert.equal(permissionApprovalTitle(request, command), "批准注册 MCP 工具");
  assert.equal(
    permissionApprovalTarget(request, command),
    "https://mcp.example.test/v1",
  );
  assert.deepEqual(permissionMCPRegistration(request), {
    id: "github_mcp",
    name: "GitHub MCP",
    transport: "streamable_http",
    target: "https://mcp.example.test/v1",
    cwd: undefined,
    roots: ["/Users/example/Aivo"],
    auth: "bearer",
    bearerTokenEnv: "GITHUB_TOKEN",
    global: true,
  });
  assert.equal(
    permissionRememberLabel(request, command),
    "此注册必须每次单独确认",
  );
});
