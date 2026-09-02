import { fallbackPopularProviders, fallbackProviders } from "@/lib/provider-catalog-fallback-providers";
import type { CatalogState } from "@/lib/provider-catalog-types";

export function fallbackCatalogState(): CatalogState {
  return {
    providers: fallbackProviders(),
    models: [],
    connected: [],
    connectedProviders: [],
    popularProviders: fallbackPopularProviders(),
    customProviders: [],
  };
}
