import { LoaderCircle, Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";
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
import { Textarea } from "@/components/ui/textarea";
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
  descriptionGenerationError,
  descriptionGenerating = false,
  onArgRowsChange,
  onDraftChange,
  onEnvRowsChange,
  onHeaderRowsChange,
  onGenerateDescription,
  onRootRowsChange,
  rootRows,
  showEnabledToggle = false,
  transportEditable = false,
}: {
  argRows: string[];
  draft: MCPServerConfig;
  envRows: KeyValueRow[];
  headerRows: KeyValueRow[];
  descriptionGenerationError?: string;
  descriptionGenerating?: boolean;
  onArgRowsChange: (rows: string[]) => void;
  onDraftChange: (draft: MCPServerConfig) => void;
  onEnvRowsChange: (rows: KeyValueRow[]) => void;
  onHeaderRowsChange: (rows: KeyValueRow[]) => void;
  onGenerateDescription?: () => void;
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
        <McpField
          action={
            onGenerateDescription ? (
              <Button
                disabled={descriptionGenerating}
                onClick={onGenerateDescription}
                size="sm"
                title="读取该 MCP 当前已发现的全部工具，并使用辅助模型生成描述"
                type="button"
                variant="outline"
              >
                {descriptionGenerating ? (
                  <LoaderCircle className="animate-spin" />
                ) : (
                  <Sparkles />
                )}
                {descriptionGenerating ? "生成中" : "AI 生成"}
              </Button>
            ) : undefined
          }
          label="功能描述"
        >
          <div className="grid gap-1.5">
            <Textarea
              className="min-h-20 resize-y"
              maxLength={500}
              placeholder="例如：查询、创建和更新 Linear 中的 issue、项目与团队信息"
              value={draft.description ?? ""}
              onChange={(event) =>
                onDraftChange({ ...draft, description: event.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">
              可选；用于辅助模型理解整个 MCP 的能力。没有描述可留空，请勿填写密钥或连接信息。
            </p>
            {descriptionGenerationError ? (
              <p className="text-xs text-destructive" role="alert">
                {descriptionGenerationError}
              </p>
            ) : null}
          </div>
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
              value={
                draft.authType === "bearer" && draft.bearerAuthMode === "env"
                  ? "bearer_env"
                  : (draft.authType ?? "none")
              }
              onValueChange={(authType) =>
                onDraftChange({
                  ...draft,
                  authType:
                    authType === "bearer_env"
                      ? "bearer"
                      : (authType as MCPServerConfig["authType"]),
                  bearerAuthMode:
                    authType === "bearer_env" ? "env" : "direct",
                  bearerToken:
                    authType === "bearer" ? draft.bearerToken : "",
                  bearerTokenEnv:
                    authType === "bearer_env" ? draft.bearerTokenEnv : "",
                })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="认证方式" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">No auth</SelectItem>
                  <SelectItem value="bearer">直接输入 Bearer Token</SelectItem>
                  <SelectItem value="bearer_env">Bearer 环境变量</SelectItem>
                  <SelectItem value="oauth">OAuth bearer env</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </McpField>
          {draft.authType === "bearer" && draft.bearerAuthMode !== "env" ? (
            <McpField label="Bearer Token">
              <div className="grid gap-1.5">
                <Input
                  autoComplete="new-password"
                  placeholder={
                    draft.bearerTokenRef
                      ? "已安全保存；留空保持不变"
                      : "输入 Bearer Token"
                  }
                  spellCheck={false}
                  type="password"
                  value={draft.bearerToken ?? ""}
                  onChange={(event) =>
                    onDraftChange({ ...draft, bearerToken: event.target.value })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Token 仅在保存时发送给本地 Core，之后不会回显。
                </p>
              </div>
            </McpField>
          ) : null}
          {(draft.authType === "bearer" && draft.bearerAuthMode === "env") ||
          draft.authType === "oauth" ? (
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
