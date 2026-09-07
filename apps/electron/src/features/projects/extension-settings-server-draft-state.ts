import { useEffect, useRef, useState } from "react";

import {
  applyGeneratedMcpDescription,
  compactStrings,
  mapToRows,
  mcpServerToDraft,
  normalizeMcpDraft,
  nonEmptyStrings,
  rowsToMap,
  type KeyValueRow,
} from "@/features/projects/extension-settings-model";
import {
  generateMCPDescription,
  saveMCPServer,
  type MCPServerConfig,
} from "@/services/aivo";

export function useMcpServerDraftState({
  onClose,
  onReload,
  open,
  server,
}: {
  onClose: () => void;
  onReload: () => Promise<void>;
  open: boolean;
  server: MCPServerConfig;
}) {
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [generatingDescription, setGeneratingDescription] = useState(false);
  const [descriptionGenerationError, setDescriptionGenerationError] =
    useState("");
  const descriptionGenerationRequest = useRef(0);
  const [draft, setDraft] = useState<MCPServerConfig>(() =>
    mcpServerToDraft(server),
  );
  const [argRows, setArgRows] = useState<string[]>(() =>
    nonEmptyStrings(server.args),
  );
  const [envRows, setEnvRows] = useState<KeyValueRow[]>(() =>
    mapToRows(server.env),
  );
  const [headerRows, setHeaderRows] = useState<KeyValueRow[]>(() =>
    mapToRows(server.headers),
  );
  const [rootRows, setRootRows] = useState<string[]>(() =>
    nonEmptyStrings(server.roots),
  );

  useEffect(() => {
    descriptionGenerationRequest.current += 1;
    setGeneratingDescription(false);
    if (!open) return;
    setDraft(mcpServerToDraft(server));
    setArgRows(nonEmptyStrings(server.args));
    setEnvRows(mapToRows(server.env));
    setHeaderRows(mapToRows(server.headers));
    setRootRows(nonEmptyStrings(server.roots));
    setSaveError("");
    setDescriptionGenerationError("");
  }, [server, open]);

  async function generateDescription() {
    const request = descriptionGenerationRequest.current + 1;
    descriptionGenerationRequest.current = request;
    setGeneratingDescription(true);
    setDescriptionGenerationError("");
    try {
      const result = await generateMCPDescription(server.id);
      if (descriptionGenerationRequest.current !== request) return;
      setDraft((current) =>
        applyGeneratedMcpDescription(current, result.description),
      );
    } catch (err) {
      if (descriptionGenerationRequest.current !== request) return;
      setDescriptionGenerationError(
        err instanceof Error ? err.message : String(err),
      );
    } finally {
      if (descriptionGenerationRequest.current === request) {
        setGeneratingDescription(false);
      }
    }
  }

  async function saveSettings() {
    setSaving(true);
    setSaveError("");
    try {
      await saveMCPServer(
        normalizeMcpDraft({
          ...draft,
          args: compactStrings(argRows),
          env: rowsToMap(envRows),
          headers: rowsToMap(headerRows),
          roots: compactStrings(rootRows),
        }),
      );
      await onReload();
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  return {
    argRows,
    draft,
    descriptionGenerationError,
    envRows,
    headerRows,
    rootRows,
    generateDescription,
    generatingDescription,
    saveError,
    saving,
    saveSettings,
    setArgRows,
    setDraft,
    setEnvRows,
    setHeaderRows,
    setRootRows,
  };
}
