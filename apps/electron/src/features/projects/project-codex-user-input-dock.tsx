import { useCallback, useEffect, useState } from "react";
import { CircleHelp, Send } from "lucide-react";

import { Button } from "@/components/ui/button";
import type {
  CodexUserInputAnswers,
  CodexUserInputRequest,
} from "@/codex-app-server";
import { cn } from "@/lib/utils";

export function useCodexUserInputRequests() {
  const [requests, setRequests] = useState<CodexUserInputRequest[]>([]);

  useEffect(() => {
    let mounted = true;
    const onRequest = ({ request }: { request: unknown }) => {
      if (!mounted || typeof request !== "object" || request === null) return;
      const r = request as CodexUserInputRequest;
      setRequests((current) =>
        current.some((item) => item.itemId === r.itemId)
          ? current
          : [...current, r],
      );
    };
    const onResolved = ({ itemId }: { itemId: string }) => {
      setRequests((current) =>
        current.filter((item) => item.itemId !== itemId),
      );
    };
    const offRequest = window.aivoDesktop.codex.onUserInputRequest(onRequest);
    const offResolved =
      window.aivoDesktop.codex.onUserInputResolved(onResolved);

    return () => {
      mounted = false;
      offRequest();
      offResolved();
    };
  }, []);

  const resolve = useCallback(
    async (itemId: string, answers: CodexUserInputAnswers) => {
      await window.aivoDesktop.codex.resolveUserInput(itemId, answers);
      setRequests((current) =>
        current.filter((item) => item.itemId !== itemId),
      );
    },
    [],
  );

  return { requests, resolve };
}

type SelectedAnswers = Record<string, string[]>;

export function CodexUserInputDock({
  requests,
  resolve,
}: {
  requests: CodexUserInputRequest[];
  resolve: (
    itemId: string,
    answers: CodexUserInputAnswers,
  ) => Promise<void>;
}) {
  const [selected, setSelected] = useState<SelectedAnswers>({});
  const [freeText, setFreeText] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  const active = requests[0];
  if (!active) return null;

  const canSubmit = active.questions.every((q) => {
    const picks = selected[q.id] ?? [];
    const text = freeText[q.id]?.trim() ?? "";
    return picks.length > 0 || text.length > 0;
  });

  const toggleOption = (questionId: string, label: string) => {
    setSelected((current) => {
      const existing = current[questionId] ?? [];
      const updated = existing.includes(label)
        ? existing.filter((item) => item !== label)
        : [...existing, label];
      return { ...current, [questionId]: updated };
    });
  };

  const submit = async () => {
    if (!active || !canSubmit || submitting) return;
    setSubmitting(true);
    try {
      const answers: CodexUserInputAnswers = {};
      for (const q of active.questions) {
        const picks = selected[q.id] ?? [];
        const text = freeText[q.id]?.trim() ?? "";
        const merged = text ? [...picks, text] : picks;
        answers[q.id] = { answers: merged };
      }
      await resolve(active.itemId, answers);
      setSelected({});
      setFreeText({});
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="absolute bottom-4 left-1/2 z-20 w-[calc(100%-2rem)] max-w-[760px] -translate-x-1/2 sm:bottom-6 sm:w-[calc(100%-48px)]">
      <div className="overflow-hidden rounded-2xl border border-border/80 bg-popover text-popover-foreground shadow-2xl shadow-foreground/15 ring-1 ring-foreground/5">
        <div className="flex items-center gap-3 border-b border-border/70 px-4 py-3">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground">
            <CircleHelp className="size-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold">Codex 有问题需要你确认</div>
          </div>
        </div>
        <div className="space-y-4 px-4 py-3">
          {active.questions.map((question) => (
            <div key={question.id} className="space-y-2">
              <div className="text-sm font-medium">{question.question}</div>
              {question.options?.map((option) => {
                const isActive = (selected[question.id] ?? []).includes(
                  option.label,
                );
                return (
                  <button
                    key={option.label}
                    type="button"
                    onClick={() => toggleOption(question.id, option.label)}
                    className={cn(
                      "w-full rounded-lg border px-3 py-2 text-left text-sm transition-colors",
                      isActive
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border hover:bg-accent hover:text-accent-foreground",
                    )}
                  >
                    <div className="font-medium">{option.label}</div>
                    {option.description ? (
                      <div className="mt-0.5 text-muted-foreground">
                        {option.description}
                      </div>
                    ) : null}
                  </button>
                );
              })}
              <input
                type="text"
                value={freeText[question.id] ?? ""}
                onChange={(event) =>
                  setFreeText((current) => ({
                    ...current,
                    [question.id]: event.target.value,
                  }))
                }
                placeholder="或输入自定义回答…"
                className="w-full rounded-lg border border-border bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          ))}
        </div>
        <div className="flex justify-end gap-2 border-t border-border/70 px-4 py-3">
          <Button
            onClick={submit}
            disabled={!canSubmit || submitting}
            size="sm"
          >
            <Send className="mr-1.5 size-3.5" />
            提交
          </Button>
        </div>
      </div>
    </div>
  );
}
