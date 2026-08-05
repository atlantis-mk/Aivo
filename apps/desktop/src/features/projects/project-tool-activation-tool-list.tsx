import { Layers3, Plug, Server, Wrench } from "lucide-react";

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
            当前插件和运行环境没有提供可手动激活的工具。
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
              <ItemMedia variant={group.section === "mcp" ? "image" : "icon"}>
                <Icon />
              </ItemMedia>
              <ItemContent>
                <ItemTitle>{group.label}</ItemTitle>
                {group.description ? (
                  <ItemDescription>{group.description}</ItemDescription>
                ) : null}
                <div className="flex flex-wrap gap-2">
                  <Badge variant="secondary">{group.tools.length} 个工具</Badge>
                  {partiallyActive ? (
                    <Badge variant="outline">部分激活</Badge>
                  ) : null}
                  {used ? <Badge variant="outline">已使用</Badge> : null}
                </div>
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
    case "apps":
      return Layers3;
    case "mcp":
      return Server;
    case "plugins":
      return Plug;
    default:
      return Wrench;
  }
}
