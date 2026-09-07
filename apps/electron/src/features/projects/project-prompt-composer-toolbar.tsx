import type { RefObject } from "react";
import { ArrowUp, Pause, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ModelSettingsMenu } from "@/features/projects/project-model-settings-menu";
import { PermissionModeMenu } from "@/features/projects/project-prompt-mode-menus";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";

type PromptComposerToolbarProps = Pick<
  PromptComposerProps,
  | "allModelOptions"
  | "modelId"
  | "modelLabel"
  | "modelOptions"
  | "onAddAttachments"
  | "onModelSelect"
  | "onOpenToolActivationDialog"
  | "onPermissionModeSelect"
  | "onReasoningEffortSelect"
  | "onServiceTierSelect"
  | "onSubmit"
  | "pending"
  | "pendingPromptPastes"
  | "permissionMode"
  | "prompt"
  | "projectPath"
  | "reasoningEffort"
  | "serviceTier"
  | "showServiceTier"
> & {
  compact: boolean;
  fileInputRef: RefObject<HTMLInputElement | null>;
  hasAttachments: boolean;
  onSelectLocalResource: () => void;
};

export function PromptComposerToolbar({
  allModelOptions,
  compact,
  fileInputRef,
  hasAttachments,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onModelSelect,
  onPermissionModeSelect,
  onReasoningEffortSelect,
  onSelectLocalResource,
  onServiceTierSelect,
  onSubmit,
  pending,
  pendingPromptPastes,
  permissionMode,
  prompt,
  projectPath,
  reasoningEffort,
  serviceTier,
  showServiceTier,
}: PromptComposerToolbarProps) {
  const hasSubmitInput =
    Boolean(prompt.trim()) ||
    hasAttachments ||
    pendingPromptPastes.length > 0;
  const isStopAction = pending && !hasSubmitInput;
  return (
    <div className="flex min-h-9 min-w-0 flex-1 items-center justify-between gap-2 sm:gap-2.5">
      <div className="flex min-w-0 items-center gap-0.5 sm:gap-1">
        <input
          className="hidden"
          multiple
          onChange={(event) => {
            onAddAttachments(event.currentTarget.files);
            event.currentTarget.value = "";
          }}
          ref={fileInputRef}
          type="file"
        />
        <Button
          aria-label="选择文件或文件夹"
          className="rounded-full"
          onClick={onSelectLocalResource}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <Plus />
        </Button>

        <PermissionModeMenu
          compact={compact}
          mode={permissionMode}
          onModeSelect={onPermissionModeSelect}
        />

      </div>

      <div className="flex min-w-0 items-center gap-0.5 sm:gap-1">
        <ModelSettingsMenu
          allModelOptions={allModelOptions}
          compact={compact}
          modelId={modelId}
          modelLabel={modelLabel}
          modelOptions={modelOptions}
          onModelSelect={onModelSelect}
          onReasoningEffortSelect={onReasoningEffortSelect}
          reasoningEffort={reasoningEffort}
          onServiceTierSelect={onServiceTierSelect}
          projectPath={projectPath}
          serviceTier={serviceTier}
          showServiceTier={showServiceTier}
        />

        <Button
          aria-label={isStopAction ? "停止" : "发送"}
          className="rounded-full"
          disabled={!pending && !hasSubmitInput}
          onClick={() => onSubmit()}
          size="icon-lg"
          type="button"
          variant="default"
        >
          {isStopAction ? <Pause /> : <ArrowUp />}
        </Button>
      </div>
    </div>
  );
}
