export function modelSelectionAfterCatalogRefresh(
  selectedModelId: string | undefined,
  refreshedDefaultModelId: string,
) {
  return selectedModelId?.trim() || refreshedDefaultModelId.trim();
}
