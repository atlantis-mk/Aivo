import assert from "node:assert/strict";
import test from "node:test";

import {
  agentModeModelsForProvider,
  agentModeSubagentCandidates,
  connectedAgentModeProviders,
} from "../src/features/projects/extension-settings-agent-mode-model.ts";
import type { CatalogState } from "../src/lib/provider-catalog-types.ts";

const catalog: CatalogState = {
  connected: ["openai", "custom"],
  connectedProviders: [],
  models: [
    { id: "custom-1", name: "Custom 1", providerId: "custom" },
    { id: "custom-1", name: "Duplicate", providerId: "custom" },
  ],
  providers: [
    {
      authMethods: [],
      builtIn: true,
      connected: true,
      custom: false,
      id: "openai",
      models: [
        { id: "gpt-5.5", name: "GPT-5.5", providerId: "openai" },
      ],
      name: "OpenAI",
      type: "openai",
    },
    {
      authMethods: [],
      builtIn: true,
      connected: false,
      custom: false,
      id: "anthropic",
      models: [],
      name: "Anthropic",
      type: "anthropic",
    },
    {
      authMethods: [],
      builtIn: false,
      connected: false,
      custom: true,
      id: "custom",
      models: [],
      name: "Custom",
      type: "openai-compatible",
    },
  ],
};

test("agent mode providers include only connected providers", () => {
  assert.deepEqual(
    connectedAgentModeProviders(catalog).map((provider) => provider.id),
    ["openai", "custom"],
  );
});

test("agent mode models are scoped to the selected connected provider", () => {
  assert.deepEqual(
    agentModeModelsForProvider(catalog, "openai").map((model) => model.id),
    ["gpt-5.5"],
  );
  assert.deepEqual(
    agentModeModelsForProvider(catalog, "custom").map((model) => model.id),
    ["custom-1"],
  );
  assert.deepEqual(agentModeModelsForProvider(catalog, "anthropic"), []);
});

test("subagent candidates exclude the owner, hidden, and primary-only modes", () => {
  assert.deepEqual(
    agentModeSubagentCandidates(
      [
        { id: "orchestrator", displayName: "Orchestrator", description: "", prompt: "", mode: "primary" },
        { id: "review", displayName: "Review", description: "", prompt: "", mode: "subagent" },
        { id: "explore", displayName: "Explore", description: "", prompt: "", mode: "all" },
        { id: "primary", displayName: "Primary", description: "", prompt: "", mode: "primary" },
        { id: "hidden", displayName: "Hidden", description: "", prompt: "", hidden: true },
      ],
      "orchestrator",
    ).map((mode) => mode.id),
    ["review", "explore"],
  );
});
