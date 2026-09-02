import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export function getAppConfig() {
  return invoke<domain.AppConfig>("GetAppConfig");
}

export function getProviderCatalog() {
  return invoke<domain.CatalogState>("GetProviderCatalog");
}

export function getProviderCatalogForProject(projectPath: string) {
  return invoke<domain.CatalogState>("GetProviderCatalogForProject", projectPath);
}

export type ProviderEcosystemRefreshResult = {
  source: string;
  cachePath: string;
  refreshedAt: string;
  providerCount: number;
  modelCount: number;
  unsupportedCount?: number;
};

export function refreshProviderEcosystemCatalog(url?: string) {
  return invoke<ProviderEcosystemRefreshResult>(
    "RefreshProviderEcosystemCatalog",
    { url },
  );
}

export function connectProvider(input: domain.ProviderConnectInput) {
  return invoke<domain.CatalogState>("ConnectProvider", input);
}

export function deleteProvider(providerId: string) {
  return invoke<domain.CatalogState>("DeleteProvider", providerId);
}

export type CompleteInitializationInput = {
  appName: string;
  initialWorkspacePath: string;
  provider?: domain.ProviderConfig;
};

export function completeInitialization(input: CompleteInitializationInput) {
  return invoke<domain.AppConfig>("CompleteInitialization", input);
}

export function updateModelPreferences(input: domain.ModelPreferencesInput) {
  return invoke<domain.AppConfig>("UpdateModelPreferences", input);
}

export function refreshProviderModels(input: domain.ProviderConnectInput) {
  return invoke<domain.CatalogState>("RefreshProviderModels", input);
}

export function deleteProviderAccount(accountId: string) {
  return invoke<domain.CatalogState>("DeleteProviderAccount", accountId);
}

export function startProviderAuth(input: domain.ProviderAuthStartInput) {
  return invoke<domain.ProviderAuthStartResult>("StartProviderAuth", input);
}

export function getProviderAuthStatus(providerId: string) {
  return invoke<domain.ProviderAuthStatus>("GetProviderAuthStatus", providerId);
}

export function cancelProviderAuth(providerId: string) {
  return invoke<domain.ProviderAuthStatus>("CancelProviderAuth", providerId);
}
