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
        <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">
          {log}
        </pre>
      ) : null}
      {serverError ? (
        <div className="text-xs text-destructive">{serverError}</div>
      ) : null}
    </>
  );
}
