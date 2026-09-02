import { useEffect, useRef, useState } from "react";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  agentTerminalWebSocketURL,
  createAgentTerminalConnectTicket,
  releaseAgentTerminalInput,
  resolveAgentTerminalInput,
  type AgentTerminalSnapshot,
} from "@/services/agent-terminal";
import type {
  AgentTerminalInputMode,
  AgentTerminalInputRequest,
} from "@/features/projects/tool-activity-types";
import {
  applyXtermFullHeightClasses,
  controlFrame,
  eventBytes,
  terminalThemeFromCSSVariables,
} from "@/features/projects/terminal/terminal-view-model";

export function AgentTerminalView({
  initialInputMode = "ask",
  initialInputRequest,
  processRef,
  sessionId,
  workspaceRoot,
}: {
  initialInputMode?: AgentTerminalInputMode;
  initialInputRequest?: AgentTerminalInputRequest;
  processRef: string;
  sessionId: string;
  workspaceRoot: string;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const cursorRef = useRef(-1);
  const disposedRef = useRef(false);
  const exitedRef = useRef(false);
  const userCanTypeRef = useRef(false);
  const pendingUserInputRef = useRef("");
  const inputRequestRef = useRef<AgentTerminalInputRequest | undefined>(initialInputRequest);
  const acquiringUserInputRef = useRef(false);
  const userAcquireRetriedRef = useRef(false);
  const [status, setStatus] = useState<AgentTerminalSnapshot["status"]>("running");
  const [inputMode, setInputMode] = useState<AgentTerminalInputMode>(initialInputMode);
  const [inputRequest, setInputRequest] = useState<AgentTerminalInputRequest | undefined>(initialInputRequest);
  const [decisionPending, setDecisionPending] = useState(false);
  const [manualDecisionOpen, setManualDecisionOpen] = useState(false);
  const [error, setError] = useState("");
  const [attention, setAttention] = useState<AgentTerminalSnapshot["attention"]>("none");
  const [inputOwner, setInputOwner] = useState<AgentTerminalSnapshot["inputOwner"]>("none");
  const [leaseVersion, setLeaseVersion] = useState(0);
  const userCanType = (inputOwner === "user" || inputMode === "user_once") && status !== "exited";

  useEffect(() => {
    const terminal = xtermRef.current;
    if (!terminal) return;
    if (status === "exited") {
      userCanTypeRef.current = false;
      terminal.options.disableStdin = true;
      terminal.options.cursorBlink = false;
      terminal.options.cursorInactiveStyle = "none";
      terminal.blur();
      return;
    }
    terminal.options.disableStdin = false;
    terminal.options.cursorBlink = true;
    terminal.options.cursorInactiveStyle = "outline";
    userCanTypeRef.current = userCanType;
    if (userCanType) terminal.focus();
  }, [status, userCanType]);

  useEffect(() => {
    disposedRef.current = false;
    const terminal = new Terminal({
      convertEol: true,
      cursorBlink: true,
      disableStdin: false,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      fontSize: 12,
      scrollback: 5000,
      theme: terminalThemeFromCSSVariables(),
    });
    const fit = new FitAddon();
    terminal.loadAddon(fit);
    terminal.loadAddon(new WebLinksAddon());
    terminal.open(hostRef.current!);
    applyXtermFullHeightClasses(hostRef.current!);
    fit.fit();
    xtermRef.current = terminal;

    const resizeObserver = new ResizeObserver(() => {
      fit.fit();
      if (socketRef.current?.readyState === WebSocket.OPEN) {
        socketRef.current.send(controlFrame({ type: "resize", rows: terminal.rows, cols: terminal.cols }));
      }
    });
    resizeObserver.observe(hostRef.current!);
    const flushPendingUserInput = () => {
      const pending = pendingUserInputRef.current;
      if (!pending || socketRef.current?.readyState !== WebSocket.OPEN) return;
      pendingUserInputRef.current = "";
      socketRef.current.send(pending);
    };
    const acquireUserInput = () => {
      if (acquiringUserInputRef.current || exitedRef.current) return;
      const socket = socketRef.current;
      if (socket?.readyState !== WebSocket.OPEN) return;
      acquiringUserInputRef.current = true;
      userAcquireRetriedRef.current = false;
      setError("");
      const request = inputRequestRef.current;
      socket.send(controlFrame({
        type: "acquire_input",
        mode: "user_once",
        requestId: request && !request.resolved ? request.id : "",
      }));
    };
    const dataDisposable = terminal.onData((data) => {
      if (exitedRef.current) return;
      if (!userCanTypeRef.current) {
        pendingUserInputRef.current = `${pendingUserInputRef.current}${data}`.slice(0, 4096);
        acquireUserInput();
        return;
      }
      if (socketRef.current?.readyState === WebSocket.OPEN) {
        socketRef.current.send(data);
      } else {
        pendingUserInputRef.current = `${pendingUserInputRef.current}${data}`.slice(0, 4096);
      }
    });

    let reconnects = 0;
    const connect = async () => {
      if (disposedRef.current) return;
      try {
        const ticket = await createAgentTerminalConnectTicket(workspaceRoot, sessionId, processRef);
        if (disposedRef.current) return;
        const socket = new WebSocket(agentTerminalWebSocketURL(workspaceRoot, sessionId, processRef, ticket, cursorRef.current));
        socket.binaryType = "arraybuffer";
        socketRef.current = socket;
        socket.addEventListener("open", () => {
          reconnects = 0;
          setError("");
          socket.send(controlFrame({ type: "resize", rows: terminal.rows, cols: terminal.cols }));
          if (userCanTypeRef.current) flushPendingUserInput();
          else if (pendingUserInputRef.current) acquireUserInput();
        });
        socket.addEventListener("message", (event) => {
          const bytes = eventBytes(event.data);
          if (bytes.length === 0) return;
          if (bytes[0] !== 0) {
            cursorRef.current += bytes.length;
            terminal.write(bytes);
            return;
          }
          try {
            const control = JSON.parse(new TextDecoder().decode(bytes.slice(1))) as AgentTerminalSnapshot;
            if (typeof control.cursor === "number") cursorRef.current = control.cursor;
            if (control.status) {
              exitedRef.current = control.status === "exited";
              setStatus(control.status);
            }
            if (control.inputMode) setInputMode(control.inputMode);
            if (control.attention) setAttention(control.attention);
            if (control.inputOwner) setInputOwner(control.inputOwner);
            if (typeof control.leaseVersion === "number") setLeaseVersion(control.leaseVersion);
            if (control.inputRequest !== undefined) {
              inputRequestRef.current = control.inputRequest ?? undefined;
              setInputRequest(control.inputRequest ?? undefined);
            }
            if (control.type === "input_granted" && control.inputOwner === "user") {
              acquiringUserInputRef.current = false;
              userAcquireRetriedRef.current = false;
              userCanTypeRef.current = true;
              setDecisionPending(false);
              setManualDecisionOpen(false);
              const request = inputRequestRef.current;
              if (request && !request.resolved) {
                const resolved = { ...request, mode: "user_once" as const, resolved: true };
                inputRequestRef.current = resolved;
                setInputRequest(resolved);
              }
              flushPendingUserInput();
              terminal.focus();
            }
            if (control.type === "exit") { exitedRef.current = true; setStatus("exited"); }
            if (control.type === "error" || control.type === "input_rejected") {
              const shouldRetryTakeover = control.type === "input_rejected"
                && acquiringUserInputRef.current
                && !userAcquireRetriedRef.current
                && Boolean(pendingUserInputRef.current)
                && socket.readyState === WebSocket.OPEN
                && !exitedRef.current;
              acquiringUserInputRef.current = false;
              setDecisionPending(false);
              if (shouldRetryTakeover) {
                userAcquireRetriedRef.current = true;
                acquiringUserInputRef.current = true;
                socket.send(controlFrame({ type: "acquire_input", mode: "user_once", requestId: "" }));
                return;
              }
              setError(control.message || "Terminal input was rejected");
            }
          } catch {
            setError("Invalid terminal control message");
          }
        });
        socket.addEventListener("close", () => {
          acquiringUserInputRef.current = false;
          setDecisionPending(false);
          if (disposedRef.current || exitedRef.current) return;
          reconnects += 1;
          if (reconnects > 4) { setError("Terminal connection closed"); return; }
          window.setTimeout(connect, Math.min(2000, reconnects * 400));
        });
      } catch {
        if (!disposedRef.current) setError("Unable to connect to the shared terminal");
      }
    };
    void connect();
    return () => {
      disposedRef.current = true;
      dataDisposable.dispose();
      resizeObserver.disconnect();
      socketRef.current?.close();
      acquiringUserInputRef.current = false;
      terminal.dispose();
      xtermRef.current = null;
    };
  }, [processRef, sessionId, workspaceRoot]);

  const decide = async (mode: Exclude<AgentTerminalInputMode, "ask">) => {
    if (decisionPending) return;
    setDecisionPending(true);
    setError("");
    if (mode === "user_once" && socketRef.current?.readyState === WebSocket.OPEN) {
      acquiringUserInputRef.current = true;
      socketRef.current.send(controlFrame({
        type: "acquire_input",
        mode,
        requestId: inputRequest?.resolved ? "" : (inputRequest?.id ?? ""),
      }));
      return;
    }
    try {
      await resolveAgentTerminalInput({ workspaceRoot, sessionId, processRef, requestId: inputRequest?.resolved ? "" : (inputRequest?.id ?? ""), mode });
      setInputMode(mode);
      setInputOwner(mode === "user_once" ? "user" : "agent");
      if (inputRequest && !inputRequest.resolved) {
        const resolved = { ...inputRequest, mode, resolved: true };
        inputRequestRef.current = resolved;
        setInputRequest(resolved);
      }
      setManualDecisionOpen(false);
      if (mode === "user_once") {
        window.setTimeout(() => {
          xtermRef.current?.focus();
          const pending = pendingUserInputRef.current;
          pendingUserInputRef.current = "";
          if (pending && socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(pending);
        }, 0);
      } else {
        pendingUserInputRef.current = "";
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to assign terminal input");
    } finally {
      setDecisionPending(false);
    }
  };

  const requestOpen = manualDecisionOpen;
  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      <div className="flex h-8 shrink-0 items-center justify-between border-b px-2 text-xs text-muted-foreground">
        <span>{status === "waiting_input" ? "明确等待输入" : status === "exited" ? "进程已结束" : attention === "interactive" ? "交互程序" : attention === "possibly_waiting" ? "可能等待输入" : inputMode === "agent_always" ? "Agent 全程输入" : userCanType ? "由你输入" : inputOwner === "agent" ? "Agent 输入" : "共享终端"}</span>
        <div className="flex min-w-0 items-center gap-2">
          {error ? <span className="truncate text-destructive">{error}</span> : null}
          {!userCanType && status !== "exited" ? (
            <Button disabled={decisionPending} onClick={() => setManualDecisionOpen(true)} size="xs" variant="outline">
              选择输入方
            </Button>
          ) : null}
          {userCanType ? (
            <Button
              disabled={decisionPending}
              onClick={() => void releaseAgentTerminalInput({ workspaceRoot, sessionId, processRef, leaseVersion }).then((snapshot) => {
                setInputMode(snapshot.inputMode ?? "ask");
                setInputOwner(snapshot.inputOwner ?? "none");
                setLeaseVersion(snapshot.leaseVersion ?? leaseVersion);
              }).catch((cause) => setError(cause instanceof Error ? cause.message : "Unable to release terminal input"))}
              size="xs"
              variant="outline"
            >
              释放输入
            </Button>
          ) : null}
        </div>
      </div>
      <div
        className="min-h-0 flex-1 overflow-hidden"
        onMouseDown={() => {
          if (status === "exited") {
            window.setTimeout(() => xtermRef.current?.blur(), 0);
            return;
          }
          xtermRef.current?.focus();
        }}
        ref={hostRef}
      />
      <Dialog open={requestOpen} onOpenChange={(open) => { if (!open && manualDecisionOpen) setManualDecisionOpen(false); }}>
        <DialogContent
          onEscapeKeyDown={(event) => { if (!manualDecisionOpen) event.preventDefault(); }}
          onInteractOutside={(event) => { if (!manualDecisionOpen) event.preventDefault(); }}
          showCloseButton={manualDecisionOpen}
        >
          <DialogHeader>
            <DialogTitle>终端正在等待输入</DialogTitle>
            <DialogDescription>选择由谁回答当前提示。选择“Agent 全程处理”后，这个进程后续提示不再逐次询问。</DialogDescription>
          </DialogHeader>
          {error ? <p className="text-destructive">{error}</p> : null}
          <DialogFooter className="sm:grid sm:grid-cols-3">
            <Button disabled={decisionPending} onClick={() => void decide("user_once")} variant="outline">我来输入</Button>
            <Button disabled={decisionPending} onClick={() => void decide("agent_once")} variant="secondary">交给 Agent</Button>
            <Button disabled={decisionPending} onClick={() => void decide("agent_always")}>Agent 全程处理</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
