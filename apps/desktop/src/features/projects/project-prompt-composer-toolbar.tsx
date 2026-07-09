import type { RefObject } from "react";
import { ArrowUp, Mic, Pause, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ModelSettingsMenu } from "@/features/projects/project-model-settings-menu";
import {
  AgentModeMenu,
  PermissionModeMenu,
} from "@/features/projects/project-prompt-mode-menus";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";

type PromptComposerToolbarProps = Pick<
  PromptComposerProps,
  | "agentMode"
  | "agentModes"
  | "allModelOptions"
  | "modelId"
  | "modelLabel"
  | "modelOptions"
  | "onAddAttachments"
  | "onAgentModeSelect"
  | "onModelSelect"
  | "onPermissionModeSelect"
  | "onReasoningEffortSelect"
  | "onServiceTierSelect"
  | "onSubmit"
  | "pending"
  | "permissionMode"
  | "prompt"
  | "reasoningEffort"
  | "serviceTier"
  | "showServiceTier"
> & {
  compact: boolean;
  fileInputRef: RefObject<HTMLInputElement | null>;
  hasAttachments: boolean;
};

export function PromptComposerToolbar({
  agentMode,
  agentModes,
  allModelOptions,
  compact,
  fileInputRef,
  hasAttachments,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onAgentModeSelect,
  onModelSelect,
  onPermissionModeSelect,
  onReasoningEffortSelect,
  onServiceTierSelect,
  onSubmit,
  pending,
  permissionMode,
  prompt,
  reasoningEffort,
  serviceTier,
  showServiceTier,
}: PromptComposerToolbarProps) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-2 sm:gap-2.5">
      <div className="flex h-9 min-w-0 items-center gap-1.5 sm:gap-3">
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
          aria-label="添加文件"
          className="rounded-full"
          onClick={() => fileInputRef.current?.click()}
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
        <AgentModeMenu
          compact={compact}
          mode={agentMode}
          modes={agentModes}
          onModeSelect={onAgentModeSelect}
        />
      </div>

      <div className="flex h-9 min-w-0 items-center gap-1.5 sm:gap-2.5">
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
          serviceTier={serviceTier}
          showServiceTier={showServiceTier}
        />

        <Button
          aria-label="语音输入"
          className="rounded-full"
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <Mic />
        </Button>

        <Button
          aria-label={pending ? "停止" : "发送"}
          className="rounded-full"
          disabled={!pending && !prompt.trim() && !hasAttachments}
          onClick={onSubmit}
          size="icon"
          type="button"
          variant="default"
        >
          {pending ? <Pause /> : <ArrowUp />}
        </Button>
      </div>
    </div>
  );
}
