import { useEffect, useRef, useState } from "react";
import { ChevronDown } from "lucide-react";

import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  toolFileChangeKey,
  toolFileChangeLabel,
  toolFileChangePath,
  type ToolFileChange,
} from "@/features/projects/conversation-timeline-tool-model";
import { AnimatedDisclosure } from "@/features/projects/conversation-timeline-disclosure";

export function ToolFileChangeLines({
  files,
  live,
}: {
  files: ToolFileChange[];
  live: boolean;
}) {
  const [expandedFileKey, setExpandedFileKey] = useState<string | null>(null);
  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-sm">
      {files.map((file) => {
        const fileKey = toolFileChangeKey(file);
        const expanded = expandedFileKey === fileKey;
        const canExpand = Boolean(file.diff?.trim());
        return (
          <div className="flex min-w-0 flex-col gap-1" key={fileKey}>
            <button
              aria-expanded={canExpand ? expanded : undefined}
              className={cn(
                "flex min-w-0 items-baseline gap-1.5 text-left",
                canExpand &&
                  "cursor-pointer rounded-md px-1 py-0.5 -mx-1 hover:bg-muted/50",
              )}
              disabled={!canExpand}
              onClick={() =>
                setExpandedFileKey((current) =>
                  current === fileKey ? null : fileKey,
                )
              }
              type="button"
            >
              <span className="shrink-0 text-muted-foreground">
                {toolFileChangeLabel(file, live)}
              </span>
              <span className="min-w-0 truncate text-sky-500 dark:text-sky-300">
                {toolFileChangePath(file)}
              </span>
              <span className="inline-flex shrink-0 items-baseline text-emerald-500 dark:text-emerald-400">
                <span>+</span>
                <RollingCount value={file.additions} live={live} direction="up" />
              </span>
              <span className="inline-flex shrink-0 items-baseline text-rose-500 dark:text-rose-400">
                <span>-</span>
                <RollingCount
                  value={file.deletions}
                  live={live}
                  direction="down"
                />
              </span>
              {canExpand ? (
                <ChevronDown
                  className={cn(
                    "shrink-0 self-center text-muted-foreground transition-transform",
                    expanded && "rotate-180",
                  )}
                />
              ) : null}
            </button>
            {canExpand ? (
              <AnimatedDisclosure open={expanded}>
                <ToolDiffPreview text={file.diff} />
              </AnimatedDisclosure>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function ToolDiffPreview({ text }: { text?: string }) {
  if (!text?.trim()) return null;
  return (
    <div className="max-h-80 overflow-hidden border-t border-border/60 pt-2">
      <ScrollArea className="max-h-80 [&_[data-radix-scroll-area-viewport]]:pr-2 [&_[data-slot=scroll-area-viewport]]:max-h-80">
        <pre className="whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">
          {text.split("\n").map((line, index) => (
            <span
              className={cn(
                "block min-h-[1.25em]",
                line.startsWith("+") && !line.startsWith("+++")
                  ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                  : line.startsWith("-") && !line.startsWith("---")
                    ? "bg-rose-500/10 text-rose-700 dark:text-rose-300"
                    : line.startsWith("@@")
                      ? "bg-sky-500/10 text-sky-700 dark:text-sky-300"
                      : "text-muted-foreground",
              )}
              key={`${index}:${line}`}
            >
              {line || " "}
            </span>
          ))}
        </pre>
      </ScrollArea>
    </div>
  );
}

function RollingCount({
  value,
  live,
  direction,
}: {
  value: number;
  live: boolean;
  direction: "up" | "down";
}) {
  const previousValueRef = useRef(value);
  const [roll, setRoll] = useState<{
    from: number;
    to: number;
    active: boolean;
  } | null>(null);

  useEffect(() => {
    const previous = previousValueRef.current;
    if (previous === value) {
      return;
    }
    previousValueRef.current = value;
    if (!live) {
      setRoll(null);
      return;
    }

    setRoll({ from: previous, to: value, active: false });
    const frame = requestAnimationFrame(() => {
      setRoll((current) => (current ? { ...current, active: true } : current));
    });
    const timeout = window.setTimeout(() => setRoll(null), 360);
    return () => {
      cancelAnimationFrame(frame);
      window.clearTimeout(timeout);
    };
  }, [value, live]);

  if (!roll) {
    return (
      <span
        className="inline-block tabular-nums"
        style={{ minWidth: `${String(value).length}ch` }}
      >
        {value}
      </span>
    );
  }

  const width = Math.max(String(roll.from).length, String(roll.to).length);
  const values =
    direction === "up" ? [roll.from, roll.to] : [roll.to, roll.from];
  const translate =
    direction === "up"
      ? roll.active
        ? "translateY(-50%)"
        : "translateY(0)"
      : roll.active
        ? "translateY(0)"
        : "translateY(-50%)";

  return (
    <span
      className="inline-block h-[1em] overflow-hidden align-[-0.08em] tabular-nums"
      style={{ minWidth: `${width}ch` }}
    >
      <span
        className="flex flex-col transition-transform duration-300 ease-out motion-reduce:transition-none"
        style={{ transform: translate }}
      >
        {values.map((item, index) => (
          <span className="h-[1em] leading-none" key={`${item}:${index}`}>
            {item}
          </span>
        ))}
      </span>
    </span>
  );
}
