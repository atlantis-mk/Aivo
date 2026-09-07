import { Card, CardContent } from "@/components/ui/card";
import type { QuestionRequestPanelProps } from "@/features/projects/project-question-request-model";
import {
  QuestionRequestAnswerList,
  QuestionRequestPanelFooter,
  QuestionRequestPanelHeader,
} from "@/features/projects/project-question-request-panel-sections";

export function QuestionRequestPanel({
  activeIndex,
  busy,
  customAnswerIndex,
  customAnswers,
  customInputRef,
  isBusy,
  onActiveAnswerIndexChange,
  onCustomAnswerChange,
  onPrimaryAction,
  onReject,
  onSelectAnswer,
  onVisibleQuestionIndexChange,
  primaryActionContinues,
  question,
  questionCount,
  questionIndex,
  selected,
}: QuestionRequestPanelProps) {
  return (
    <div
      className="absolute bottom-4 left-1/2 z-30 w-[calc(100%-2rem)] max-w-[680px] -translate-x-1/2 transition-[margin,transform] duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] sm:bottom-6 sm:w-[calc(100%-48px)]"
      data-assistant-hover-ignore="true"
    >
      <Card className="gap-2 py-2 [--card-spacing:--spacing(2.5)]" size="sm">
        <QuestionRequestPanelHeader
          isBusy={isBusy}
          onVisibleQuestionIndexChange={onVisibleQuestionIndexChange}
          question={question}
          questionCount={questionCount}
          questionIndex={questionIndex}
        />
        <CardContent>
          <QuestionRequestAnswerList
            activeIndex={activeIndex}
            customAnswerIndex={customAnswerIndex}
            customAnswers={customAnswers}
            customInputRef={customInputRef}
            isBusy={isBusy}
            onActiveAnswerIndexChange={onActiveAnswerIndexChange}
            onCustomAnswerChange={onCustomAnswerChange}
            onSelectAnswer={onSelectAnswer}
            question={question}
            questionIndex={questionIndex}
            selected={selected}
          />
        </CardContent>
        <QuestionRequestPanelFooter
          busy={busy}
          isBusy={isBusy}
          onPrimaryAction={onPrimaryAction}
          onReject={onReject}
          primaryActionContinues={primaryActionContinues}
        />
      </Card>
    </div>
  );
}
