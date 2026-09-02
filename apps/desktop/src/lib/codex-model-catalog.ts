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
  const providers = catalog.providers.map((provider) =>
    provider.id === "openai"
      ? { ...provider, defaultModelId, models }
      : provider,
  );

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
  };
}
