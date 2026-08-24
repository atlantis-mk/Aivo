export const MODEL_REASONING_EFFORT_OPTIONS = [
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
  { value: "ultra", label: "超高" },
];

export const MODEL_SERVICE_TIER_OPTIONS = [
  { value: "default", label: "标准" },
  { value: "priority", label: "快速" },
];

export function compactModelLabel(modelLabel: string) {
  return modelLabel.toLowerCase();
}
