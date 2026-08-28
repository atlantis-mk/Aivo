import type { PermissionRequest } from "@/services/aivo";

export type PermissionActionState =
  "idle" | "approving" | "denying" | "approved" | "denied";

const writePermissionToolNames = new Set([
  "write_file",
  "edit_file",
]);

export type PermissionCommandInfo = {
  command: string;
  cwd?: string;
  riskLevel?: string;
  category?: string;
  networkHint?: string;
};

export type PermissionFileInfo = {
  path: string;
  movePath?: string;
  type: string;
  typeLabel: string;
  additions: number;
  deletions: number;
  baseHash?: string;
  currentHash?: string;
  stale?: boolean;
};

export type PermissionProjectInfo = {
  operation: "add" | "associate";
  name?: string;
  rootPath: string;
  immutableAssociation: boolean;
};

export type PermissionMCPRegistrationInfo = {
  id: string;
  name: string;
  transport: string;
  target: string;
  cwd?: string;
  roots: string[];
  auth: string;
  bearerTokenEnv?: string;
  global: boolean;
};

export function permissionMCPRegistration(
  permission: PermissionRequest,
): PermissionMCPRegistrationInfo | null {
  const args = permission.arguments;
  if (!args || args.registrationKind !== "mcp") return null;
  const id = args.registrationServerId;
  const name = args.registrationName;
  const transport = args.registrationTransport;
  const target = args.registrationTarget;
  if (
    typeof id !== "string" ||
    typeof name !== "string" ||
    typeof transport !== "string" ||
    typeof target !== "string" ||
    !id ||
    !name ||
    !transport ||
    !target
  ) {
    return null;
  }
  return {
    id,
    name,
    transport,
    target,
    cwd:
      typeof args.registrationCwd === "string" && args.registrationCwd
        ? args.registrationCwd
        : undefined,
    roots: Array.isArray(args.registrationRoots)
      ? args.registrationRoots.filter(
          (root): root is string => typeof root === "string" && Boolean(root),
        )
      : [],
    auth:
      typeof args.registrationAuth === "string"
        ? args.registrationAuth
        : "none",
    bearerTokenEnv:
      typeof args.registrationBearerTokenEnv === "string"
        ? args.registrationBearerTokenEnv
        : undefined,
    global: args.registrationGlobal === true,
  };
}

export function permissionProject(
  permission: PermissionRequest,
): PermissionProjectInfo | null {
  const args = permission.arguments;
  if (!args) return null;
  const operation = args.projectOperation;
  const rootPath = args.projectRoot;
  if (
    (operation !== "add" && operation !== "associate") ||
    typeof rootPath !== "string" ||
    !rootPath
  ) {
    return null;
  }
  const project: PermissionProjectInfo = {
    operation,
    rootPath,
    immutableAssociation: args.immutableAssociation === true,
  };
  if (typeof args.projectName === "string") project.name = args.projectName;
  return project;
}

export function permissionCommand(
  permission: PermissionRequest,
): PermissionCommandInfo | null {
  const args = permission.arguments;
  if (!args) return null;
  const command =
    typeof args.command === "string"
      ? args.command
      : typeof args.normalizedCommand === "string"
        ? args.normalizedCommand
        : "";
  if (!command.trim()) return null;
  return {
    command,
    cwd: typeof args.cwd === "string" ? args.cwd : undefined,
    riskLevel: typeof args.riskLevel === "string" ? args.riskLevel : undefined,
    category: typeof args.category === "string" ? args.category : undefined,
    networkHint:
      typeof args.networkHint === "string" ? args.networkHint : undefined,
  };
}

export function permissionAgentMode(permission: PermissionRequest) {
  const mode = permission.arguments?.agentMode;
  return typeof mode === "string" ? mode : "";
}

export function permissionApprovalTitle(
  permission: PermissionRequest,
  command: PermissionCommandInfo | null,
) {
  if (command) {
    return permission.toolName === "run_tests"
      ? "批准测试命令"
      : "批准命令执行";
  }
  if (writePermissionToolNames.has(permission.toolName)) {
    return "批准文件修改";
  }
  if (permission.toolName === "aivo_projects_add") {
    return "批准添加项目";
  }
  if (permission.toolName === "aivo_projects_associate") {
    return "批准关联项目";
  }
  if (permissionMCPRegistration(permission)) {
    return "批准注册 MCP 工具";
  }
  return `批准 ${permission.toolName}`;
}

export function permissionApprovalTarget(
  permission: PermissionRequest,
  command: PermissionCommandInfo | null,
) {
  if (command) {
    return command.command;
  }
  const project = permissionProject(permission);
  if (project) return project.rootPath;
  const registration = permissionMCPRegistration(permission);
  if (registration) return registration.target;
  return permission.paths?.length ? permission.paths.join(", ") : permission.action;
}

export function permissionRememberLabel(
  permission: PermissionRequest,
  command: PermissionCommandInfo | null,
) {
  if (command) return "记住此命令和 cwd";
  if (permissionProject(permission)) return "记住此项目操作";
  if (permissionMCPRegistration(permission)) return "此注册必须每次单独确认";
  return "记住这类权限";
}

export function permissionToolsets(permission: PermissionRequest) {
  const toolsets = permission.arguments?.toolsets;
  if (!Array.isArray(toolsets)) return [];
  return toolsets.filter(
    (toolset): toolset is string => typeof toolset === "string",
  );
}

export function permissionFiles(
  permission: PermissionRequest,
): PermissionFileInfo[] {
  const rawFiles = permission.arguments?.files;
  if (!Array.isArray(rawFiles)) return [];
  return rawFiles.flatMap((file) => {
    if (!file || typeof file !== "object") return [];
    const record = file as Record<string, unknown>;
    const path = typeof record.path === "string" ? record.path : "";
    if (!path) return [];
    const type = typeof record.type === "string" ? record.type : "update";
    return [
      {
        path,
        movePath:
          typeof record.movePath === "string" ? record.movePath : undefined,
        type,
        typeLabel: permissionFileTypeLabel(type),
        additions: typeof record.additions === "number" ? record.additions : 0,
        deletions: typeof record.deletions === "number" ? record.deletions : 0,
        baseHash:
          typeof record.baseHash === "string" ? record.baseHash : undefined,
        currentHash:
          typeof record.currentHash === "string"
            ? record.currentHash
            : undefined,
        stale: record.stale === true,
      },
    ];
  });
}

function permissionFileTypeLabel(type: string) {
  switch (type) {
    case "add":
      return "A";
    case "delete":
      return "D";
    case "move":
      return "R";
    default:
      return "M";
  }
}
