import type { RefObject } from "react";

import type { QuestionRequest } from "@/services/aivo";

export type QuestionRequestBusyState = "idle" | "submitting" | "rejecting";

export type QuestionRequestQuestion = QuestionRequest["questions"][number];

export type QuestionRequestPanelProps = {
  activeIndex: number;
  busy: QuestionRequestBusyState;
  customAnswerIndex: number;
  customAnswers: string[];
  customInputRef: RefObject<HTMLInputElement | null>;
  isBusy: boolean;
  onActiveAnswerIndexChange: (index: number) => void;
  onCustomAnswerChange: (index: number, value: string) => void;
  onPrimaryAction: () => void;
  onReject: () => void;
  onSelectAnswer: (index: number, value: string, multiple: boolean) => void;
  onVisibleQuestionIndexChange: (index: number) => void;
  primaryActionContinues: boolean;
  question: QuestionRequestQuestion;
  questionCount: number;
  questionIndex: number;
  selected: string[];
};

export function initialQuestionAnswers(request?: QuestionRequest) {
  return (request?.questions ?? []).map((_, index) =>
    request?.answers?.[index] ? [...request.answers[index]] : [],
  );
}

export function initialQuestionCustomAnswers(request?: QuestionRequest) {
  return (request?.questions ?? []).map((question, index) => {
    const optionLabels = new Set(
      (question.options ?? []).map((option) => option.label),
    );
    return (
      request?.answers?.[index]?.find((answer) => !optionLabels.has(answer)) ??
      ""
    );
  });
}
