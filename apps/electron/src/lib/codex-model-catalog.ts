import type { CatalogState, ModelInfo } from "@/lib/provider-catalog";

type CodexModel = {
  id: string;
  name: string;
};

export function catalogWithCodexModels(
  catalog: CatalogState,
  codexModels: CodexModel[],
): CatalogState {
  const models: ModelInfo[] = codexModels.map((model) => ({
    id: model.id,
    providerId: "openai",
    name: model.name,
  }));
  const defaultModelId = models[0]?.id;
  const updateProvider = (provider: (typeof catalog.providers)[number]) =>
    provider.id === "openai"
      ? { ...provider, defaultModelId, models }
      : provider;
  const providers = catalog.providers.map(updateProvider);

  return {
    ...catalog,
    providers,
    models: [
      ...catalog.models.filter((model) => model.providerId !== "openai"),
      ...models,
    ],
    defaultModel: defaultModelId
      ? { providerId: "openai", modelId: defaultModelId }
      : catalog.defaultModel,
    connectedProviders: catalog.connectedProviders?.map(updateProvider),
    popularProviders: catalog.popularProviders?.map(updateProvider),
    customProviders: catalog.customProviders?.map(updateProvider),
  };
}
