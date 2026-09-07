import { useState } from "react";
import { Plug, Settings } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Switch } from "@/components/ui/switch";
import { McpServerSettingsDialog } from "@/features/projects/extension-settings-server-settings-dialog";
import {
  setMCPServerEnabled,
  type MCPServerListItem,
} from "@/services/aivo";

export function McpRow({
  item,
  onReload,
  sessionId,
}: {
  item: MCPServerListItem;
  onReload: () => Promise<void>;
  sessionId?: string;
}) {
  const server = item.server;
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
      <Item>
        <ItemMedia className="bg-muted text-muted-foreground" variant="image">
          <Plug />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>{server.displayName || server.name || server.id}</ItemTitle>
        </ItemContent>
        <ItemActions>
          <Button
            aria-label={`设置 ${server.displayName || server.name || server.id}`}
            onClick={() => setSettingsOpen(true)}
            size="icon"
            type="button"
            variant="ghost"
          >
            <Settings />
          </Button>
          <Switch
            checked={server.enabled}
            onCheckedChange={(enabled) =>
              void setMCPServerEnabled(server.id, enabled).then(onReload)
            }
          />
        </ItemActions>
      </Item>
      <McpServerSettingsDialog
        item={item}
        onClose={() => setSettingsOpen(false)}
        onReload={onReload}
        open={settingsOpen}
        sessionId={sessionId}
      />
    </Dialog>
  );
}
