import assert from "node:assert/strict";
import test from "node:test";

import type { ConversationTurn } from "../src/features/projects/conversation-timeline-model.ts";
import {
  deriveSessionRuntimeStats,
  formatDuration,
  formatSessionRuntimeStats,
  formatSessionRuntimeStatsValue,
  formatTokens,
  runtimeMetricsFromEventPayload,
  type TurnRuntimeMetrics,
} from "../src/features/projects/project-session-runtime-stats.ts";

function conversationTurn(runtimeMetrics?: TurnRuntimeMetrics): ConversationTurn {
  return {
    activityVisible: false,
    id: crypto.randomUUID(),
    prompt: "hello",
    preToolText: "",
    responseText: "hi",
    toolCalls: [],
    startedAt: 0,
    submittedAt: new Date(0),
    thinkingSeconds: 0,
    responseCompletedAt: new Date(1_000),
    responseVisible: true,
    runtimeMetrics,
    stopped: false,
  };
}

test("runtime metrics payload parsing validates pairs and clamps cache reads", () => {
  assert.deepEqual(
    runtimeMetricsFromEventPayload({
      runtimeMetrics: {
        steps: 2,
        llmMs: 2_500,
        ttftMs: 1_200,
        ttftSteps: 2,
        decodeMs: 3_000,
        decodeTokens: 40,
        inputTokens: 100,
        outputTokens: 40,
        cacheReadTokens: 120,
      },
    }),
    {
      steps: 2,
      llmMs: 2_500,
      ttftMs: 1_200,
      ttftSteps: 2,
      decodeMs: 3_000,
      decodeTokens: 40,
      inputTokens: 100,
      outputTokens: 40,
      cacheReadTokens: 100,
      cacheReadAvailable: true,
    },
  );
  assert.equal(
    runtimeMetricsFromEventPayload({ runtimeMetrics: { steps: -1, llmMs: 1 } }),
    undefined,
  );
  assert.deepEqual(
    runtimeMetricsFromEventPayload({
      runtimeMetrics: {
        steps: 1,
        llmMs: 10,
        inputTokens: 10,
        outputTokens: 2,
        cacheReadAvailable: true,
      },
    }),
    {
      steps: 1,
      llmMs: 10,
      inputTokens: 10,
      outputTokens: 2,
      cacheReadTokens: 0,
      cacheReadAvailable: true,
    },
  );
});

test("session runtime stats count turns once and sum LLM steps", () => {
  const turns = [
    conversationTurn({
      steps: 2,
      llmMs: 2_000,
      ttftMs: 800,
      ttftSteps: 1,
      decodeMs: 2_000,
      decodeTokens: 30,
      inputTokens: 90,
      outputTokens: 30,
      cacheReadTokens: 80,
      cacheReadAvailable: true,
    }),
    conversationTurn({
      steps: 1,
      llmMs: 500,
      ttftMs: 400,
      ttftSteps: 1,
      decodeMs: 1_000,
      decodeTokens: 10,
      inputTokens: 10,
      outputTokens: 10,
      cacheReadTokens: 10,
      cacheReadAvailable: true,
    }),
    conversationTurn(),
  ];

  assert.deepEqual(deriveSessionRuntimeStats(turns), {
    turns: 2,
    steps: 3,
    llmMs: 2_500,
    ttftMs: 1_200,
    ttftSteps: 2,
    decodeMs: 3_000,
    decodeTokens: 40,
    inputTokens: 100,
    outputTokens: 40,
    cacheReadTokens: 90,
    cacheReadAvailable: true,
  });
  assert.equal(
    formatSessionRuntimeStats(turns),
    "2 轮 · 3 步 | LLM 2.5s | 首 token 平均 0.6s · 13 tok/s | 缓存命中 90% | 输入 100 tok · 输出 40 tok",
  );
});

test("session runtime stats omit unavailable groups and compact values", () => {
  assert.equal(
    formatSessionRuntimeStats([
      conversationTurn({ steps: 1, llmMs: 0 }),
    ]),
    "1 轮 · 1 步",
  );
  assert.equal(formatTokens(17_200), "17.2K");
  assert.equal(formatTokens(2_400_000), "2.4M");
  assert.equal(formatDuration(162_000), "2m42s");
  assert.equal(
    formatSessionRuntimeStatsValue({
      turns: 1,
      steps: 1,
      llmMs: 1_100,
      ttftMs: 700,
      ttftSteps: 1,
      decodeMs: 385,
      decodeTokens: 50,
      inputTokens: 9_400,
      outputTokens: 50,
      cacheReadTokens: 0,
      cacheReadAvailable: true,
    }),
    "1 轮 · 1 步 | LLM 1.1s | 首 token 平均 0.7s · 130 tok/s | 缓存命中 0% | 输入 9.4K tok · 输出 50 tok",
  );
  assert.equal(
    formatSessionRuntimeStatsValue({
      turns: 1,
      steps: 1,
      llmMs: 100,
      inputTokens: 10,
      outputTokens: 2,
    }),
    "1 轮 · 1 步 | LLM 0.1s | 输入 10 tok · 输出 2 tok",
  );
  assert.equal(
    formatSessionRuntimeStats([
      conversationTurn({
        steps: 1,
        llmMs: 100,
        inputTokens: 10,
        outputTokens: 2,
        cacheReadTokens: 5,
        cacheReadAvailable: true,
      }),
      conversationTurn({ steps: 1, llmMs: 100, inputTokens: 10, outputTokens: 2 }),
    ]),
    "2 轮 · 2 步 | LLM 0.2s | 输入 20 tok · 输出 4 tok",
  );
});
