import { useEffect, useState } from "react";

import {
  compactStrings,
  mapToRows,
  mcpServerToDraft,
  normalizeMcpDraft,
  nonEmptyStrings,
  rowsToMap,
  type KeyValueRow,
} from "@/features/projects/extension-settings-model";
import { saveMCPServer, type MCPServerConfig } from "@/services/aivo";

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
    if (!open) return;
    setDraft(mcpServerToDraft(server));
    setArgRows(nonEmptyStrings(server.args));
    setEnvRows(mapToRows(server.env));
    setHeaderRows(mapToRows(server.headers));
    setRootRows(nonEmptyStrings(server.roots));
    setSaveError("");
  }, [server, open]);

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
    envRows,
    headerRows,
    rootRows,
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
