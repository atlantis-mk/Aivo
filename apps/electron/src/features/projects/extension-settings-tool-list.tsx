import { Wrench } from "lucide-react";

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
import { EmptyState } from "@/features/projects/extension-settings-empty-state";
import { Switch } from "@/components/ui/switch";
import {
  groupToolCatalogEntries,
  isToolCatalogGroupActive,
  isToolCatalogGroupPartiallyActive,
} from "@/features/projects/project-tool-activation-model";
import type { ToolCatalogEntry } from "@/services/aivo";

export function ToolItemGroup({
  activeToolSet,
  disabled,
  emptyLabel,
  onToggle,
  tools,
}: {
  activeToolSet: Set<string>;
  disabled: boolean;
  emptyLabel: string;
  onToggle: (toolNames: string[], enabled: boolean) => void;
  tools: ToolCatalogEntry[];
}) {
  if (tools.length === 0) {
    return <EmptyState label={emptyLabel} />;
  }
  const groups = groupToolCatalogEntries(tools, {});
  return (
    <ItemGroup>
      {groups.map((group) => (
        <Item
          key={group.id}
        >
          <ItemMedia variant="icon">
            <Wrench />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>{group.label}</ItemTitle>
            <ItemDescription>
              {group.description}
            </ItemDescription>
            {group.grouped ? (
              <div
                aria-label={`${group.label} 的工具`}
                className="mt-2 grid gap-1 rounded-md border bg-muted/30 px-3 py-2"
              >
                {group.tools.map((tool) => (
                  <div className="min-w-0" key={tool.name}>
                    <div className="truncate text-xs font-medium">{tool.name}</div>
                    {tool.description ? (
                      <div className="line-clamp-1 text-xs text-muted-foreground">
                        {tool.description}
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
            ) : null}
          </ItemContent>
          <ItemActions>
            {group.grouped ? (
              <Badge variant="secondary">{group.tools.length} 个工具</Badge>
            ) : (
              <Badge variant="outline">单工具</Badge>
            )}
            {isToolCatalogGroupPartiallyActive(group, activeToolSet) ? (
              <Badge variant="outline">部分启用</Badge>
            ) : null}
            <Switch
              aria-label={`启用 ${group.label}`}
              checked={isToolCatalogGroupActive(group, activeToolSet)}
              disabled={disabled}
              onCheckedChange={(enabled) =>
                onToggle(
                  group.tools.map((tool) => tool.name),
                  enabled,
                )
              }
              size="sm"
            />
          </ItemActions>
        </Item>
      ))}
    </ItemGroup>
  );
}
