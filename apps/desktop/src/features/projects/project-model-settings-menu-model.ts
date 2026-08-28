import type { ModelInfo } from "@/lib/provider-catalog";

const MODEL_REASONING_EFFORT_OPTIONS = [
  { value: "none", label: "无" },
  { value: "minimal", label: "最小" },
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
  { value: "xhigh", label: "特高" },
  { value: "max", label: "最大" },
  { value: "ultra", label: "超高" },
];

const LEGACY_REASONING_EFFORTS = new Set(["low", "medium", "high", "ultra"]);

const MODEL_SERVICE_TIER_OPTIONS = [
  { value: "default", label: "标准" },
  { value: "priority", label: "快速" },
  { value: "flex", label: "弹性" },
];

export function reasoningEffortOptionsForModel(model?: ModelInfo) {
  const declared = new Set(model?.supportedReasoningEfforts ?? []);
  const supported = declared.size > 0 ? declared : LEGACY_REASONING_EFFORTS;
  return MODEL_REASONING_EFFORT_OPTIONS.filter((option) =>
    supported.has(option.value),
  );
}

export function serviceTierOptionsForModel(model?: ModelInfo) {
  const declaredValues = [
    ...(model?.serviceTiers ?? []),
    model?.defaultServiceTier ?? "",
  ]
    .map((tier) => (tier === "fast" ? "priority" : tier))
    .filter(Boolean);
  if (declaredValues.length === 0) {
    return MODEL_SERVICE_TIER_OPTIONS.filter((option) =>
      option.value === "default" || option.value === "priority",
    );
  }
  const declared = new Set(
    declaredValues,
  );
  declared.add("default");
  return MODEL_SERVICE_TIER_OPTIONS.filter((option) =>
    declared.has(option.value),
  );
}

export function compactModelLabel(modelLabel: string) {
  return modelLabel.toLowerCase();
}
