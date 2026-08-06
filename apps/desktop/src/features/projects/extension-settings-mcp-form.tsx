import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  McpField,
  McpKeyValueRows,
  McpStringRows,
} from "@/features/projects/extension-settings-mcp-fields";
import {
  mcpTransportLabel,
  parseWords,
  type KeyValueRow,
} from "@/features/projects/extension-settings-model";
import { type MCPServerConfig } from "@/services/aivo";

export function McpServerDraftForm({
  argRows,
  draft,
  envRows,
  headerRows,
  onArgRowsChange,
  onDraftChange,
  onEnvRowsChange,
  onHeaderRowsChange,
  onRootRowsChange,
  rootRows,
  showEnabledToggle = false,
  transportEditable = false,
}: {
  argRows: string[];
  draft: MCPServerConfig;
  envRows: KeyValueRow[];
  headerRows: KeyValueRow[];
  onArgRowsChange: (rows: string[]) => void;
  onDraftChange: (draft: MCPServerConfig) => void;
  onEnvRowsChange: (rows: KeyValueRow[]) => void;
  onHeaderRowsChange: (rows: KeyValueRow[]) => void;
  onRootRowsChange: (rows: string[]) => void;
  rootRows: string[];
  showEnabledToggle?: boolean;
  transportEditable?: boolean;
}) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-3 rounded-md border p-3">
        <McpField label="服务器 ID">
          <Input
            placeholder={transportEditable ? "filesystem" : undefined}
            value={draft.id}
            onChange={(event) => onDraftChange({ ...draft, id: event.target.value })}
          />
        </McpField>
        <McpField label="显示名称">
          <Input
            placeholder={transportEditable ? "Filesystem" : undefined}
            value={draft.displayName ?? ""}
            onChange={(event) => onDraftChange({ ...draft, displayName: event.target.value })}
          />
        </McpField>
        <McpField label="服务器类型">
          {transportEditable ? (
            <Select
              value={draft.transport}
              onValueChange={(transport) =>
                onDraftChange({
                  ...draft,
                  transport: transport as MCPServerConfig["transport"],
                })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="服务器类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="stdio">stdio</SelectItem>
                  <SelectItem value="streamable_http">Streamable HTTP</SelectItem>
                  <SelectItem value="sse">SSE</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          ) : (
            <Input value={mcpTransportLabel(draft.transport)} disabled />
          )}
        </McpField>
      </div>

      {draft.transport === "stdio" ? (
        <div className="grid gap-3 rounded-md border p-3">
          <McpField label="启动命令">
            <Input
              placeholder={transportEditable ? "npx" : undefined}
              value={draft.command ?? ""}
              onChange={(event) => onDraftChange({ ...draft, command: event.target.value })}
            />
          </McpField>
          <McpStringRows
            addLabel="添加参数"
            label="参数"
            onChange={onArgRowsChange}
            placeholder="参数"
            rows={argRows}
          />
        </div>
      ) : (
        <div className="grid gap-3 rounded-md border p-3">
          <McpField label="Server URL">
            <Input
              placeholder={transportEditable ? "https://example.com/mcp" : undefined}
              value={draft.url ?? ""}
              onChange={(event) => onDraftChange({ ...draft, url: event.target.value })}
            />
          </McpField>
          <McpField label="认证方式">
            <Select
              value={draft.authType ?? "none"}
              onValueChange={(authType) =>
                onDraftChange({
                  ...draft,
                  authType: authType as MCPServerConfig["authType"],
                })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="认证方式" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">No auth</SelectItem>
                  <SelectItem value="bearer">Bearer env</SelectItem>
                  <SelectItem value="oauth">OAuth bearer env</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </McpField>
          {draft.authType === "bearer" || draft.authType === "oauth" ? (
            <McpField label="Token 环境变量">
              <Input
                placeholder={transportEditable ? "MCP_TOKEN" : undefined}
                value={draft.bearerTokenEnv ?? ""}
                onChange={(event) =>
                  onDraftChange({ ...draft, bearerTokenEnv: event.target.value })
                }
              />
            </McpField>
          ) : null}
          {draft.authType === "oauth" ? (
            <>
              <McpField label="OAuth issuer URL">
                <Input
                  value={draft.oauthIssuerUrl ?? ""}
                  onChange={(event) =>
                    onDraftChange({ ...draft, oauthIssuerUrl: event.target.value })
                  }
                />
              </McpField>
              <McpField label="OAuth client id">
                <Input
                  value={draft.oauthClientId ?? ""}
                  onChange={(event) =>
                    onDraftChange({ ...draft, oauthClientId: event.target.value })
                  }
                />
              </McpField>
              <McpField label="OAuth scopes">
                <Input
                  value={(draft.oauthScopes ?? []).join(" ")}
                  onChange={(event) =>
                    onDraftChange({ ...draft, oauthScopes: parseWords(event.target.value) })
                  }
                />
              </McpField>
            </>
          ) : null}
          <McpKeyValueRows
            addLabel="添加请求头"
            label="请求头"
            leftPlaceholder="键"
            onChange={onHeaderRowsChange}
            rightPlaceholder="值"
            rows={headerRows}
          />
        </div>
      )}

      <div className="grid gap-3 rounded-md border p-3">
        <McpKeyValueRows
          addLabel="添加环境变量"
          label="环境变量"
          leftPlaceholder="键"
          onChange={onEnvRowsChange}
          rightPlaceholder="值"
          rows={envRows}
        />
        <McpStringRows
          addLabel="添加变量"
          label="环境变量传递"
          onChange={onRootRowsChange}
          placeholder="变量"
          rows={rootRows}
        />
        <McpField label="工作目录">
          <Input
            placeholder="~/code"
            value={draft.cwd ?? ""}
            onChange={(event) => onDraftChange({ ...draft, cwd: event.target.value })}
          />
        </McpField>
        {showEnabledToggle ? (
          <div className="flex items-center justify-between gap-3">
            <Label>保存后启用并探测</Label>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(enabled) => onDraftChange({ ...draft, enabled })}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}
