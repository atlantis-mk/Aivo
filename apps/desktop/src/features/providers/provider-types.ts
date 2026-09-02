export type ProviderAuthMode = "oauth-browser" | "oauth-headless" | "api-key";

export type ProviderDialogStep = "options" | "details";

export type CustomProviderProtocol =
  | "openai"
  | "openai-compatible"
  | "anthropic"
  | "google"
  | "openrouter";

export type CustomProviderRow = {
  id: string;
  name: string;
  value: string;
};

export type CustomProviderForm = {
  providerId: string;
  displayName: string;
  protocol: CustomProviderProtocol;
  baseUrl: string;
  apiKey: string;
  models: CustomProviderRow[];
  headers: CustomProviderRow[];
};

export type ProviderChoice = {
  id: string;
  name: string;
  iconClassName?: string;
  iconSrc?: string;
  custom?: boolean;
  opensProviderPicker?: boolean;
};
