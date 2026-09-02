import assert from "node:assert/strict";
import test from "node:test";

import {
  defaultBaseURLForProvider,
  knownDefaultModelForProvider,
  providerProtocolForProvider,
} from "../src/features/providers/provider-defaults.ts";

test("provider defaults align Responses templates with backend-specific providers", () => {
  for (const providerId of ["deepseek", "groq", "xai", "xiaomi"]) {
    assert.equal(providerProtocolForProvider(providerId), "openai");
  }
  assert.equal(providerProtocolForProvider("openrouter"), "openrouter");
  assert.equal(providerProtocolForProvider("deepinfra"), "openai-compatible");
});

test("provider defaults include Xiaomi MiMo setup values", () => {
  assert.equal(
    defaultBaseURLForProvider("xiaomi"),
    "https://api.xiaomimimo.com/v1",
  );
  assert.equal(knownDefaultModelForProvider("xiaomi"), "mimo-v2.5-pro");
});

test("provider defaults keep backend-specific fallback models aligned", () => {
  assert.equal(knownDefaultModelForProvider("xai"), "grok-4.3");
  assert.equal(knownDefaultModelForProvider("groq"), "openai/gpt-oss-120b");
  assert.equal(
    knownDefaultModelForProvider("deepinfra"),
    "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo",
  );
  assert.equal(
    knownDefaultModelForProvider("fireworks-ai"),
    "accounts/fireworks/models/glm-5p2",
  );
  assert.equal(defaultBaseURLForProvider("togetherai"), "https://api.together.ai/v1");
  assert.equal(knownDefaultModelForProvider("togetherai"), "MiniMaxAI/MiniMax-M3");
  assert.equal(defaultBaseURLForProvider("inference"), "https://api.inference.net/v1");
  assert.equal(knownDefaultModelForProvider("inference"), "glm-5.2");
  assert.equal(knownDefaultModelForProvider("tencent-coding-plan"), "kimi-k2.6");
  assert.equal(knownDefaultModelForProvider("scaleway"), "qwen3.5-397b-a17b");
  assert.equal(knownDefaultModelForProvider("stackit"), "Qwen/Qwen3.6-27B");
  assert.equal(knownDefaultModelForProvider("vultr"), "kimi-k2-instruct");
  assert.equal(knownDefaultModelForProvider("github-models"), "github-models-retired");
  assert.equal(defaultBaseURLForProvider("cloudferro-sherlock"), "https://api-sherlock.cloudferro.com/openai/v1");
  assert.equal(knownDefaultModelForProvider("helicone"), "gpt-4o-mini");
  assert.equal(knownDefaultModelForProvider("ovhcloud"), "gpt-oss-20b");
  assert.equal(knownDefaultModelForProvider("upstage"), "solar-pro4");
  assert.equal(providerProtocolForProvider("perplexity-agent"), "openai");
  assert.equal(knownDefaultModelForProvider("perplexity-agent"), "openai/gpt-5.6-terra");
  assert.equal(knownDefaultModelForProvider("poe"), "GPT-5.4");
  assert.equal(providerProtocolForProvider("vivgrid"), "openai-compatible");
  assert.equal(knownDefaultModelForProvider("vivgrid"), "gpt-5.6-terra");
  assert.equal(
    knownDefaultModelForProvider("requesty"),
    "anthropic/claude-sonnet-4-20250514",
  );
  assert.equal(knownDefaultModelForProvider("nebius"), "moonshotai/Kimi-K2.5");
  assert.equal(knownDefaultModelForProvider("wandb"), "openai/gpt-oss-20b");
  assert.equal(knownDefaultModelForProvider("venice"), "zai-org-glm-5");
});
