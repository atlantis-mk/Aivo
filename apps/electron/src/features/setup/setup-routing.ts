type InitializationConfig = {
  initialized?: boolean;
  initialWorkspacePath?: string;
};

export function hasCompletedInitialization(
  config: InitializationConfig | null | undefined,
) {
  return Boolean(
    config?.initialized && config.initialWorkspacePath?.trim(),
  );
}

export function startupRouteFor(
  config: InitializationConfig | null | undefined,
) {
  return hasCompletedInitialization(config) ? "/projects/chat" : "/setup";
}
