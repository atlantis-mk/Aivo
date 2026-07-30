import { invoke } from "@/services/aivo/invoke";

export type GitWorktree = {
  id: string;
  repositoryRoot: string;
  path: string;
  branch?: string;
  baseRef?: string;
  head?: string;
  status: "ready" | "missing" | "removed" | "error";
  managed: boolean;
  ownsBranch?: boolean;
  detached?: boolean;
  dirty?: boolean;
  activeSessions?: string[];
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export type CreateGitWorktreeInput = {
  repositoryPath: string;
  path?: string;
  approvedRoot?: string;
  name?: string;
  branch?: string;
  baseRef?: string;
  detached?: boolean;
  startupCommand?: string;
  startupConfirmed?: boolean;
  sessionId?: string;
};

export function createGitWorktree(input: CreateGitWorktreeInput) {
  return invoke<GitWorktree>("CreateGitWorktree", input);
}

export function listGitWorktrees(
  repositoryPath?: string,
  includeRemoved = false,
) {
  return invoke<GitWorktree[]>("ListGitWorktrees", {
    repositoryPath,
    includeRemoved,
  });
}

export function resetGitWorktree(
  worktreeId: string,
  options: { targetRef?: string; clean?: boolean; confirmed: boolean },
) {
  return invoke<GitWorktree>("ResetGitWorktree", {
    worktreeId,
    ...options,
  });
}

export function removeGitWorktree(
  worktreeId: string,
  options: { force?: boolean; deleteBranch?: boolean; confirmed: boolean },
) {
  return invoke<GitWorktree>("RemoveGitWorktree", {
    worktreeId,
    ...options,
  });
}

export function bindSessionToGitWorktree(
  sessionId: string,
  worktreeId: string,
) {
  return invoke("BindSessionToGitWorktree", { sessionId, worktreeId });
}
