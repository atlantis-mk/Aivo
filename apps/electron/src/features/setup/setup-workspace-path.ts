type WorkspacePathConfig = {
  initialWorkspacePath?: string;
  defaultInitialWorkspacePath?: string;
};

export function resolveSetupWorkspacePath(config: WorkspacePathConfig | null) {
  const savedPath = config?.initialWorkspacePath?.trim() ?? "";
  const defaultPath = config?.defaultInitialWorkspacePath?.trim() ?? "";
  if (!savedPath) return defaultPath;
  if (!defaultPath) return savedPath;

  const legacyDefaultPath = defaultPath.replace(
    /([\\/])Aivo-Workspaces$/,
    "$1Aivo Workspaces",
  );
  return savedPath === legacyDefaultPath ? defaultPath : savedPath;
}
