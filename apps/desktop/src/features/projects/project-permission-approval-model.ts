import type { PermissionRequest } from "@/services/aivo";

export type PermissionActionState =
  "idle" | "approving" | "denying" | "approved" | "denied";

const writePermissionToolNames = new Set([
  "apply_patch",
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
  return `批准 ${permission.toolName}`;
}

export function permissionApprovalTarget(
  permission: PermissionRequest,
  command: PermissionCommandInfo | null,
) {
  if (command) {
    return command.command;
  }
  return permission.paths?.length ? permission.paths.join(", ") : permission.action;
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
