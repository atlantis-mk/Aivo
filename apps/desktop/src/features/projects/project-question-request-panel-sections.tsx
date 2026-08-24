import { ArrowLeft, ArrowRight, CornerDownLeft, Pencil } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  CardAction,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { Kbd } from "@/components/ui/kbd";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { QuestionRequestPanelProps } from "@/features/projects/project-question-request-model";
import { cn } from "@/lib/utils";

export function QuestionRequestPanelHeader({
  isBusy,
  onVisibleQuestionIndexChange,
  question,
  questionCount,
  questionIndex,
}: Pick<
  QuestionRequestPanelProps,
  | "isBusy"
  | "onVisibleQuestionIndexChange"
  | "question"
  | "questionCount"
  | "questionIndex"
>) {
  return (
    <CardHeader className="gap-0.5">
      <CardTitle>{question.question || "需要你的选择"}</CardTitle>
      {questionCount > 1 ? (
        <CardAction className="flex items-center gap-2">
          <Button
            aria-label="上一个问题"
            disabled={isBusy || questionIndex === 0}
            onClick={() =>
              onVisibleQuestionIndexChange(Math.max(0, questionIndex - 1))
            }
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ArrowLeft />
          </Button>
          <CardDescription>
            {questionIndex + 1} / {questionCount}
          </CardDescription>
          <Button
            aria-label="下一个问题"
            disabled={isBusy || questionIndex >= questionCount - 1}
            onClick={() =>
              onVisibleQuestionIndexChange(
                Math.min(questionCount - 1, questionIndex + 1),
              )
            }
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ArrowRight />
          </Button>
        </CardAction>
      ) : null}
      {question.header ? (
        <CardDescription>{question.header}</CardDescription>
      ) : null}
    </CardHeader>
  );
}

export function QuestionRequestAnswerList({
  activeIndex,
  customAnswerIndex,
  customAnswers,
  customInputRef,
  isBusy,
  onActiveAnswerIndexChange,
  onCustomAnswerChange,
  onSelectAnswer,
  question,
  questionIndex,
  selected,
}: Pick<
  QuestionRequestPanelProps,
  | "activeIndex"
  | "customAnswerIndex"
  | "customAnswers"
  | "customInputRef"
  | "isBusy"
  | "onActiveAnswerIndexChange"
  | "onCustomAnswerChange"
  | "onSelectAnswer"
  | "question"
  | "questionIndex"
  | "selected"
>) {
  return (
    <ScrollArea className="max-h-[min(58vh,460px)]">
      <ItemGroup className="!gap-1" data-size="xs">
        {(question.options ?? []).map((option, optionIndex) => {
          const picked = selected.includes(option.label);
          const active = activeIndex === optionIndex;
          return (
            <Item
              asChild
              key={`${option.label}:${optionIndex}`}
              variant={active ? "muted" : "default"}
            >
              <button
                disabled={isBusy}
                onClick={() =>
                  onSelectAnswer(
                    questionIndex,
                    option.label,
                    Boolean(question.multiple),
                  )
                }
                onMouseEnter={() => onActiveAnswerIndexChange(optionIndex)}
                type="button"
              >
                <ItemMedia
                  className={cn(
                    "size-5 !translate-y-0 rounded-full border text-xs font-medium leading-none !self-center",
                    picked
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-border text-muted-foreground",
                  )}
                >
                  {optionIndex + 1}
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>{option.label}</ItemTitle>
                  {option.description ? (
                    <ItemDescription>{option.description}</ItemDescription>
                  ) : null}
                </ItemContent>
              </button>
            </Item>
          );
        })}
        <Item asChild variant={activeIndex === customAnswerIndex ? "muted" : "default"}>
          <label
            onMouseEnter={() => onActiveAnswerIndexChange(customAnswerIndex)}
          >
            <ItemMedia
              className={cn(
                "size-5 !translate-y-0 !self-center",
                customAnswers[questionIndex]
                  ? "text-primary"
                  : "border-border text-muted-foreground",
              )}
              variant="icon"
            >
              <Pencil />
            </ItemMedia>
            <ItemContent>
              <Input
                disabled={isBusy}
                onChange={(event) =>
                  onCustomAnswerChange(questionIndex, event.target.value)
                }
                onFocus={() => onActiveAnswerIndexChange(customAnswerIndex)}
                placeholder="否，请告知 Aivo 如何调整"
                ref={customInputRef}
                value={customAnswers[questionIndex] ?? ""}
              />
            </ItemContent>
          </label>
        </Item>
      </ItemGroup>
    </ScrollArea>
  );
}

export function QuestionRequestPanelFooter({
  busy,
  isBusy,
  onPrimaryAction,
  onReject,
  primaryActionContinues,
}: Pick<
  QuestionRequestPanelProps,
  "busy" | "isBusy" | "onPrimaryAction" | "onReject" | "primaryActionContinues"
>) {
  return (
    <CardFooter className="justify-end gap-2">
      <Button disabled={isBusy} onClick={onReject} type="button" variant="ghost">
        忽略
        <Kbd>ESC</Kbd>
      </Button>
      <Button disabled={isBusy} onClick={onPrimaryAction} type="button">
        {primaryActionContinues
          ? "继续"
          : busy === "submitting"
            ? "提交中"
            : "提交"}
        {primaryActionContinues ? <ArrowRight /> : <CornerDownLeft />}
      </Button>
    </CardFooter>
  );
}
