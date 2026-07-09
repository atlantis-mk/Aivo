import { MoreHorizontal, Plug, Plus, RefreshCw, Search } from "lucide-react";

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
import { addButtonLabel } from "@/features/projects/plugin-mcp-settings-model";
import type { PluginSettingsSection } from "@/features/projects/plugin-mcp-settings-model";

export function PluginMcpSettingsToolbar({
  applicationPluginCount,
  loading,
  onAdd,
  onQueryChange,
  onReload,
  onReloadPlugins,
  pluginCount,
  query,
  section,
  serverCount,
  skillCount,
  toolCount,
}: {
  applicationPluginCount: number;
  loading: boolean;
  onAdd: () => void;
  onQueryChange: (query: string) => void;
  onReload: () => void;
  onReloadPlugins: () => void;
  pluginCount: number;
  query: string;
  section: PluginSettingsSection;
  serverCount: number;
  skillCount: number;
  toolCount: number;
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-b p-4">
      <TabsList>
        <TabsTrigger value="plugins">
          插件 <span>{pluginCount}</span>
        </TabsTrigger>
        <TabsTrigger value="apps">
          应用 <span>{applicationPluginCount}</span>
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
      <div className="flex min-w-0 items-center gap-2">
        <InputGroup className="max-w-sm">
          <InputGroupAddon>
            <Search />
          </InputGroupAddon>
          <InputGroupInput
            aria-label="搜索插件"
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="搜索插件"
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
            <Button aria-label="插件操作" size="icon" variant="ghost">
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuGroup>
              <DropdownMenuItem disabled={loading} onSelect={onReload}>
                <RefreshCw />
                刷新
              </DropdownMenuItem>
              <DropdownMenuItem disabled={loading} onSelect={onReloadPlugins}>
                <Plug />
                重载插件
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
