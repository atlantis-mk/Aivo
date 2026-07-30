import { useState, type ComponentType, type KeyboardEvent, type MouseEvent } from "react";
import {
  ChevronDown,
  FileDiff,
  GitBranch,
  GitCommitHorizontal,
  Globe,
  Laptop,
  Plus,
  Terminal,
  Wrench,
} from "lucide-react";

import { topBarMenuItemClassName } from "@/components/app-top-bar-menu-model";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function EnvironmentSummaryPanel({
  className,
  onAddContext,
  onCommitOrPush,
  onOpenTools,
  onSelectBranch,
  onSelectLocalEnvironment,
}: {
  className?: string;
  onAddContext?: () => void;
  onCommitOrPush?: () => void;
  onOpenTools?: () => void;
  onSelectBranch?: () => void;
  onSelectLocalEnvironment?: () => void;
}) {
  const [expanded, setExpanded] = useState(true);
  const [sourceExpanded, setSourceExpanded] = useState(true);

  function toggleExpanded() {
    setExpanded((current) => !current);
  }

  function toggleSourceExpanded() {
    setSourceExpanded((current) => !current);
  }

  function handleHeaderKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    toggleExpanded();
  }

  function handleSourceHeaderKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    toggleSourceExpanded();
  }

  function handleAddContextClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    onAddContext?.();
  }

  function handleToggleButtonClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    toggleExpanded();
  }

  function handleSourceToggleButtonClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation();
    toggleSourceExpanded();
  }

  return (
    <div
      className={cn(
        "w-72 overflow-hidden rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10",
        className,
      )}
    >
      <div
        aria-expanded={expanded}
        className={cn(
          "group/env-header flex cursor-pointer items-center rounded-md px-2 py-1.5 outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/30",
        )}
        onClick={toggleExpanded}
        onKeyDown={handleHeaderKeyDown}
        role="button"
        tabIndex={0}
      >
        <div className="flex min-w-0 flex-1 items-center">
          <span className="min-w-0 truncate text-xs text-muted-foreground">
            环境信息
          </span>
          <Button
            aria-label={expanded ? "收起环境信息" : "展开环境信息"}
            className="ml-1 opacity-0 transition-opacity group-hover/env-header:opacity-100 group-focus-within/env-header:opacity-100"
            onClick={handleToggleButtonClick}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ChevronDown
              className={cn(
                "transition-transform duration-150",
                !expanded && "-rotate-90",
              )}
            />
          </Button>
        </div>
        <Button
          aria-label="添加上下文"
          onClick={handleAddContextClick}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <Plus />
        </Button>
      </div>
      <div
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-200 ease-out",
          expanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="flex flex-col">
            <EnvironmentSummaryRow icon={FileDiff} label="变更" />
            <EnvironmentSummaryRow
              action="expand"
              icon={Laptop}
              label="本地"
              onClick={onSelectLocalEnvironment}
            />
            <EnvironmentSummaryRow
              action="expand"
              icon={GitBranch}
              label="main"
              onClick={onSelectBranch}
            />
            <EnvironmentSummaryRow
              icon={GitCommitHorizontal}
              label="提交或推送"
              onClick={onCommitOrPush}
            />
            <EnvironmentSummaryRow icon={Wrench} label="工具" onClick={onOpenTools} />
            <EnvironmentSummaryRow
              disabled
              icon={Terminal}
              label="GitHub CLI 未通过身份验证"
            />
          </div>
        </div>
      </div>
      <div className="-mx-1 my-1 h-px bg-border/50" />
      <div
        aria-expanded={sourceExpanded}
        className="group/source-header flex cursor-pointer items-center rounded-md px-2 py-1.5 outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/30"
        onClick={toggleSourceExpanded}
        onKeyDown={handleSourceHeaderKeyDown}
        role="button"
        tabIndex={0}
      >
        <span className="min-w-0 truncate text-xs text-muted-foreground">
          来源
        </span>
        <Button
          aria-label={sourceExpanded ? "收起来源" : "展开来源"}
          className="ml-1 opacity-0 transition-opacity group-hover/source-header:opacity-100 group-focus-within/source-header:opacity-100"
          onClick={handleSourceToggleButtonClick}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ChevronDown
            className={cn(
              "transition-transform duration-150",
              !sourceExpanded && "-rotate-90",
            )}
          />
        </Button>
      </div>
      <div
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-200 ease-out",
          sourceExpanded ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <EnvironmentSummaryRow icon={Globe} label="" />
        </div>
      </div>
    </div>
  );
}

function EnvironmentSummaryRow({
  action,
  disabled,
  icon: Icon,
  label,
  onClick,
}: {
  action?: "expand";
  disabled?: boolean;
  icon: ComponentType<{ className?: string }>;
  label: string;
  onClick?: () => void;
}) {
  const Comp = onClick ? "button" : "div";

  return (
    <Comp
      aria-label={onClick ? label : undefined}
      className={cn(
        topBarMenuItemClassName,
        disabled ? "text-muted-foreground opacity-70" : "text-card-foreground",
      )}
      onClick={onClick}
      type={onClick ? "button" : undefined}
    >
      <Icon className="text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate text-left">{label}</span>
      {action === "expand" && <ChevronDown className="text-muted-foreground" />}
    </Comp>
  );
}
