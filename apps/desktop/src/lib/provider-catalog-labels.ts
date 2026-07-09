import type { ProviderInfo } from "@/lib/provider-catalog-types";

export function providerTypeLabel(type?: string) {
  switch (type) {
    case "openai-compatible":
      return "OpenAI Compatible";
    case "openai":
      return "OpenAI";
    case "anthropic":
      return "Anthropic";
    case "google":
      return "Google Gemini";
    case "openrouter":
      return "OpenRouter";
    case "claude-code":
      return "Claude Code";
    case "gemini":
      return "Gemini";
    default:
      return type || "Unspecified";
  }
}

export function providerConnectionLabel(provider: ProviderInfo) {
  switch (provider.connectionSource) {
    case "api-key":
      return "API key";
    case "env":
      return provider.environment ? `Env: ${provider.environment}` : "Environment";
    case "oauth-browser":
      return "Browser auth";
    case "oauth":
      return "OAuth";
    case "oauth-headless":
      return "Headless auth";
    default:
      return provider.connected ? "Connected" : "Not connected";
  }
}
