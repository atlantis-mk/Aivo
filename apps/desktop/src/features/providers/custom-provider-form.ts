import type {
  CustomProviderForm,
  CustomProviderRow,
  ProviderChoice,
} from "@/features/providers/provider-types";
import {
  defaultBaseURLForProvider,
  knownDefaultModelForProvider,
  providerProtocolForProvider,
} from "@/features/providers/provider-defaults";

type CustomProviderFormDefaults = {
  baseUrl?: string;
  fallbackModelId?: string;
};

export function emptyCustomProviderForm(): CustomProviderForm {
  return {
    providerId: "",
    displayName: "",
    protocol: "openai-compatible",
    baseUrl: "",
    apiKey: "",
    models: [emptyCustomRow()],
    headers: [emptyCustomRow()],
  };
}

export function customProviderFormFor(
  provider: ProviderChoice,
  defaults: CustomProviderFormDefaults = {},
): CustomProviderForm {
  if (!isCustomProviderChoice(provider)) return emptyCustomProviderForm();
  const defaultModel =
    knownDefaultModelForProvider(provider.id) || defaults.fallbackModelId || "";
  return {
    ...emptyCustomProviderForm(),
    providerId: provider.id === "custom-api" ? "" : provider.id,
    displayName: provider.id === "custom-api" ? "" : provider.name,
    protocol: providerProtocolForProvider(provider.id),
    baseUrl: defaults.baseUrl || defaultBaseURLForProvider(provider.id) || "",
    models: [
      defaultModel ? { ...emptyCustomRow(), name: defaultModel } : emptyCustomRow(),
    ],
  };
}

export function emptyCustomRow(): CustomProviderRow {
  return { id: crypto.randomUUID(), name: "", value: "" };
}

export function customProviderFormIsValid(form: CustomProviderForm) {
  return Boolean(
    form.providerId.trim() &&
      form.displayName.trim() &&
      form.baseUrl.trim() &&
      form.models.some((model) => model.name.trim()),
  );
}

export function headersFromRows(rows: CustomProviderRow[]) {
  return Object.fromEntries(
    rows
      .map((row) => [row.name.trim(), row.value.trim()] as const)
      .filter(([name, value]) => name && value),
  );
}

export function isCustomProviderChoice(provider: ProviderChoice | null) {
  return provider?.id === "custom-api";
}
