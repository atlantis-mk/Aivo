import { useMcpServerCapabilityState } from "@/features/projects/extension-settings-server-capability-state";
import { useMcpServerDraftState } from "@/features/projects/extension-settings-server-draft-state";
import type { MCPServerListItem } from "@/services/aivo";

export function useMcpServerSettingsState({
  item,
  onClose,
  onReload,
  open,
  sessionId,
}: {
  item: MCPServerListItem;
  onClose: () => void;
  onReload: () => Promise<void>;
  open: boolean;
  sessionId?: string;
}) {
  const server = item.server;
  const tools = item.tools ?? [];
  const prompts = item.prompts ?? [];
  const resources = item.resources ?? [];
  const templates = item.resourceTemplates ?? [];
  const draftState = useMcpServerDraftState({
    onClose,
    onReload,
    open,
    server,
  });
  const capabilityState = useMcpServerCapabilityState({
    onReload,
    serverId: server.id,
    sessionId,
  });

  return {
    prompts,
    resources,
    server,
    templates,
    tools,
    ...draftState,
    ...capabilityState,
  };
}
