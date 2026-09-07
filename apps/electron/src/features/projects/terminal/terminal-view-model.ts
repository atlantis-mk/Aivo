import type { TerminalRuntimeStatus } from "@/features/projects/terminal/terminal-types";

export type TerminalViewStatus = TerminalRuntimeStatus;

export function terminalThemeFromCSSVariables() {
  const style = getComputedStyle(document.documentElement);
  const cssVar = (name: string, fallback: string) =>
    style.getPropertyValue(name).trim() || fallback;
  return {
    background: cssVar("--background", "Canvas"),
    foreground: cssVar("--foreground", "CanvasText"),
    cursor: cssVar("--foreground", "CanvasText"),
    selectionBackground: cssVar("--muted", "Highlight"),
  };
}

export function applyXtermFullHeightClasses(host: HTMLElement) {
  const terminalElement = host.querySelector<HTMLElement>(".xterm");
  if (terminalElement) {
    terminalElement.classList.add("h-full");
    terminalElement.style.height = "100%";
  }

  if (!isMacPlatform()) return;
  host
    .querySelector<HTMLElement>(".xterm-scrollable-element")
    ?.classList.add("h-full");
}

export function controlFrame(value: Record<string, unknown>) {
  const raw = new TextEncoder().encode(JSON.stringify(value));
  const frame = new Uint8Array(raw.length + 1);
  frame[0] = 0;
  frame.set(raw, 1);
  return frame;
}

export function eventBytes(value: unknown) {
  if (typeof value === "string") return new TextEncoder().encode(value);
  if (value instanceof ArrayBuffer) return new Uint8Array(value);
  if (value instanceof Blob) return new Uint8Array();
  return new Uint8Array();
}

export function parseControl(bytes: Uint8Array) {
  try {
    return JSON.parse(new TextDecoder().decode(bytes)) as {
      type?: string;
      cursor?: number;
    };
  } catch {
    return {};
  }
}

function isMacPlatform() {
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();
  return platform.includes("mac") || userAgent.includes("mac os");
}
