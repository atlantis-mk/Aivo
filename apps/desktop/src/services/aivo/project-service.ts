export async function selectProjectDirectory() {
  const selected = window.aivoDesktop?.workspace
    ? await window.aivoDesktop.workspace.choose()
    : await window.aivo.selectProjectDirectory();
  return selected || "";
}

export type ComposerLocalSelection =
  | { kind: "directory"; path: string }
  | {
      kind: "file";
      name: string;
      mimeType: string;
      size: number;
      data: string;
    };

export function selectComposerFileOrDirectory() {
  return window.aivo.selectComposerFileOrDirectory() as Promise<
    ComposerLocalSelection | null
  >;
}

export function inspectDroppedComposerResources(files: File[]) {
  return window.aivo.inspectDroppedComposerResources(files) as Promise<
    ComposerLocalSelection[]
  >;
}

export function exportDiagnostics() {
  return window.aivo.exportDiagnostics();
}
