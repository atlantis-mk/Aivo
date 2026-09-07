import { Check, Plus, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  reasoningEffortOptionsForModel,
  serviceTierOptionsForModel,
} from "@/features/projects/project-model-settings-menu-model";
import {
  getModelLabel,
  modelOptionKey,
  normalizeReasoningEffort,
  normalizeServiceTier,
  type ModelOption,
} from "@/features/projects/project-model-options";
import type { ModelInfo } from "@/lib/provider-catalog";

type GroupedModelOptions = {
  providerId: string;
  providerName: string;
  models: ModelOption[];
};

export function ReasoningEffortMenuItems({
  model,
  onReasoningEffortSelect,
  reasoningEffort,
}: {
  model?: ModelInfo;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  reasoningEffort: string;
}) {
  return (
    <>
      <DropdownMenuLabel>推理</DropdownMenuLabel>
      {reasoningEffortOptionsForModel(model).map((level) => (
        <DropdownMenuItem
          key={level.value}
          onSelect={(event: Event) => {
            event.preventDefault();
            onReasoningEffortSelect(level.value);
          }}
        >
          <span>{level.label}</span>
          {level.value === normalizeReasoningEffort(reasoningEffort) && (
            <Check className="ml-auto" />
          )}
        </DropdownMenuItem>
      ))}
      <DropdownMenuSeparator />
    </>
  );
}

export function ModelPickerSubmenu({
  activeModelKey,
  groupedModels,
  modelId,
  modelLabel,
  modelOptions,
  onAddProvider,
  onModelSelect,
  query,
  setQuery,
}: {
  activeModelKey: string;
  groupedModels: GroupedModelOptions[];
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onAddProvider: () => void;
  onModelSelect: (option: ModelOption) => void;
  query: string;
  setQuery: (query: string) => void;
}) {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <span>{getModelLabel(modelOptions, modelId) || modelLabel}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuSubContent>
        <div className="flex items-center gap-1 px-1 py-1">
          <div className="relative min-w-0 flex-1">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              onChange={(event) => setQuery(event.target.value)}
              onKeyDown={(event) => event.stopPropagation()}
              placeholder="搜索模型"
              value={query}
            />
          </div>
          <Button
            aria-label="添加提供商"
            className="shrink-0"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onAddProvider();
            }}
            onMouseDown={(event) => event.preventDefault()}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <Plus />
          </Button>
        </div>
        <ScrollArea>
          {groupedModels.map((group) => (
            <div key={group.providerId} className="py-1">
              <DropdownMenuLabel>{group.providerName}</DropdownMenuLabel>
              {group.models.map((model) => (
                <DropdownMenuItem
                  key={modelOptionKey(model.providerId, model.id)}
                  onSelect={() => onModelSelect(model)}
                >
                  <span className="min-w-0 truncate">
                    {model.name || model.id}
                  </span>
                  {modelOptionKey(model.providerId, model.id) ===
                  activeModelKey ? (
                    <Check className="ml-auto" />
                  ) : null}
                </DropdownMenuItem>
              ))}
            </div>
          ))}
          {groupedModels.length === 0 ? (
            <div className="px-2 py-6 text-center text-sm text-muted-foreground">
              没有匹配的模型
            </div>
          ) : null}
        </ScrollArea>
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );
}

export function ServiceTierSubmenu({
  model,
  onServiceTierSelect,
  serviceTier,
}: {
  model?: ModelInfo;
  onServiceTierSelect: (serviceTier: string) => void;
  serviceTier: string;
}) {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>速度</DropdownMenuSubTrigger>
      <DropdownMenuSubContent>
        {serviceTierOptionsForModel(model).map((tier) => (
          <DropdownMenuItem
            key={tier.value}
            onSelect={(event: Event) => {
              event.preventDefault();
              onServiceTierSelect(tier.value);
            }}
          >
            <span>{tier.label}</span>
            {tier.value === normalizeServiceTier(serviceTier) && (
              <Check className="ml-auto" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );
}
