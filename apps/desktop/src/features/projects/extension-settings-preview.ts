import type {
  MCPResourceReadResult,
  MCPOAuthDiscoveryResult,
  MCPOAuthStartResult,
  MCPOAuthStatus,
} from "@/services/aivo";

export function templateVariables(template: string) {
  const variables: string[] = [];
  for (const match of template.matchAll(/\{([A-Za-z0-9_.-]+)\}/g)) {
    const name = match[1];
    if (!variables.includes(name)) {
      variables.push(name);
    }
  }
  return variables;
}

export function applySimpleTemplate(
  template: string,
  values: Record<string, string>,
) {
  return template.replace(/\{([A-Za-z0-9_.-]+)\}/g, (_match, name: string) =>
    encodeURIComponent(values[name] ?? ""),
  );
}

export function resourcePreview(result: MCPResourceReadResult) {
  if (result.content?.trim()) {
    return result.content;
  }
  const blobCount =
    result.contents?.filter((content) => content.blob).length ?? 0;
  if (blobCount > 0) {
    return `${blobCount} binary content block${blobCount === 1 ? "" : "s"}`;
  }
  return stringifyStructured(result.structured);
}

export function oauthDiscoveryPreview(result: MCPOAuthDiscoveryResult) {
  const lines = [
    ["Resource metadata", result.resourceMetadataUrl],
    ["Resource", result.resource],
    ["Issuer", result.selectedIssuer],
    ["Authorize", result.authorizationEndpoint],
    ["Token", result.tokenEndpoint],
    ["Register", result.registrationEndpoint],
    ["Scopes", result.scopesSupported?.join(" ")],
    ["PKCE", result.codeChallengeMethods?.join(", ")],
    [
      "Dynamic registration",
      result.requiresDynamicClientRegistration ? "available" : "",
    ],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`);
  if (result.discoveryErrors?.length) {
    lines.push("", "Discovery warnings:", ...result.discoveryErrors);
  }
  return lines.join("\n") || stringifyStructured(result.authorizationMetadata);
}

export function oauthStartPreview(result: MCPOAuthStartResult) {
  return [
    ["Status", result.status],
    ["URL", result.url],
    ["Expires", result.expiresAt],
    ["Instructions", result.instructions],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`)
    .join("\n");
}

export function oauthStatusPreview(result: MCPOAuthStatus) {
  return [
    ["Status", result.status],
    ["Connected", result.connected ? "yes" : "no"],
    ["Client ID", result.clientId],
    ["Token source", result.tokenSource],
    ["Expires", result.expiresAt],
    ["Error", result.error],
  ]
    .filter(([, value]) => Boolean(value))
    .map(([label, value]) => `${label}: ${value}`)
    .join("\n");
}

export function stringifyStructured(
  value: Record<string, unknown> | undefined,
) {
  if (!value) {
    return "";
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
