import type { domain } from "../../../bridge/go/models";
import { invoke } from "@/services/aivo/invoke";

export async function selectProjectDirectory() {
  const selected = await window.aivo.selectProjectDirectory();
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

export function listRecentProjects(limit: number) {
  return invoke<domain.AssistantProject[]>("ListRecentProjects", limit);
}

export function upsertProject(path: string) {
  return invoke<domain.AssistantProject>("UpsertProject", path);
}

export function setProjectSidebarHidden(path: string, hidden: boolean) {
  return invoke<domain.AssistantProject>("SetProjectSidebarHidden", path, hidden);
}
