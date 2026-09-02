import { PreviewBlock } from "@/features/projects/extension-settings-components";
import {
  oauthDiscoveryPreview,
  oauthStartPreview,
  oauthStatusPreview,
  resourcePreview,
  stringifyStructured,
} from "@/features/projects/extension-settings-model";
import type {
  MCPPromptGetResult,
  MCPResourceReadResult,
  MCPOAuthDiscoveryResult,
  MCPOAuthStartResult,
  MCPOAuthStatus,
} from "@/services/aivo";

export function McpServerPreviewStack({
  log,
  logError,
  oauthDiscovery,
  oauthError,
  oauthStart,
  oauthStatus,
  promptError,
  promptResult,
  resourceError,
  resourceResult,
  saveError,
  serverError,
}: {
  log: string;
  logError: string;
  oauthDiscovery: MCPOAuthDiscoveryResult | null;
  oauthError: string;
  oauthStart: MCPOAuthStartResult | null;
  oauthStatus: MCPOAuthStatus | null;
  promptError: string;
  promptResult: MCPPromptGetResult | null;
  resourceError: string;
  resourceResult: MCPResourceReadResult | null;
  saveError: string;
  serverError?: string;
}) {
  return (
    <>
      {saveError ? (
        <div className="text-xs text-destructive">{saveError}</div>
      ) : null}
      {promptError ? (
        <div className="text-xs text-destructive">{promptError}</div>
      ) : null}
      {promptResult ? (
        <PreviewBlock
          label={`Prompt: ${promptResult.name}`}
          value={
            promptResult.content || stringifyStructured(promptResult.structured)
          }
        />
      ) : null}
      {resourceError ? (
        <div className="text-xs text-destructive">{resourceError}</div>
      ) : null}
      {resourceResult ? (
        <PreviewBlock
          label={`Resource: ${resourceResult.uri}`}
          value={resourcePreview(resourceResult)}
        />
      ) : null}
      {oauthError ? (
        <div className="text-xs text-destructive">{oauthError}</div>
      ) : null}
      {oauthDiscovery ? (
        <PreviewBlock
          label="OAuth discovery"
          value={oauthDiscoveryPreview(oauthDiscovery)}
        />
      ) : null}
      {oauthStart ? (
        <PreviewBlock
          label="OAuth authorization"
          value={oauthStartPreview(oauthStart)}
        />
      ) : null}
      {oauthStatus ? (
        <PreviewBlock
          label="OAuth status"
          value={oauthStatusPreview(oauthStatus)}
        />
      ) : null}
      {logError ? (
        <div className="text-xs text-destructive">{logError}</div>
      ) : null}
      {log ? (
        <section className="mt-3 overflow-hidden rounded-2xl border border-border/80 bg-card text-card-foreground shadow-sm shadow-foreground/[0.03]">
          <div className="flex min-h-11 items-center px-4 pt-3 pb-2">
            <div className="min-w-0 truncate text-xs font-semibold text-foreground">
              运行日志
            </div>
          </div>
          <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words px-4 pb-4 pt-1 font-mono text-xs leading-relaxed text-muted-foreground">
            {log}
          </pre>
        </section>
      ) : null}
      {serverError ? (
        <div className="text-xs text-destructive">{serverError}</div>
      ) : null}
    </>
  );
}
