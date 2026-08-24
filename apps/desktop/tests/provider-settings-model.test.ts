import assert from "node:assert/strict";
import test from "node:test";

import {
  configuredProviderRefreshInput,
  configuredProviders,
  providerCanRefreshModels,
  providerConnectionMethodLabel,
  providerModelLabel,
  providerRefreshUnavailableMessage,
  providerReadinessLabel,
} from "../src/features/providers/provider-settings-model.ts";

test("provider settings lists only configured providers with the default first", () => {
  const visible = configuredProviders(
    [
      { id: "idle", name: "Idle" },
      {
        id: "anthropic",
        name: "Anthropic",
        auth: { connected: true, type: "api-key" },
      },
      {
        id: "openai",
        name: "OpenAI",
        accounts: [{ id: "account", method: "oauth-browser" }],
      },
    ],
    "openai",
  );

  assert.deepEqual(
    visible.map((provider) => provider.id),
    ["openai", "anthropic"],
  );
});

test("provider settings refreshes only refreshable catalogs with a secret-free input", () => {
  const provider = {
    id: "team-provider",
    name: "Team Provider",
    connected: true,
    modelRefresh: { refreshable: true },
  };

  assert.equal(providerCanRefreshModels(provider), true);
  assert.equal(
    providerCanRefreshModels({
      ...provider,
      modelRefresh: { refreshable: false },
    }),
    false,
  );
  assert.equal(providerRefreshUnavailableMessage(provider), "");
  assert.equal(
    providerRefreshUnavailableMessage({
      ...provider,
      modelRefresh: { refreshable: false },
    }),
    "Team Provider 暂不支持远程模型刷新，请使用“更新模型目录”获取最新的内置模型列表。",
  );
  assert.deepEqual(configuredProviderRefreshInput(provider), {
    providerId: "team-provider",
    name: "Team Provider",
  });
});

test("provider settings derives safe model, auth, and readiness summaries", () => {
  const provider = {
    id: "openai",
    name: "OpenAI",
    connected: true,
    defaultModelId: "gpt-5.5",
    models: [{ id: "gpt-5.5", name: "GPT 5.5" }],
    accounts: [
      { id: "browser", method: "oauth-browser" },
      { id: "key", method: "api-key" },
    ],
    readiness: { ready: true },
  };

  assert.equal(providerModelLabel(provider), "GPT 5.5");
  assert.equal(
    providerConnectionMethodLabel(provider),
    "浏览器 OAuth、API Key",
  );
  assert.equal(providerReadinessLabel(provider), "可用");
});

test("provider settings keeps catalog failure reasons actionable", () => {
  assert.equal(
    providerReadinessLabel({
      id: "custom",
      name: "Custom",
      connected: true,
      readiness: { ready: false, reason: "missing model" },
    }),
    "missing model",
  );
});
