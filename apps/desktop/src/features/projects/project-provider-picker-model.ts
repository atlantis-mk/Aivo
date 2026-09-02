import type { ProviderChoice } from "@/features/providers/provider-types";
import {
  providerBaseURLDefaults,
  primaryDefaultModelForProvider,
  providerDisplayName,
  providerProtocolForProvider,
} from "@/features/providers/provider-defaults";
import { normalizeModelId, type ModelOption } from "@/features/projects/project-model-options";
import type { CatalogState, ModelInfo, ProviderInfo } from "@/lib/provider-catalog";

export type ProviderPickOption = ProviderChoice & {
  type: string;
  baseUrl?: string;
  defaultModelId?: string;
  models: ModelInfo[];
};

const providerIconModules = import.meta.glob<string>(
  "@/assets/icons/provider/*.svg",
  {
    eager: true,
    import: "default",
    query: "?url",
  },
);

export function providerPickerOptions(catalogProviders: ProviderInfo[]) {
  const options = new Map<string, ProviderPickOption>();
  for (const [path, iconSrc] of Object.entries(providerIconModules)) {
    const id =
      path
        .split("/")
        .pop()
        ?.replace(/\.svg$/, "") ?? "";
    if (!id) continue;
    options.set(id, {
      id,
      name: providerDisplayName(id),
      type: providerProtocolForProvider(id),
      baseUrl: providerBaseURLDefaults[id],
      defaultModelId: defaultModelForProvider(id),
      iconSrc,
      models: [],
    });
  }
  for (const provider of catalogProviders) {
    const existing = options.get(provider.id);
    options.set(provider.id, {
      id: provider.id,
      name: provider.name || existing?.name || providerDisplayName(provider.id),
      type:
        provider.type ||
        existing?.type ||
        providerProtocolForProvider(provider.id),
      baseUrl:
        provider.baseUrl ||
        existing?.baseUrl ||
        providerBaseURLDefaults[provider.id],
      defaultModelId:
        provider.defaultModelId ||
        existing?.defaultModelId ||
        defaultModelForProvider(provider.id),
      iconSrc: existing?.iconSrc,
      models: provider.models ?? existing?.models ?? [],
    });
  }
  return Array.from(options.values()).sort((first, second) => {
    if (first.id === "custom-api") return -1;
    if (second.id === "custom-api") return 1;
    return first.name.localeCompare(second.name);
  });
}

export function catalogDefaultModelForProvider(
  catalogProviders: ProviderInfo[],
  providerId: string,
) {
  const provider = catalogProviders.find((item) => item.id === providerId);
  if (!provider) return "";
  if (provider.defaultModelId) return provider.defaultModelId;
  return (
    provider.models.find((model) => model.recommended)?.id ||
    provider.models[0]?.id ||
    ""
  );
}

export function defaultModelForProvider(providerId: string) {
  return primaryDefaultModelForProvider(providerId) || "custom-profile";
}

export function defaultModelFromCatalog(catalog: CatalogState, providerId: string) {
  const provider = catalog.providers.find((item) => item.id === providerId);
  if (!provider) return "";
  return (
    provider.defaultModelId ||
    provider.models.find((model) => model.recommended)?.id ||
    provider.models[0]?.id ||
    ""
  );
}

export function modelOptionFromCatalog(
  catalog: CatalogState,
  providerId: string,
  modelId: string,
): ModelOption | null {
  const provider = catalog.providers.find((item) => item.id === providerId);
  if (!provider) return null;
  const model =
    provider.models.find((item) => item.id === modelId) ?? provider.models[0];
  if (!model) return null;
  return {
    ...model,
    id: normalizeModelId(provider.id, model.id),
    providerId: provider.id,
    providerName: provider.name,
  };
}
