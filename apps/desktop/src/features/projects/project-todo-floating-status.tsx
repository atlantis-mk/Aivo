import { Check, Circle } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { isTodoDone } from "@/features/projects/project-todo-status-model";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import type { TodoItem } from "@/services/aivo";

export function TodoFloatingStatus({ todoItems }: { todoItems: TodoItem[] }) {
  const todos = todoItems;
  if (todos.length === 0) return null;

  const runningIndex = todos.findIndex((todo) => todo.status === "in_progress");
  const openIndex = todos.findIndex((todo) => !isTodoDone(todo.status));
  const currentIndex =
    runningIndex >= 0
      ? runningIndex
      : openIndex >= 0
        ? openIndex
        : todos.length - 1;
  const currentTodo = todos[currentIndex] ?? todos[0];

  return (
    <HoverCard openDelay={100}>
      <HoverCardTrigger asChild>
        <Button type="button">
          {todoStatusIcon(currentTodo.status)}
          <span>
            第 {currentIndex + 1} / {todos.length} 步
          </span>
        </Button>
      </HoverCardTrigger>
      <HoverCardContent side="top">
        <div className="flex flex-col gap-2">
          <div className="font-medium">计划列表</div>
          <Separator />
          <div className="flex flex-col gap-2">
            {todos.map((todo) => (
              <div className="flex min-w-0 items-start gap-2" key={todo.id}>
                {todoStatusIcon(todo.status)}
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">
                    {todo.title || "待办"}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function todoStatusIcon(status: string) {
  const iconClassName = "size-4";
  const icon = isTodoDone(status) ? (
    <Check className={iconClassName} />
  ) : status === "in_progress" ? (
    <Spinner className={iconClassName} />
  ) : (
    <Circle className={iconClassName} />
  );
  return (
    <span className="flex size-5 shrink-0 items-center justify-center">
      {icon}
    </span>
  );
}
