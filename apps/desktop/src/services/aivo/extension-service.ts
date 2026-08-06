import { invoke } from "@/services/aivo/invoke";

export type ExtensionInstallSummary = {
  id: string;
  name: string;
  version: string;
  description?: string;
  apiVersion: string;
  runtimeType: "builtin" | "process" | "service" | "external" | "static";
  transport?: string;
  command?: string;
  permissions?: string[];
  credentialIds?: string[];
  platforms?: string[];
  network?: boolean;
  tools?: string[];
  views?: string[];
  contexts?: string[];
  policies?: string[];
  executable: boolean;
};

export type ExtensionInstallPreview = {
  path: string;
  manifestPath: string;
  integrity: string;
  summary: ExtensionInstallSummary;
  update: boolean;
};

export type ExtensionInstall = {
  id: string;
  summary: ExtensionInstallSummary;
  installMode: "linked" | "managed";
  rootPath: string;
  manifestPath: string;
  integrity: string;
  enabled: boolean;
  status: string;
  error?: string;
  timeCreated: string;
  timeUpdated: string;
};

export function previewExtensionInstall(path: string) {
  return invoke<ExtensionInstallPreview>("PreviewExtensionInstall", { path });
}

export function installExtension(
  path: string,
  integrity: string,
  enable: boolean,
) {
  return invoke<ExtensionInstall>("InstallExtension", {
    path,
    integrity,
    enable,
  });
}

export function listExtensionInstalls() {
  return invoke<ExtensionInstall[]>("ListExtensionInstalls");
}

export function setExtensionInstalledEnabled(id: string, enabled: boolean) {
  return invoke<ExtensionInstall>("SetExtensionInstalledEnabled", {
    id,
    enabled,
  });
}

export function uninstallExtension(id: string) {
  return invoke<{ uninstalled: boolean }>("UninstallExtension", { id });
}
