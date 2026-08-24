import {
  ChevronDown,
  Plus,
  TerminalSquare,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { TerminalView } from "@/features/projects/terminal/terminal-view";
import type { WorkspaceTerminalState } from "@/features/projects/terminal/terminal-state";
import type { TerminalRuntimeStatus } from "@/features/projects/terminal/terminal-types";
import { cn } from "@/lib/utils";
import type { TerminalInfo } from "@/services/terminal";

export type TerminalPanelViewStatus = TerminalRuntimeStatus;

export type TerminalPanelViewProps = {
  activeTerminal?: TerminalInfo;
  canUseTerminal: boolean;
  creatingTerminal: boolean;
  error: string;
  onAddTerminal: () => void;
  onCloseTerminal: (terminalId: string) => void;
  onCollapse: () => void;
  onRenameTerminal: (terminal: TerminalInfo) => void;
  onSelectTerminal: (terminalId: string) => void;
  onTerminalCursor: (terminalId: string, cursor: number) => void;
  onTerminalResize: (terminalId: string, rows: number, cols: number) => void;
  onTerminalStatus: (
    terminalId: string,
    status: TerminalPanelViewStatus,
  ) => void;
  state: WorkspaceTerminalState;
  terminals: TerminalInfo[];
  terminalsLoaded: boolean;
  workspaceRoot: string;
};

export function TerminalPanelView({
  activeTerminal,
  canUseTerminal,
  creatingTerminal,
  error,
  onAddTerminal,
  onCloseTerminal,
  onCollapse,
  onRenameTerminal,
  onSelectTerminal,
  onTerminalCursor,
  onTerminalResize,
  onTerminalStatus,
  state,
  terminals,
  terminalsLoaded,
  workspaceRoot,
}: TerminalPanelViewProps) {
  return (
    <section className="flex h-full min-h-0 flex-col overflow-hidden bg-card shadow-xl shadow-foreground/10">
      <div className="flex shrink-0 items-center gap-2 border-t border-border/70 px-4 py-1">
        <Button
          aria-label="收起终端"
          className="size-7"
          onClick={onCollapse}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <ChevronDown />
        </Button>
        <div className="flex min-w-0 flex-1 items-center gap-1.5 overflow-x-auto">
          {state.tabs.map((tab) => {
            const terminal = terminals.find((item) => item.id === tab.id);
            const active = tab.id === activeTerminal?.id;
            return (
              <div
                className={cn(
                  "group/tab relative flex min-w-0 shrink-0 items-center gap-1.5 rounded-lg py-1 text-xs font-medium transition-colors",
                  active
                    ? "bg-muted pl-2 pr-4 text-foreground"
                    : "pl-1.5 pr-3.5 text-muted-foreground hover:bg-muted/60 hover:text-foreground",
                )}
                key={tab.id}
              >
                <button
                  className="flex min-w-0 items-center gap-1.5"
                  onClick={() => onSelectTerminal(tab.id)}
                  onDoubleClick={() => terminal && onRenameTerminal(terminal)}
                  type="button"
                >
                  <TerminalSquare className="shrink-0" />
                  <span className="max-w-28 truncate">
                    {terminal?.title || tab.title}
                  </span>
                </button>
                <Button
                  aria-label="关闭终端"
                  className={cn(
                    "absolute right-1 top-1/2 size-4 shrink-0 !-translate-y-1/2 rounded-full active:!-translate-y-1/2",
                    active
                      ? "invisible inline-flex group-hover/tab:visible focus-visible:visible"
                      : "hidden",
                  )}
                  onClick={() => onCloseTerminal(tab.id)}
                  size="icon-xs"
                  type="button"
                >
                  <X />
                </Button>
              </div>
            );
          })}
          <Button
            aria-label="新建终端"
            className="shrink-0 text-muted-foreground hover:text-foreground"
            onClick={onAddTerminal}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <Plus />
          </Button>
        </div>
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {!canUseTerminal ? (
          <div className="flex h-full items-center justify-center px-4 text-xs text-muted-foreground">
            终端暂不可用
          </div>
        ) : error ? (
          <div className="flex h-full items-center justify-center px-4 text-xs text-destructive">
            {error}
          </div>
        ) : activeTerminal ? (
          <TerminalView
            key={activeTerminal.id}
            onCursor={onTerminalCursor}
            onResize={onTerminalResize}
            onStatus={onTerminalStatus}
            terminal={activeTerminal}
            workspaceRoot={workspaceRoot}
          />
        ) : (
          <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
            {creatingTerminal || !terminalsLoaded
              ? "正在创建终端"
              : "没有终端"}
          </div>
        )}
      </div>
    </section>
  );
}
