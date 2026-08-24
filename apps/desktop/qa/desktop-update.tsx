import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { DesktopUpdateSettings } from "../src/features/updates/desktop-update-settings";
import type { DesktopUpdateState } from "../src/features/updates/desktop-update-model";
import "../src/index.css";

const ready: DesktopUpdateState = {
  phase: "ready",
  currentVersion: "0.1.0",
  availableVersion: "0.1.1",
  progress: 100,
  message: "v0.1.1 已下载，并通过 R2 与 GitHub Release 双源完整性校验。",
  errorCode: "",
  automaticChecksEnabled: true,
};

const updates = {
  getState: async () => ready,
  check: async () => ready,
  download: async () => ready,
  install: async () => ready,
  cancel: async () => ready,
  onState: () => () => {},
};

Object.defineProperty(window, "aivo", {
  configurable: true,
  value: { platform: "darwin", updates },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <main className="h-screen bg-background text-foreground">
      <DesktopUpdateSettings />
    </main>
  </StrictMode>,
);
