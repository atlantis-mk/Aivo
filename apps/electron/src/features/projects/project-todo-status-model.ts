import type { TodoItem } from "@/services/aivo";

export function isTodoDone(status: string) {
  return status === "done" || status === "completed";
}

export function isTodoPlanComplete(todoItems: TodoItem[]) {
  return (
    todoItems.length > 0 && todoItems.every((todo) => isTodoDone(todo.status))
  );
}

export function getTodoPlanKey(todoItems: TodoItem[]) {
  return todoItems
    .map((todo) =>
      [todo.id, todo.title, todo.status, todo.timeUpdated].join("\u0001"),
    )
    .join("\u0002");
}
