import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  CapabilityLine,
  StatusBadge,
} from "@/features/projects/extension-settings-components";
import type {
  MCPPromptRecord,
  MCPResourceRecord,
  MCPServerConfig,
  MCPToolRecord,
} from "@/services/aivo";

export function McpServerSummaryPanel({
  loadingLog,
  loadingOAuth,
  onConnectOAuth,
  onLoadLog,
  onLoadOAuthDiscovery,
  onProbe,
  onRefreshOAuthStatus,
  prompts,
  resources,
  server,
  templates,
  tools,
}: {
  loadingLog: boolean;
  loadingOAuth: boolean;
  onConnectOAuth: () => void;
  onLoadLog: () => void;
  onLoadOAuthDiscovery: () => void;
  onProbe: () => void;
  onRefreshOAuthStatus: () => void;
  prompts: MCPPromptRecord[];
  resources: MCPResourceRecord[];
  server: MCPServerConfig;
  templates: MCPResourceRecord[];
  tools: MCPToolRecord[];
}) {
  return (
    <div className="grid gap-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge
          status={server.status || (server.enabled ? "enabled" : "disabled")}
        />
        <Badge variant="secondary">{tools.length} tools</Badge>
        <Badge variant="outline">{prompts.length} prompts</Badge>
        <Badge variant="outline">{resources.length} resources</Badge>
        <Badge variant="outline">{templates.length} templates</Badge>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          disabled={loadingLog}
          onClick={onLoadLog}
          size="sm"
          variant="outline"
        >
          日志
        </Button>
        {server.authType === "oauth" && server.transport !== "stdio" ? (
          <>
            <Button
              disabled={loadingOAuth}
              onClick={onLoadOAuthDiscovery}
              size="sm"
              variant="outline"
            >
              OAuth 信息
            </Button>
            <Button
              disabled={loadingOAuth}
              onClick={onConnectOAuth}
              size="sm"
              variant="outline"
            >
              连接
            </Button>
            <Button
              disabled={loadingOAuth}
              onClick={onRefreshOAuthStatus}
              size="sm"
              variant="outline"
            >
              状态
            </Button>
          </>
        ) : null}
        <Button onClick={onProbe} size="sm" variant="outline">
          探测
        </Button>
      </div>
      {tools.length > 0 ? (
        <div className="grid gap-1 text-xs text-muted-foreground">
          {tools.slice(0, 3).map((tool) => (
            <CapabilityLine
              key={tool.id}
              label="tool"
              name={tool.name}
              detail={tool.description}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}
