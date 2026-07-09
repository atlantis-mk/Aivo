import { Wrench } from "lucide-react";

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
  toolActivationSwitchId,
  type ToolCatalogGroup,
} from "./project-tool-activation-model";

export function ToolActivationToolList({
  activeToolSet,
  disabled,
  groupedTools,
  onToggleTool,
  usedToolSet,
}: {
  activeToolSet: Set<string>;
  disabled: boolean;
  groupedTools: ToolCatalogGroup[];
  onToggleTool: (name: string, enabled: boolean) => void;
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
    <div className="flex min-w-0 flex-col gap-4">
      {groupedTools.map((group) => (
        <section className="flex min-w-0 flex-col gap-2" key={group.id}>
          <div className="flex items-center gap-2">
            <div className="min-w-0 flex-1 truncate text-sm font-medium">
              {group.label}
            </div>
            <Badge variant="outline">{group.tools.length}</Badge>
          </div>

          <div className="overflow-hidden rounded-md border">
            {group.tools.map((tool) => {
              const active = activeToolSet.has(tool.name);
              const switchId = toolActivationSwitchId(tool.name);
              return (
                <label
                  className="grid w-full min-w-0 cursor-pointer grid-cols-[minmax(0,1fr)_auto] items-center gap-3 overflow-hidden border-b p-3 transition-colors last:border-b-0 hover:bg-muted/50"
                  htmlFor={switchId}
                  key={tool.name}
                >
                  <div className="min-w-0 flex-1 overflow-hidden">
                    <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-2">
                      <div className="truncate text-sm font-medium">
                        {tool.name}
                      </div>
                      {usedToolSet.has(tool.name) ? (
                        <Badge variant="secondary">已使用</Badge>
                      ) : null}
                    </div>
                    {tool.description ? (
                      <div
                        className="mt-1 truncate text-xs text-muted-foreground"
                        title={tool.description}
                      >
                        {tool.description}
                      </div>
                    ) : null}
                  </div>
                  <Switch
                    checked={active}
                    className="shrink-0"
                    disabled={disabled}
                    id={switchId}
                    onCheckedChange={(checked) =>
                      onToggleTool(tool.name, checked)
                    }
                    size="sm"
                  />
                </label>
              );
            })}
          </div>
        </section>
      ))}
    </div>
  );
}
