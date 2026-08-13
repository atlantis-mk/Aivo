export type ProviderSettingsItem = {
  id: string;
  name: string;
  connected?: boolean;
  defaultModelId?: string;
  models?: Array<{ id: string; name: string }>;
  accounts?: Array<{ id: string; method?: string }>;
  auth?: { connected?: boolean; type?: string };
  readiness?: { ready?: boolean; reason?: string };
};

export function configuredProviders<T extends ProviderSettingsItem>(
  providers: T[],
  defaultProviderId?: string,
) {
  return providers
    .filter(
      (provider) =>
        provider.connected ||
        provider.auth?.connected ||
        Boolean(provider.accounts?.length),
    )
    .sort((first, second) => {
      if (first.id === defaultProviderId) return -1;
      if (second.id === defaultProviderId) return 1;
      return first.name.localeCompare(second.name);
    });
}

export function providerModelLabel(provider: ProviderSettingsItem) {
  if (!provider.defaultModelId) return "未选择默认模型";
  const model = provider.models?.find(
    (candidate) => candidate.id === provider.defaultModelId,
  );
  return model?.name || provider.defaultModelId;
}

export function providerConnectionMethodLabel(provider: ProviderSettingsItem) {
  const methods = Array.from(
    new Set(
      (provider.accounts ?? [])
        .map((account) => account.method?.trim())
        .filter((method): method is string => Boolean(method)),
    ),
  );
  if (methods.length === 0 && provider.auth?.type) {
    methods.push(provider.auth.type);
  }
  if (methods.length === 0) return "已配置";
  return methods.map(connectionMethodLabel).join("、");
}

export function providerReadinessLabel(provider: ProviderSettingsItem) {
  if (provider.readiness?.ready) return "可用";
  if (provider.readiness?.reason?.trim()) return provider.readiness.reason.trim();
  return provider.connected || provider.auth?.connected ? "已连接" : "待连接";
}

function connectionMethodLabel(method: string) {
  if (method === "api-key") return "API Key";
  if (method === "oauth-browser") return "浏览器 OAuth";
  if (method === "oauth-headless") return "设备授权";
  if (method === "env") return "环境变量";
  return method;
}
