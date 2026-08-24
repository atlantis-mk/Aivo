import type { domain } from "../../../bridge/go/models";
import type { ProviderConnectInput } from "@/lib/provider-catalog";

export type AuxiliaryModelPreferenceInput = domain.ModelPreferencesInput & {
  auxiliaryModel: domain.ModelRef;
};

export function providerConnectInputForBridge(
  input: ProviderConnectInput,
): domain.ProviderConnectInput {
  return input;
}

export function modelRefForProvider(
  providerId: string,
  modelId: string,
): domain.ModelRef {
  return { providerId, modelId };
}

export function auxiliaryModelPreference(
  auxiliaryModel: domain.ModelRef,
): AuxiliaryModelPreferenceInput {
  return { auxiliaryModel } as AuxiliaryModelPreferenceInput;
}
