import { CheckCircle2, Wrench } from "lucide-react";

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
import { EmptyState } from "@/features/projects/plugin-mcp-settings-empty-state";
import type { ToolCatalogEntry } from "@/services/aivo";

export function ToolItemGroup({
  emptyLabel,
  tools,
}: {
  emptyLabel: string;
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
            <Badge variant="outline">{tool.source}</Badge>
            {tool.enabled ? <CheckCircle2 /> : null}
          </ItemActions>
        </Item>
      ))}
    </ItemGroup>
  );
}
