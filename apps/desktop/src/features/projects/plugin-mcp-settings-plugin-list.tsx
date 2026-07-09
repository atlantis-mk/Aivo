import { Layers3, type LucideIcon, Plug, Server, Wrench } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/features/projects/plugin-mcp-settings-empty-state";
import { StatusBadge } from "@/features/projects/plugin-mcp-settings-mcp-actions";
import {
  normalizeSearch,
  pluginToolsForDisplay,
} from "@/features/projects/plugin-mcp-settings-model";
import {
  setPluginEnabled,
  type PluginListItem,
} from "@/services/aivo";

export function PluginItemGroup({
  emptyLabel,
  items,
  onReload,
}: {
  emptyLabel: string;
  items: PluginListItem[];
  onReload: () => Promise<void>;
}) {
  if (items.length === 0) {
    return <EmptyState label={emptyLabel} />;
  }
  return (
    <ItemGroup>
      {items.map((item) => (
        <PluginRow key={item.plugin.id} item={item} onReload={onReload} />
      ))}
    </ItemGroup>
  );
}

export function PluginRow({
  item,
  onReload,
}: {
  item: PluginListItem;
  onReload: () => Promise<void>;
}) {
  const plugin = item.plugin;
  const tools = pluginToolsForDisplay(item);
  const Icon = pluginIcon(plugin.manifest);
  return (
    <Item variant={plugin.error ? "outline" : "default"}>
      <ItemMedia variant="icon">
        <Icon />
      </ItemMedia>
      <ItemContent>
        <ItemTitle>
          {plugin.manifest.displayName || plugin.manifest.name}
          <StatusBadge status={plugin.status} />
        </ItemTitle>
        <ItemDescription>
          {plugin.manifest.description || plugin.rootPath}
        </ItemDescription>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{tools.length} tools</Badge>
          <Badge variant="outline">
            {plugin.manifest.hooks?.length ?? 0} hooks
          </Badge>
          {plugin.error ? (
            <Badge variant="destructive">{plugin.error}</Badge>
          ) : null}
        </div>
      </ItemContent>
      <ItemActions>
        <Switch
          checked={plugin.enabled}
          onCheckedChange={(enabled) =>
            void setPluginEnabled(plugin.id, enabled).then(onReload)
          }
        />
      </ItemActions>
    </Item>
  );
}

function pluginIcon(manifest: PluginListItem["plugin"]["manifest"]): LucideIcon {
  const text = normalizeSearch(
    [
      manifest.displayName,
      manifest.name,
      manifest.description,
      ...(manifest.keywords ?? []),
    ]
      .filter(Boolean)
      .join(" "),
  );
  if (/mcp|server/.test(text)) {
    return Server;
  }
  if (/skill|tool|template/.test(text)) {
    return Wrench;
  }
  if (/app|browser|chrome|figma|github|notion/.test(text)) {
    return Layers3;
  }
  return Plug;
}
