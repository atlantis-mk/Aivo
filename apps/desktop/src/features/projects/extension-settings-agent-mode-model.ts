import type {
  CatalogState,
  ModelInfo,
  ProviderInfo,
} from "@/lib/provider-catalog";
import type { AgentModeDefinition } from "@/services/aivo";

export function agentModeSubagentCandidates(
  modes: AgentModeDefinition[],
  ownerId: string,
): AgentModeDefinition[] {
  const normalizedOwner = ownerId.trim().toLowerCase();
  return modes.filter(
    (mode) =>
      !mode.hidden &&
      mode.id !== normalizedOwner &&
      mode.mode !== "primary",
  );
}

export function connectedAgentModeProviders(
  catalog: CatalogState | null,
): ProviderInfo[] {
  if (!catalog) return [];

  const explicitlyConnectedIds = new Set([
    ...(catalog.connected ?? []),
    ...(catalog.connectedProviders ?? []).map((provider) => provider.id),
  ]);
  const providersById = new Map<string, ProviderInfo>();

  for (const provider of [
    ...catalog.providers,
    ...(catalog.connectedProviders ?? []),
  ]) {
    if (!provider.id || providersById.has(provider.id)) continue;
    if (
      explicitlyConnectedIds.has(provider.id) ||
      provider.connected ||
      provider.auth?.connected ||
      provider.readiness?.ready ||
      Boolean(provider.accounts?.length)
    ) {
      providersById.set(provider.id, provider);
    }
  }

  return Array.from(providersById.values());
}

export function agentModeModelsForProvider(
  catalog: CatalogState | null,
  providerId: string,
): ModelInfo[] {
  const provider = connectedAgentModeProviders(catalog).find(
    (item) => item.id === providerId,
  );
  if (!provider) return [];

  const candidates = provider.models?.length
    ? provider.models
    : (catalog?.models ?? []).filter((model) => model.providerId === providerId);
  const models = new Map<string, ModelInfo>();
  for (const model of candidates) {
    if (!model.id || models.has(model.id)) continue;
    models.set(model.id, model);
  }
  return Array.from(models.values());
}
