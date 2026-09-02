declare module "*.cjs" {
type DesktopUpdatePhase =
  | "idle"
  | "checking"
  | "up-to-date"
  | "available"
  | "downloading"
  | "ready"
  | "unsupported"
  | "error";

interface DesktopUpdateState {
  phase: DesktopUpdatePhase;
  currentVersion: string;
  availableVersion: string;
  progress: number;
  message: string;
  errorCode: string;
  automaticChecksEnabled: boolean;
}

interface DesktopUpdater {
  getState(): DesktopUpdateState;
  check(): Promise<DesktopUpdateState>;
  download(): Promise<DesktopUpdateState>;
  install(): Promise<DesktopUpdateState>;
  cancel(): DesktopUpdateState;
  dispose(): void;
}

export function createDesktopUpdater(options: {
  appVersion: string;
  platform: NodeJS.Platform;
  arch: string;
  isPackaged: boolean;
  tempRoot: string;
  shell: {
    openPath(path: string): Promise<string>;
    showItemInFolder(path: string): void;
  };
  onState?: (state: DesktopUpdateState) => void;
}): DesktopUpdater;
}
