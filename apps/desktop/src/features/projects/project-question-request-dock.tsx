import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import {
  initialQuestionAnswers,
  initialQuestionCustomAnswers,
  type QuestionRequestBusyState,
} from "@/features/projects/project-question-request-model";
import { QuestionRequestPanel } from "@/features/projects/project-question-request-panel";
import {
  rejectQuestionRequest,
  replyQuestionRequest,
  type QuestionRequest,
} from "@/services/aivo";

export function QuestionRequestDock({
  request,
}: {
  request?: QuestionRequest;
}) {
  const [answers, setAnswers] = useState<string[][]>(() =>
    initialQuestionAnswers(request),
  );
  const [customAnswers, setCustomAnswers] = useState<string[]>(() =>
    initialQuestionCustomAnswers(request),
  );
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [activeAnswerIndex, setActiveAnswerIndex] = useState(0);
  const [busy, setBusy] = useState<QuestionRequestBusyState>("idle");
  const customInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setAnswers(initialQuestionAnswers(request));
    setCustomAnswers(initialQuestionCustomAnswers(request));
    setCurrentQuestionIndex(0);
    setActiveAnswerIndex(0);
    setBusy("idle");
  }, [request]);

  useEffect(() => {
    if (!request?.id) return;
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target;
      const isTextInput =
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        (target instanceof HTMLElement && target.isContentEditable);

      if (event.key === "Escape") {
        event.preventDefault();
        void reject();
        return;
      }

      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        if (isTextInput && target instanceof HTMLElement) target.blur();
        setActiveAnswerIndex((current) => {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          return (current + direction + answerItemCount) % answerItemCount;
        });
        return;
      }

      if (isTextInput) {
        if (event.key === "Enter") {
          event.preventDefault();
          void handlePrimaryAction();
        }
        return;
      }

      if (event.key === "Enter") {
        event.preventDefault();
        if (question.multiple) {
          void handlePrimaryAction();
        } else {
          selectActiveAnswer();
        }
        return;
      }

      if (event.key === " " && question.multiple) {
        event.preventDefault();
        selectActiveAnswer();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  if (!request || request.questions.length === 0) return null;

  const currentRequest = request;
  const isBusy = busy !== "idle";
  const questionCount = request.questions.length;
  const questionIndex = Math.min(currentQuestionIndex, questionCount - 1);
  const question = request.questions[questionIndex]!;
  const selected = answers[questionIndex] ?? [];
  const optionCount = question.options?.length ?? 0;
  const customAnswerIndex = optionCount;
  const answerItemCount = optionCount + 1;
  const activeIndex = Math.min(activeAnswerIndex, customAnswerIndex);
  const hasNextQuestion = questionIndex < questionCount - 1;
  const primaryActionContinues = hasNextQuestion;

  function setVisibleQuestionIndex(index: number) {
    setCurrentQuestionIndex(index);
    setActiveAnswerIndex(0);
  }

  function setQuestionAnswer(index: number, value: string, multiple: boolean) {
    setCustomAnswers((current) => {
      if (!current[index]) return current;
      const next = [...current];
      next[index] = "";
      return next;
    });
    setAnswers((current) => {
      const next = [...current];
      const existing = next[index] ?? [];
      if (multiple) {
        next[index] = existing.includes(value)
          ? existing.filter((item) => item !== value)
          : [...existing, value];
      } else {
        next[index] = [value];
      }
      return next;
    });
    if (!multiple && index < questionCount - 1) {
      setVisibleQuestionIndex(index + 1);
    }
  }

  function selectActiveAnswer() {
    if (isBusy) return;
    const activeOption = question.options?.[activeIndex];
    if (activeOption) {
      setQuestionAnswer(
        questionIndex,
        activeOption.label,
        Boolean(question.multiple),
      );
      return;
    }
    customInputRef.current?.focus();
  }

  function setCustomAnswer(index: number, value: string) {
    setCustomAnswers((current) => {
      const next = [...current];
      next[index] = value;
      return next;
    });
    const trimmed = value.trim();
    setAnswers((current) => {
      const next = [...current];
      next[index] = trimmed ? [trimmed] : [];
      return next;
    });
  }

  async function submit() {
    if (isBusy) return;
    setBusy("submitting");
    try {
      await replyQuestionRequest(
        currentRequest.id,
        currentRequest.questions.map((_, index) => answers[index] ?? []),
      );
    } catch (err) {
      setBusy("idle");
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  async function handlePrimaryAction() {
    if (isBusy) return;
    if (primaryActionContinues) {
      setVisibleQuestionIndex(questionIndex + 1);
      return;
    }
    await submit();
  }

  async function reject() {
    if (isBusy) return;
    setBusy("rejecting");
    try {
      await rejectQuestionRequest(currentRequest.id, "Dismissed by user");
    } catch (err) {
      setBusy("idle");
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <QuestionRequestPanel
      activeIndex={activeIndex}
      busy={busy}
      customAnswerIndex={customAnswerIndex}
      customAnswers={customAnswers}
      customInputRef={customInputRef}
      isBusy={isBusy}
      onActiveAnswerIndexChange={setActiveAnswerIndex}
      onCustomAnswerChange={setCustomAnswer}
      onPrimaryAction={() => void handlePrimaryAction()}
      onReject={() => void reject()}
      onSelectAnswer={setQuestionAnswer}
      onVisibleQuestionIndexChange={setVisibleQuestionIndex}
      primaryActionContinues={primaryActionContinues}
      question={question}
      questionCount={questionCount}
      questionIndex={questionIndex}
      selected={selected}
    />
  );
}
