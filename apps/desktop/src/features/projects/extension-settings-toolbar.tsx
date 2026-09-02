import { Plus, RefreshCw, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
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
          aria-label="刷新"
          disabled={loading}
          onClick={onReload}
          size="icon"
          title="刷新"
          type="button"
          variant="ghost"
        >
          <RefreshCw className={loading ? "animate-spin" : undefined} />
        </Button>
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
      </div>
    </div>
  );
}
