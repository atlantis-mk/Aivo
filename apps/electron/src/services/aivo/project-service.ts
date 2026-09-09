export async function selectProjectDirectory() {
  const selected = await window.aivoDesktop?.workspace.choose();
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
  return window.aivoDesktop?.composer.selectLocalResource() ?? Promise.resolve(null);
}

export function inspectDroppedComposerResources(files: File[]) {
  return window.aivoDesktop?.composer.inspectDroppedResources(files) ??
    Promise.resolve([] as ComposerLocalSelection[]);
}

export function exportDiagnostics() {
  return Promise.resolve();
}
