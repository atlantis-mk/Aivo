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
  onToggle: (toolName: string, enabled: boolean) => void;
  tools: ToolCatalogEntry[];
}) {
  if (tools.length === 0) {
    return <EmptyState label={emptyLabel} />;
  }
  return (
    <ItemGroup>
      {tools.map((tool) => (
        <Item
          key={`${tool.source}:${tool.sourceId ?? ""}:${tool.registrationId ?? ""}:${tool.name}`}
        >
          <ItemMedia variant="icon">
            <Wrench />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>{tool.name}</ItemTitle>
            <ItemDescription>
              {tool.description || tool.capability || tool.namespace}
            </ItemDescription>
          </ItemContent>
          <ItemActions>
            <Badge variant="outline">内置</Badge>
            <Switch
              aria-label={`启用 ${tool.name}`}
              checked={activeToolSet.has(tool.name)}
              disabled={disabled}
              onCheckedChange={(enabled) => onToggle(tool.name, enabled)}
              size="sm"
            />
          </ItemActions>
        </Item>
      ))}
    </ItemGroup>
  );
}
