import { useEffect, useRef } from "react";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

import {
  createTerminalConnectTicket,
  terminalWebSocketURL,
  type TerminalInfo,
} from "@/services/terminal";

type TerminalViewStatus =
  | "connecting"
  | "running"
  | "reconnecting"
  | "exited"
  | "failed";

export function TerminalView({
  terminal,
  workspaceRoot,
  onCursor,
  onResize,
  onStatus,
  resizePaused = false,
}: {
  terminal: TerminalInfo;
  workspaceRoot: string;
  onCursor: (terminalId: string, cursor: number) => void;
  onResize: (terminalId: string, rows: number, cols: number) => void;
  onStatus: (terminalId: string, status: TerminalViewStatus) => void;
  resizePaused?: boolean;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const cursorRef = useRef(terminal.cursor ?? -1);
  const reconnectAttemptRef = useRef(0);
  const disposedRef = useRef(false);
  const resizeTimerRef = useRef<number>(0);
  const resizePausedRef = useRef(resizePaused);
  const pendingResizeRef = useRef(false);

  useEffect(() => {
    resizePausedRef.current = resizePaused;
    if (resizePaused) return;
    if (!pendingResizeRef.current) return;
    pendingResizeRef.current = false;
    window.clearTimeout(resizeTimerRef.current);
    resizeTimerRef.current = window.setTimeout(() => {
      const xterm = xtermRef.current;
      const fit = fitRef.current;
      if (!xterm || !fit || disposedRef.current) return;
      fit.fit();
      const rows = xterm.rows;
      const cols = xterm.cols;
      onResize(terminal.id, rows, cols);
      const socket = socketRef.current;
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(controlFrame({ type: "resize", rows, cols }));
      }
    }, 0);
  }, [onResize, resizePaused, terminal.id]);

  useEffect(() => {
    disposedRef.current = false;
    reconnectAttemptRef.current = 0;
    cursorRef.current = terminal.cursor ?? -1;
    const xterm = new Terminal({
      allowProposedApi: false,
      convertEol: true,
      cursorBlink: true,
      disableStdin: false,
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      fontSize: 12,
      scrollback: 5000,
      theme: terminalThemeFromCSSVariables(),
    });
    const fit = new FitAddon();
    xterm.loadAddon(fit);
    xterm.loadAddon(new WebLinksAddon());
    xterm.open(hostRef.current!);
    applyXtermFullHeightClasses(hostRef.current!);
    fit.fit();
    xterm.focus();
    xtermRef.current = xterm;
    fitRef.current = fit;
    const themeObserver = new MutationObserver(() => {
      xterm.options.theme = terminalThemeFromCSSVariables();
    });
    themeObserver.observe(document.documentElement, {
      attributeFilter: ["class", "style"],
      attributes: true,
    });
    if (document.body) {
      themeObserver.observe(document.body, {
        attributeFilter: ["class", "style"],
        attributes: true,
      });
    }

    const sendResize = () => {
      if (resizePausedRef.current) {
        pendingResizeRef.current = true;
        return;
      }
      fit.fit();
      const rows = xterm.rows;
      const cols = xterm.cols;
      onResize(terminal.id, rows, cols);
      const socket = socketRef.current;
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(
          controlFrame({
            type: "resize",
            rows,
            cols,
          }),
        );
      }
    };
    const resizeObserver = new ResizeObserver(() => {
      window.clearTimeout(resizeTimerRef.current);
      resizeTimerRef.current = window.setTimeout(sendResize, 80);
    });
    if (hostRef.current) resizeObserver.observe(hostRef.current);
    const dataDisposable = xterm.onData((data) => {
      const socket = socketRef.current;
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(data);
      }
    });

    function connect() {
      if (disposedRef.current) return;
      onStatus(
        terminal.id,
        reconnectAttemptRef.current > 0 ? "reconnecting" : "connecting",
      );
      void createTerminalConnectTicket(workspaceRoot, terminal.id)
        .then((ticket) => {
          if (disposedRef.current) return;
          const socket = new WebSocket(
            terminalWebSocketURL(
              workspaceRoot,
              terminal.id,
              ticket,
              cursorRef.current,
            ),
          );
          socket.binaryType = "arraybuffer";
          socketRef.current = socket;
          socket.addEventListener("open", () => {
            reconnectAttemptRef.current = 0;
            onStatus(terminal.id, "running");
            sendResize();
          });
          socket.addEventListener("message", (event) => {
            const bytes = eventBytes(event.data);
            if (bytes.length === 0) return;
            if (bytes[0] === 0) {
              const control = parseControl(bytes.slice(1));
              if (typeof control.cursor === "number") {
                cursorRef.current = control.cursor;
                onCursor(terminal.id, control.cursor);
              }
              if (control.type === "exit") {
                onStatus(terminal.id, "exited");
              }
              return;
            }
            cursorRef.current += bytes.length;
            onCursor(terminal.id, cursorRef.current);
            xterm.write(bytes);
          });
          socket.addEventListener("close", () => {
            if (disposedRef.current) return;
            reconnectAttemptRef.current += 1;
            if (reconnectAttemptRef.current > 4) {
              onStatus(terminal.id, "failed");
              xterm.writeln("\r\n[terminal] connection closed");
              return;
            }
            window.setTimeout(connect, Math.min(2000, reconnectAttemptRef.current * 400));
          });
          socket.addEventListener("error", () => {
            if (disposedRef.current) return;
            onStatus(terminal.id, "failed");
            xterm.writeln("\r\n[terminal] websocket connection failed");
          });
        })
        .catch(() => {
          if (disposedRef.current) return;
          reconnectAttemptRef.current += 1;
          if (reconnectAttemptRef.current > 4) {
            onStatus(terminal.id, "failed");
            xterm.writeln("\r\n[terminal] failed to create connection ticket");
            return;
          }
          window.setTimeout(connect, Math.min(2000, reconnectAttemptRef.current * 400));
        });
    }

    connect();

    return () => {
      disposedRef.current = true;
      window.clearTimeout(resizeTimerRef.current);
      dataDisposable.dispose();
      themeObserver.disconnect();
      resizeObserver.disconnect();
      socketRef.current?.close();
      xterm.dispose();
      xtermRef.current = null;
      fitRef.current = null;
    };
  }, [onCursor, onResize, onStatus, terminal.id, terminal.cursor, workspaceRoot]);

  return (
    <div
      className="h-full min-h-0 w-full overflow-hidden bg-background text-foreground"
      onMouseDown={() => xtermRef.current?.focus()}
      ref={hostRef}
    />
  );
}

function terminalThemeFromCSSVariables() {
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

function applyXtermFullHeightClasses(host: HTMLElement) {
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

function isMacPlatform() {
  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();
  return platform.includes("mac") || userAgent.includes("mac os");
}

function controlFrame(value: Record<string, unknown>) {
  const raw = new TextEncoder().encode(JSON.stringify(value));
  const frame = new Uint8Array(raw.length + 1);
  frame[0] = 0;
  frame.set(raw, 1);
  return frame;
}

function eventBytes(value: unknown) {
  if (typeof value === "string") return new TextEncoder().encode(value);
  if (value instanceof ArrayBuffer) return new Uint8Array(value);
  if (value instanceof Blob) return new Uint8Array();
  return new Uint8Array();
}

function parseControl(bytes: Uint8Array) {
  try {
    return JSON.parse(new TextDecoder().decode(bytes)) as {
      type?: string;
      cursor?: number;
    };
  } catch {
    return {};
  }
}
