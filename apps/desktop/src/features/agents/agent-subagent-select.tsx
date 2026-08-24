import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { AgentModeDefinition } from "@/services/aivo";

export function AgentSubagentSelect({
  candidates,
  describedBy,
  disabled,
  onChange,
  value,
}: {
  candidates: AgentModeDefinition[];
  describedBy?: string;
  disabled?: boolean;
  onChange: (value: string[]) => void;
  value: string[];
}) {
  const selected = candidates.filter((candidate) => value.includes(candidate.id));
  const summary = selected.length
    ? selected.length === 1
      ? selected[0].displayName
      : `已选择 ${selected.length} 个子 Agent`
    : "选择子 Agent";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-describedby={describedBy}
          className="w-full justify-between font-normal"
          disabled={disabled || candidates.length === 0}
          type="button"
          variant="outline"
        >
          <span className="truncate">{candidates.length ? summary : "暂无可关联的子 Agent"}</span>
          <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="max-h-64">
        {candidates.map((candidate) => {
          const checked = value.includes(candidate.id);
          return (
            <DropdownMenuCheckboxItem
              checked={checked}
              disabled={!checked && value.length >= 16}
              key={candidate.id}
              onCheckedChange={(nextChecked) =>
                onChange(
                  nextChecked
                    ? checked
                      ? value
                      : [...value, candidate.id]
                    : value.filter((id) => id !== candidate.id),
                )
              }
              onSelect={(event) => event.preventDefault()}
            >
              <span className="min-w-0 flex-1">
                <span className="flex items-center gap-2">
                  <span className="truncate font-medium">{candidate.displayName}</span>
                  <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
                    {candidate.id}
                  </span>
                </span>
                {candidate.description ? (
                  <span className="mt-0.5 line-clamp-1 block text-muted-foreground">
                    {candidate.description}
                  </span>
                ) : null}
              </span>
            </DropdownMenuCheckboxItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
