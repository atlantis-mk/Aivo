import { Plug, Wrench } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Switch } from "@/components/ui/switch";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import {
  isToolCatalogGroupActive,
  isToolCatalogGroupPartiallyActive,
  isToolCatalogGroupUsed,
  toolActivationSwitchId,
  type ToolCatalogGroup,
} from "./project-tool-activation-model";

export function ToolActivationToolList({
  activeToolSet,
  disabled,
  groupedTools,
  onToggleToolGroup,
  usedToolSet,
}: {
  activeToolSet: Set<string>;
  disabled: boolean;
  groupedTools: ToolCatalogGroup[];
  onToggleToolGroup: (names: string[], enabled: boolean) => void;
  usedToolSet: Set<string>;
}) {
  if (groupedTools.length === 0) {
    return (
      <Empty className="min-h-48 border">
        <EmptyMedia variant="icon">
          <Wrench />
        </EmptyMedia>
        <EmptyHeader>
          <EmptyTitle>没有可激活工具</EmptyTitle>
          <EmptyDescription>
            当前扩展和运行环境没有提供可手动激活的工具。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <ItemGroup className="gap-3">
      {groupedTools.map((group) => {
        const active = isToolCatalogGroupActive(group, activeToolSet);
        const partiallyActive = isToolCatalogGroupPartiallyActive(
          group,
          activeToolSet,
        );
        const used = isToolCatalogGroupUsed(group, usedToolSet);
        const switchId = toolActivationSwitchId(group.id);
        const Icon = toolGroupIcon(group);
        return (
          <Item asChild key={group.id} variant="outline">
            <label className="cursor-pointer" htmlFor={switchId}>
              <ItemMedia variant="icon">
                <Icon />
              </ItemMedia>
              <ItemContent>
                <ItemTitle>{group.label}</ItemTitle>
                {group.description ? (
                  <ItemDescription>{group.description}</ItemDescription>
                ) : null}
                <div className="flex flex-wrap gap-2">
                  {group.grouped ? (
                    <Badge variant="secondary">{group.tools.length} 个工具</Badge>
                  ) : null}
                  {partiallyActive ? (
                    <Badge variant="outline">部分激活</Badge>
                  ) : null}
                  {used ? <Badge variant="outline">已使用</Badge> : null}
                </div>
                {group.grouped ? (
                  <div
                    aria-label={`${group.label} 的工具`}
                    className="mt-2 grid gap-1 rounded-md border bg-muted/30 px-3 py-2"
                  >
                    {group.tools.map((tool) => (
                      <div className="min-w-0" key={tool.name}>
                        <div className="truncate text-xs font-medium">
                          {tool.name}
                        </div>
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
                <Switch
                  aria-label={`激活 ${group.label}`}
                  checked={active}
                  disabled={disabled}
                  id={switchId}
                  onCheckedChange={(checked) =>
                    onToggleToolGroup(
                      group.tools.map((tool) => tool.name),
                      checked,
                    )
                  }
                  size="sm"
                />
              </ItemActions>
            </label>
          </Item>
        );
      })}
    </ItemGroup>
  );
}

function toolGroupIcon(group: ToolCatalogGroup) {
  switch (group.section) {
    case "extensions":
    case "mcp":
      return Plug;
    default:
      return Wrench;
  }
}
