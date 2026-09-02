import {
  Bot,
  Check,
  ChevronDown,
  Hand,
  ShieldAlert,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  agentModeShortLabel,
  fallbackAgentModes,
  normalizeAgentMode,
} from "@/features/projects/project-agent-mode-model";
import { normalizePermissionMode } from "@/features/projects/project-model-options";
import { cn } from "@/lib/utils";
import { appNameFromConfig } from "@/lib/app-identity";
import { useAppConfig } from "@/lib/app-config";
import type {
  AgentModeDefinition,
  AgentModeId,
  PermissionMode,
} from "@/services/aivo";

export function AgentModeMenu({
  compact,
  mode,
  modes,
  onModeSelect,
}: {
  compact: boolean;
  mode: AgentModeId;
  modes: AgentModeDefinition[];
  onModeSelect: (mode: AgentModeId) => void;
}) {
  const appName = appNameFromConfig(useAppConfig((state) => state.config));
  const options = modes.length > 0 ? modes : fallbackAgentModes;
  const visibleOptions = options.filter((option) => !option.hidden);
  const selectedMode = normalizeAgentMode(mode);
  const selectedOption =
    options.find((option) => option.id === selectedMode) ??
    visibleOptions[0] ??
    options[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          className="rounded-full font-semibold"
          size="sm"
          type="button"
          variant="ghost"
        >
          <Bot />
          <span className={cn(compact && "hidden")}>
            {agentModeShortLabel(selectedOption, appName)}
          </span>
          <ChevronDown data-icon="inline-end" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex w-72 max-w-[calc(100vw-2rem)] flex-col gap-1"
        side="top"
      >
        <DropdownMenuLabel>
          <span>Agent 模式</span>
        </DropdownMenuLabel>
        {visibleOptions.map((option) => {
          const selected = option.id === selectedMode;
          return (
            <DropdownMenuItem
              className={cn(selected && "bg-accent")}
              key={option.id}
              onSelect={() => onModeSelect(option.id)}
            >
              <Bot className="text-foreground" />
              <span className="min-w-0 flex-1">
                <span className="block font-semibold text-foreground">
                  {agentModeShortLabel(option, appName)}
                </span>
                <span className="block truncate text-muted-foreground">
                  {option.description}
                </span>
              </span>
              {selected ? <Check className="text-foreground" /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const permissionModeOptions: Array<{
  mode: PermissionMode;
  label: string;
  description: string;
}> = [
  {
    mode: "request_approval",
    label: "请求批准",
    description: "编辑外部文件和使用互联网时始终询问",
  },
  {
    mode: "full_access",
    label: "完全访问权限",
    description: "可不受限制地访问互联网和您电脑上的任何文件；新对话将沿用此选择",
  },
];

export function PermissionModeMenu({
  compact,
  mode,
  onModeSelect,
}: {
  compact: boolean;
  mode: PermissionMode;
  onModeSelect: (mode: PermissionMode) => void;
}) {
  const selectedMode = normalizePermissionMode(mode);
  const selectedOption =
    permissionModeOptions.find((option) => option.mode === selectedMode) ??
    permissionModeOptions[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          className="rounded-full font-semibold text-primary"
          size="sm"
          type="button"
          variant="ghost"
        >
          {permissionModeIcon(selectedMode)}
          <span className={cn(compact && "hidden")}>
            {selectedOption.label.replace("权限", "")}
          </span>
          <ChevronDown data-icon="inline-end" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-1"
        side="top"
      >
        <DropdownMenuLabel>
          <span>应如何批准 Codex 操作？</span>
        </DropdownMenuLabel>
        {permissionModeOptions.map((option) => {
          const selected = option.mode === selectedMode;
          return (
            <DropdownMenuItem
              className={cn(selected && "bg-accent")}
              key={option.mode}
              onSelect={() => onModeSelect(option.mode)}
            >
              <span className="text-foreground">
                {permissionModeIcon(option.mode)}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-semibold text-foreground">
                  {option.label}
                </span>
                <span className="block truncate text-muted-foreground">
                  {option.description}
                </span>
              </span>
              {selected ? <Check className="text-foreground" /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function permissionModeIcon(mode: PermissionMode, className?: string) {
  switch (normalizePermissionMode(mode)) {
    case "full_access":
      return <ShieldAlert className={className} data-icon="inline-start" />;
    default:
      return <Hand className={className} data-icon="inline-start" />;
  }
}
