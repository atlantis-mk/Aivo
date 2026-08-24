import { Bot, MoreHorizontal, Plus, RefreshCw, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { TabsList, TabsTrigger } from "@/components/ui/tabs";
import { addButtonLabel } from "@/features/projects/extension-settings-model";
import type { ExtensionSettingsSection } from "@/features/projects/extension-settings-model";

export function ExtensionSettingsToolbar({
  extensionCount,
  loading,
  onAdd,
  onManageAgentModes,
  onQueryChange,
  onReload,
  query,
  section,
  serverCount,
  skillCount,
  toolCount,
}: {
  extensionCount: number;
  loading: boolean;
  onAdd: () => void;
  onManageAgentModes: () => void;
  onQueryChange: (query: string) => void;
  onReload: () => void;
  query: string;
  section: ExtensionSettingsSection;
  serverCount: number;
  skillCount: number;
  toolCount: number;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-b p-4">
      <div className="min-w-0 overflow-x-auto overflow-y-hidden">
      <TabsList>
        <TabsTrigger value="extensions">
          扩展 <span>{extensionCount}</span>
        </TabsTrigger>
        <TabsTrigger value="mcp">
          MCP <span>{serverCount}</span>
        </TabsTrigger>
        <TabsTrigger value="skills">
          技能 <span>{skillCount}</span>
        </TabsTrigger>
        <TabsTrigger value="tools">
          工具 <span>{toolCount}</span>
        </TabsTrigger>
      </TabsList>
      </div>
      <div className="flex min-w-0 items-center gap-2">
        <InputGroup className="max-w-sm">
          <InputGroupAddon>
            <Search />
          </InputGroupAddon>
          <InputGroupInput
            aria-label="搜索扩展、MCP、技能和工具"
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="搜索扩展、MCP、技能和工具"
            value={query}
          />
        </InputGroup>
        <Button
          aria-label={addButtonLabel(section)}
          onClick={onAdd}
          size="icon"
          title={addButtonLabel(section)}
          type="button"
          variant="ghost"
        >
          <Plus />
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button aria-label="扩展操作" size="icon" variant="ghost">
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuGroup>
              <DropdownMenuItem onSelect={onManageAgentModes}>
                <Bot />
                管理 Agent 模式
              </DropdownMenuItem>
              <DropdownMenuItem disabled={loading} onSelect={onReload}>
                <RefreshCw />
                刷新
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
