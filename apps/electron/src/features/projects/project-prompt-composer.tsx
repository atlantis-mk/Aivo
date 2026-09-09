import { useCallback, useLayoutEffect, useRef, useState } from "react";
import { CornerDownRight, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { ComposerAttachmentList } from "@/features/projects/project-prompt-attachments";
import { PromptContextBar } from "@/features/projects/project-prompt-context-bar";
import { PromptComposerTextarea } from "@/features/projects/project-prompt-composer-textarea";
import { parsePromptSubmission } from "@/features/projects/project-prompt-editor-model";
import { PromptComposerToolbar } from "@/features/projects/project-prompt-composer-toolbar";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";
import { cn } from "@/lib/utils";
import {
  selectComposerFileOrDirectory,
} from "@/services/aivo/project-service";

export function PromptComposer({
  agentMode,
  agentModes,
  allModelOptions,
  modelId,
  modelLabel,
  modelOptions,
  onAddAttachments,
  onAgentModeSelect,
  onExtraHeightChange,
  onHeightChange,
  onModelSelect,
  onPromptMentionRemove,
  onPromptMentionSelect,
  onPromptPaste,
  onPromptPasteRemove,
  onOpenToolActivationDialog,
  onPermissionModeSelect,
  onPromptChange,
  onProjectAdd,
  onProjectClear,
  onProjectSelect,
  onReasoningEffortSelect,
  onRemoveAttachment,
  onRemoveQueuedPrompt,
  onServiceTierSelect,
  onSteerQueuedPrompt,
  onSubmit,
  pending,
  hasPausedTurn,
  pendingPromptPastes,
  permissionMode,
  prompt,
  queuedPrompts,
  promptResourceReferences,
  project,
  projectPath,
  projects,
  attachments,
  reasoningEffort,
  serviceTier,
  showProjectPicker,
  showServiceTier,
}: PromptComposerProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const composerCardRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [compactToolbar, setCompactToolbar] = useState(false);
  const selectLocalResource = useCallback(async () => {
    if (!window.aivoDesktop?.composer) {
      fileInputRef.current?.click();
      return;
    }
    const selection = await selectComposerFileOrDirectory();
    if (!selection) return;
    if (selection.kind === "directory") {
      onAddAttachments(selection);
      return;
    }
    onAddAttachments(selection);
  }, [onAddAttachments, onProjectAdd]);
  useLayoutEffect(() => {
    const rootElement = rootRef.current;
    const cardElement = composerCardRef.current;
    if (!rootElement || !cardElement) return;
    const updateHeight = () => {
      const rootHeight = Math.ceil(rootElement.getBoundingClientRect().height);
      const cardHeight = Math.ceil(cardElement.getBoundingClientRect().height);
      const cardWidth = Math.ceil(cardElement.getBoundingClientRect().width);
      onHeightChange(cardHeight);
      onExtraHeightChange(Math.max(0, rootHeight - cardHeight));
      setCompactToolbar(cardWidth < 640);
    };
    updateHeight();
    const resizeObserver = new ResizeObserver(updateHeight);
    resizeObserver.observe(rootElement);
    resizeObserver.observe(cardElement);
    return () => resizeObserver.disconnect();
  }, [onExtraHeightChange, onHeightChange, showProjectPicker]);

  return (
    <div
      className="flex min-w-0 flex-col bg-background"
      data-testid="prompt-composer"
      ref={rootRef}
    >
      {showProjectPicker ? (
        <PromptContextBar
          agentMode={agentMode}
          agentModes={agentModes}
          onAgentModeSelect={onAgentModeSelect}
          onProjectAdd={onProjectAdd}
          onProjectClear={onProjectClear}
          onProjectSelect={onProjectSelect}
          project={project}
          projectPath={projectPath}
          projects={projects}
        />
      ) : null}
      {queuedPrompts.length > 0 ? (
        <div className="mb-2 flex flex-col gap-2">
          {queuedPrompts.map((queuedPrompt) => (
            <Card
              className="min-w-0 gap-0 rounded-xl py-0 shadow-lg shadow-foreground/5"
              key={queuedPrompt.id}
            >
              <CardContent className="flex min-w-0 items-center gap-3 px-4 py-3">
                <CornerDownRight className="size-4 shrink-0 text-muted-foreground" />
                <p className="min-w-0 flex-1 truncate text-sm font-medium">
                  {parsePromptSubmission(queuedPrompt.text, queuedPrompt.references ?? []).text || "新消息"}
                </p>
                <Button
                  aria-label="调整方向"
                  className="h-7 shrink-0 gap-1 rounded-full px-2 text-xs text-muted-foreground hover:bg-transparent hover:text-foreground"
                  onClick={() => onSteerQueuedPrompt(queuedPrompt.id)}
                  type="button"
                  variant="ghost"
                >
                  <CornerDownRight className="size-3.5" />
                  调整方向
                </Button>
                <Button
                  aria-label="删除排队消息"
                  onClick={() => onRemoveQueuedPrompt(queuedPrompt.id)}
                  size="icon-sm"
                  type="button"
                  variant="ghost"
                >
                  <Trash2 />
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}
      <Card
        className={cn(
          "aivo-prompt-composer relative z-10 min-w-0 gap-0 overflow-visible rounded-xl py-0 shadow-[0_2px_14px_rgb(0_0_0_/_0.06)]",
          showProjectPicker && "-mt-4",
        )}
        ref={composerCardRef}
      >
        <CardContent className="px-5 pb-1 pt-3.5">
          <ComposerAttachmentList
            attachments={attachments}
            onRemoveAttachment={onRemoveAttachment}
          />
          <PromptComposerTextarea
            onAddAttachments={onAddAttachments}
            onPromptChange={onPromptChange}
            onPromptMentionRemove={onPromptMentionRemove}
            onPromptMentionSelect={onPromptMentionSelect}
            onPromptPaste={onPromptPaste}
            onPromptPasteRemove={onPromptPasteRemove}
            onSelectLocalResource={selectLocalResource}
            onSubmit={onSubmit}
            pendingPromptPastes={pendingPromptPastes}
            prompt={prompt}
            promptResourceReferences={promptResourceReferences}
            projectPath={projectPath}
            projects={projects}
            textareaRef={textareaRef}
          />
        </CardContent>
        <CardFooter className="min-w-0 px-3 pb-2.5 pt-1">
          <PromptComposerToolbar
            allModelOptions={allModelOptions}
            compact={compactToolbar}
            fileInputRef={fileInputRef}
            hasAttachments={attachments.length > 0}
            hasPausedTurn={hasPausedTurn}
            modelId={modelId}
            modelLabel={modelLabel}
            modelOptions={modelOptions}
            onAddAttachments={onAddAttachments}
            onModelSelect={onModelSelect}
            onOpenToolActivationDialog={onOpenToolActivationDialog}
            onPermissionModeSelect={onPermissionModeSelect}
            onReasoningEffortSelect={onReasoningEffortSelect}
            onSelectLocalResource={selectLocalResource}
            onServiceTierSelect={onServiceTierSelect}
            onSubmit={onSubmit}
            pending={pending}
            pendingPromptPastes={pendingPromptPastes}
            permissionMode={permissionMode}
            prompt={prompt}
            projectPath={projectPath}
            reasoningEffort={reasoningEffort}
            serviceTier={serviceTier}
            showServiceTier={showServiceTier}
          />
        </CardFooter>
      </Card>
    </div>
  );
}
