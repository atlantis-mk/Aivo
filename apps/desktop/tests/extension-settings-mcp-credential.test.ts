import assert from "node:assert/strict";
import test from "node:test";

import {
  applyGeneratedMcpDescription,
  canSaveMcpDraft,
  mcpServerToDraft,
  normalizeMcpDraft,
} from "../src/features/projects/extension-settings-mcp-draft.ts";

test("generated MCP description changes only the unsaved draft description", () => {
  const draft = {
    id: "linear",
    name: "linear",
    description: "原描述",
    transport: "streamable_http" as const,
    url: "https://private.example.test/mcp",
    headers: { Authorization: "secret" },
    enabled: true,
  };

  const generated = applyGeneratedMcpDescription(
    draft,
    "查询、创建和更新 Linear 中的 issue、项目与团队信息",
  );

  assert.notEqual(generated, draft);
  assert.equal(generated.description, "查询、创建和更新 Linear 中的 issue、项目与团队信息");
  assert.deepEqual({ ...generated, description: draft.description }, draft);
});

test("direct bearer requires a new token or an existing secure reference", () => {
  const direct = {
    id: "remote",
    name: "remote",
    description: "访问远程 MCP 能力",
    transport: "streamable_http" as const,
    url: "https://mcp.example.test",
    authType: "bearer" as const,
    bearerAuthMode: "direct" as const,
    enabled: true,
  };

  assert.equal(canSaveMcpDraft(direct), false);
  assert.equal(canSaveMcpDraft({ ...direct, bearerToken: "secret" }), true);
  assert.equal(
    canSaveMcpDraft({ ...direct, bearerTokenRef: "mcp-auth/remote/access-token" }),
    true,
  );
});

test("bearer mode normalization keeps only the selected credential source", () => {
  const direct = normalizeMcpDraft({
    id: "remote",
    name: "remote",
    description: "访问远程 MCP 能力",
    transport: "streamable_http",
    url: "https://mcp.example.test",
    authType: "bearer",
    bearerAuthMode: "direct",
    bearerToken: "secret",
    bearerTokenEnv: "MCP_TOKEN",
    enabled: true,
  });
  assert.equal(direct.bearerToken, "secret");
  assert.equal(direct.bearerTokenEnv, "");

  const environment = normalizeMcpDraft({
    ...direct,
    bearerAuthMode: "env",
    bearerTokenEnv: "MCP_TOKEN",
  });
  assert.equal(environment.bearerToken, "");
  assert.equal(environment.bearerTokenEnv, "MCP_TOKEN");
});

test("editing a reference-backed bearer source starts direct mode without a token", () => {
  const draft = mcpServerToDraft({
    id: "remote",
    name: "remote",
    description: "访问远程 MCP 能力",
    transport: "streamable_http",
    url: "https://mcp.example.test",
    authType: "bearer",
    bearerTokenRef: "mcp-auth/remote/access-token",
    enabled: true,
  });

  assert.equal(draft.bearerAuthMode, "direct");
  assert.equal(draft.bearerToken, undefined);
  assert.equal(canSaveMcpDraft(draft), true);
});

test("MCP creation allows a blank functional description", () => {
  const draft = {
    id: "filesystem",
    name: "filesystem",
    transport: "stdio" as const,
    command: "npx",
    enabled: true,
  };

  assert.equal(canSaveMcpDraft(draft), true);
  assert.equal(
    canSaveMcpDraft({ ...draft, description: "读取和管理指定目录中的文件" }),
    true,
  );
  assert.equal(
    canSaveMcpDraft({ ...draft, description: "a".repeat(501) }),
    false,
  );
});
