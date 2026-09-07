export const PREVIEW_STATE_STORAGE_VERSION = 1;

interface VersionedPreviewState {
  auth?: object;
  config?: object;
  pendingAuth?: unknown;
  storageVersion?: number;
}

export function migratePreviewStateStorage<T extends VersionedPreviewState>(
  state: T,
): T & { storageVersion: number } {
  if (state.storageVersion === PREVIEW_STATE_STORAGE_VERSION) {
    return state as T & { storageVersion: number };
  }

  const config = state.config
    ? {
        ...state.config,
        initialized: false,
        provider: undefined,
        providers: { custom: {}, disabled: [] },
        defaultModel: undefined,
        auxiliaryModel: undefined,
      }
    : undefined;
  return {
    ...state,
    auth: {},
    config,
    pendingAuth: null,
    storageVersion: PREVIEW_STATE_STORAGE_VERSION,
  } as unknown as T & { storageVersion: number };
}
