import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";

export type TurnRuntimeMetrics = {
  steps: number;
  llmMs: number;
  ttftMs?: number;
  ttftSteps?: number;
  decodeMs?: number;
  decodeTokens?: number;
  inputTokens?: number;
  outputTokens?: number;
  cacheReadTokens?: number;
  cacheReadAvailable?: boolean;
};

export type SessionRuntimeStats = TurnRuntimeMetrics & {
  turns: number;
};

export function runtimeMetricsFromEventPayload(
  payload: Record<string, unknown> | undefined,
): TurnRuntimeMetrics | undefined {
  const raw = payload?.["runtimeMetrics"];
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const metrics = raw as Record<string, unknown>;
  const steps = nonNegativeInteger(metrics["steps"]);
  const llmMs = nonNegativeInteger(metrics["llmMs"]);
  if (steps === undefined || steps < 1 || llmMs === undefined) return undefined;

  const result: TurnRuntimeMetrics = { steps, llmMs };
  assignPair(result, metrics, "ttftMs", "ttftSteps");
  assignPair(result, metrics, "decodeMs", "decodeTokens");

  const inputTokens = nonNegativeInteger(metrics["inputTokens"]);
  const outputTokens = nonNegativeInteger(metrics["outputTokens"]);
  if (inputTokens !== undefined && outputTokens !== undefined) {
    result.inputTokens = inputTokens;
    result.outputTokens = outputTokens;
    const cacheReadTokens = nonNegativeInteger(metrics["cacheReadTokens"]);
    const explicitlyAvailable = metrics["cacheReadAvailable"] === true;
    if (explicitlyAvailable || (cacheReadTokens ?? 0) > 0) {
      result.cacheReadAvailable = true;
      result.cacheReadTokens = Math.min(inputTokens, cacheReadTokens ?? 0);
    }
  }
  return result;
}

function assignPair(
  target: TurnRuntimeMetrics,
  source: Record<string, unknown>,
  first: "ttftMs" | "decodeMs",
  second: "ttftSteps" | "decodeTokens",
) {
  const firstValue = nonNegativeInteger(source[first]);
  const secondValue = nonNegativeInteger(source[second]);
  if (firstValue === undefined || secondValue === undefined) return;
  target[first] = firstValue;
  target[second] = secondValue;
}

function nonNegativeInteger(value: unknown) {
  return typeof value === "number" &&
    Number.isSafeInteger(value) &&
    value >= 0
    ? value
    : undefined;
}

export function deriveSessionRuntimeStats(
  turns: ConversationTurn[],
): SessionRuntimeStats | undefined {
  const settled = turns.flatMap((turn) =>
    turn.runtimeMetrics ? [turn.runtimeMetrics] : [],
  );
  if (settled.length === 0) return undefined;

  const stats: SessionRuntimeStats = {
    turns: settled.length,
    steps: 0,
    llmMs: 0,
  };
  let usageGroups = 0;
  let cacheUsageGroups = 0;
  for (const metrics of settled) {
    stats.steps += metrics.steps;
    stats.llmMs += metrics.llmMs;
    addOptional(stats, metrics, "ttftMs");
    addOptional(stats, metrics, "ttftSteps");
    addOptional(stats, metrics, "decodeMs");
    addOptional(stats, metrics, "decodeTokens");
    addOptional(stats, metrics, "inputTokens");
    addOptional(stats, metrics, "outputTokens");
    addOptional(stats, metrics, "cacheReadTokens");
    if (metrics.inputTokens !== undefined && metrics.outputTokens !== undefined) {
      usageGroups += 1;
      if (metrics.cacheReadAvailable === true) cacheUsageGroups += 1;
    }
  }
  if (usageGroups > 0 && cacheUsageGroups === usageGroups) {
    stats.cacheReadAvailable = true;
  } else {
    delete stats.cacheReadTokens;
  }
  return stats;
}

function addOptional(
  target: TurnRuntimeMetrics,
  source: TurnRuntimeMetrics,
  key: Exclude<keyof TurnRuntimeMetrics, "steps" | "llmMs" | "cacheReadAvailable">,
) {
  const value = source[key];
  if (value === undefined) return;
  target[key] = (target[key] ?? 0) + value;
}

export function formatSessionRuntimeStats(turns: ConversationTurn[]) {
  return formatSessionRuntimeStatsValue(deriveSessionRuntimeStats(turns));
}

export function formatSessionRuntimeStatsValue(
  stats: SessionRuntimeStats | undefined,
) {
  if (!stats || stats.turns < 1 || stats.steps < 1) return "";

  const groups = [`${stats.turns} 轮 · ${stats.steps} 步`];
  if (stats.llmMs > 0) groups.push(`LLM ${formatDuration(stats.llmMs)}`);

  const speeds: string[] = [];
  if ((stats.ttftSteps ?? 0) > 0) {
    speeds.push(
      `首 token 平均 ${formatDuration((stats.ttftMs ?? 0) / (stats.ttftSteps ?? 1))}`,
    );
  }
  if ((stats.decodeMs ?? 0) > 0 && stats.decodeTokens !== undefined) {
    const throughput = stats.decodeTokens / ((stats.decodeMs ?? 0) / 1_000);
    speeds.push(`${formatTokensPerSecond(throughput)} tok/s`);
  }
  if (speeds.length > 0) groups.push(speeds.join(" · "));

  if (stats.cacheReadAvailable === true && (stats.inputTokens ?? 0) > 0) {
    groups.push(
      `缓存命中 ${formatCacheHitPercent(stats.cacheReadTokens ?? 0, stats.inputTokens ?? 0)}%`,
    );
  }
  if ((stats.inputTokens ?? 0) > 0 || (stats.outputTokens ?? 0) > 0) {
    groups.push(
      `输入 ${formatTokens(stats.inputTokens ?? 0)} tok · 输出 ${formatTokens(stats.outputTokens ?? 0)} tok`,
    );
  }
  return groups.join(" | ");
}

export function formatTokens(tokens: number) {
  const scaled = (value: number) =>
    value >= 100
      ? String(Math.round(value))
      : String(Math.round(value * 10) / 10);
  if (tokens < 1_000) return String(tokens);
  if (tokens < 1_000_000) return `${scaled(tokens / 1_000)}K`;
  return `${scaled(tokens / 1_000_000)}M`;
}

export function formatDuration(milliseconds: number) {
  const seconds = milliseconds / 1_000;
  if (seconds < 60) return `${Math.round(seconds * 10) / 10}s`;
  const wholeSeconds = Math.round(seconds);
  return `${Math.floor(wholeSeconds / 60)}m${wholeSeconds % 60}s`;
}

function formatTokensPerSecond(tokensPerSecond: number) {
  const clamped = Math.max(0, tokensPerSecond);
  return clamped >= 10
    ? String(Math.round(clamped))
    : String(Math.round(clamped * 10) / 10);
}

function formatCacheHitPercent(cacheReadTokens: number, inputTokens: number) {
  if (inputTokens <= 0) return "0";
  const clampedRead = Math.min(Math.max(0, cacheReadTokens), inputTokens);
  if (clampedRead === inputTokens) return "100";
  const percent = (clampedRead / inputTokens) * 100;
  const integerPercent = Math.round(percent);
  if (integerPercent < 100) return String(integerPercent);
  for (let precision = 1; precision <= 5; precision += 1) {
    const formatted = percent.toFixed(precision);
    if (Number(formatted) < 100) return formatted;
  }
  return "99.99999";
}
