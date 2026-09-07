import assert from "node:assert/strict";
import { describe, it } from "node:test";

import type { ProviderInfo } from "@/lib/provider-catalog";
import type { domain } from "@/types/codex-domain";

import {
  getActiveProvider,
  getDefaultModelId,
} from "./project-model-options.ts";

const providers: ProviderInfo[] = [
  { id: "provider-a", name: "Provider A" } as ProviderInfo,
  { id: "provider-b", name: "Provider B" } as ProviderInfo,
];

describe("getActiveProvider", () => {
  it("prefers the last selected model's provider over the setup provider", () => {
    const config = {
      provider: { id: "provider-a", model: "model-a" },
      defaultModel: { providerId: "provider-b", modelId: "model-b" },
    } as unknown as domain.AppConfig;

    assert.equal(getActiveProvider(config, providers)?.id, "provider-b");
  });

  it("keeps an explicit in-session provider selection above all", () => {
    const config = {
      provider: { id: "provider-a", model: "model-a" },
      defaultModel: { providerId: "provider-b", modelId: "model-b" },
    } as unknown as domain.AppConfig;

    assert.equal(
      getActiveProvider(config, providers, "provider-a")?.id,
      "provider-a",
    );
  });

  it("falls back to the setup provider when no model was ever selected", () => {
    const config = {
      provider: { id: "provider-a", model: "model-a" },
    } as unknown as domain.AppConfig;

    assert.equal(getActiveProvider(config, providers)?.id, "provider-a");
  });
});

describe("getDefaultModelId", () => {
  it("prefers the persisted default model over the setup model", () => {
    const provider = {
      id: "provider-a",
      name: "Provider A",
      defaultModelId: "fallback-model",
    } as ProviderInfo;
    const config = {
      provider: { id: "provider-a", model: "setup-model" },
      defaultModel: { providerId: "provider-a", modelId: "selected-model" },
    } as unknown as domain.AppConfig;

    assert.equal(getDefaultModelId(config, provider, []), "selected-model");
  });
});
