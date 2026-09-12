import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  credentialReferenceFor,
  defaultBaseURLForProvider,
  knownDefaultModelForProvider,
  primaryDefaultModelForProvider,
  providerProtocolForProvider,
} from "./provider-defaults.ts";
import {
  fallbackPopularProviders,
  fallbackProviders,
} from "../../lib/provider-catalog-fallback-providers.ts";

describe("DeepSeek provider", () => {
  it("uses the native Responses API defaults", () => {
    assert.equal(providerProtocolForProvider("deepseek"), "responses");
    assert.equal(
      defaultBaseURLForProvider("deepseek"),
      "https://api.deepseek.com",
    );
    assert.equal(knownDefaultModelForProvider("deepseek"), "deepseek-v4-flash");
    assert.equal(
      primaryDefaultModelForProvider("deepseek"),
      "deepseek-v4-flash",
    );
    assert.equal(credentialReferenceFor("deepseek"), "DEEPSEEK_API_KEY");
  });

  it("exposes the current V4 models and vision capability", () => {
    const provider = fallbackProviders().find(
      (candidate) => candidate.id === "deepseek",
    );
    assert.ok(provider);
    assert.equal(provider.type, "responses");
    assert.equal(provider.defaultModelId, "deepseek-v4-flash");
    assert.deepEqual(
      provider.models.map((model) => model.id),
      ["deepseek-v4-flash", "deepseek-v4-pro", "deepseek-v4-flash-vision-exp"],
    );
    assert.deepEqual(provider.models.at(-1)?.modalities, ["text", "image"]);
    assert.ok(
      fallbackPopularProviders().some(
        (candidate) => candidate.id === "deepseek",
      ),
    );
  });
});
