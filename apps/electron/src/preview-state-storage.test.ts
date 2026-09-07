import assert from "node:assert/strict";
import { test } from "node:test";

import {
  migratePreviewStateStorage,
  PREVIEW_STATE_STORAGE_VERSION,
} from "./preview-state-storage.ts";

test("invalidates provider state while preserving unrelated Aivo preferences", () => {
  const migrated = migratePreviewStateStorage({
    auth: { openai: [{ type: "oauth-browser" }] },
    config: {
      appName: "My Aivo",
      auxiliaryModel: { modelId: "secondary", providerId: "provider" },
      defaultModel: { modelId: "primary", providerId: "provider" },
      initialized: true,
      initialWorkspacePath: "/workspace",
      provider: { id: "provider" },
      providers: { custom: { provider: { id: "provider" } }, disabled: ["other"] },
      reasoningEffort: "high",
    },
    pendingAuth: { status: "pending" },
  });

  assert.equal(migrated.storageVersion, PREVIEW_STATE_STORAGE_VERSION);
  assert.deepEqual(migrated.auth, {});
  assert.equal(migrated.pendingAuth, null);
  assert.deepEqual(migrated.config, {
    appName: "My Aivo",
    auxiliaryModel: undefined,
    defaultModel: undefined,
    initialized: false,
    initialWorkspacePath: "/workspace",
    provider: undefined,
    providers: { custom: {}, disabled: [] },
    reasoningEffort: "high",
  });
});

test("leaves current preview state unchanged", () => {
  const state = {
    auth: { provider: [{ type: "api-key" }] },
    storageVersion: PREVIEW_STATE_STORAGE_VERSION,
  };

  assert.equal(migratePreviewStateStorage(state), state);
});
