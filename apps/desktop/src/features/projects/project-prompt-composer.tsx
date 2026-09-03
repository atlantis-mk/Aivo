import { useCallback, useLayoutEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { routeComposerLocalSelections } from "@/features/projects/project-composer-attachments";
import { ComposerAttachmentList } from "@/features/projects/project-prompt-attachments";
import { PromptContextBar } from "@/features/projects/project-prompt-context-bar";
import { useAutoTextareaHeight } from "@/features/projects/project-prompt-composer-height";
import { PromptComposerTextarea } from "@/features/projects/project-prompt-composer-textarea";
import { PromptComposerToolbar } from "@/features/projects/project-prompt-composer-toolbar";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";
import { hasAppBridge } from "@/lib/app-config";
import { cn } from "@/lib/utils";
import { selectComposerFileOrDirectory } from "@/services/aivo/project-service";

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
  onOpenToolActivationDialog,
  onPermissionModeSelect,
  onPromptChange,
  onProjectAdd,
  onProjectClear,
  onProjectSelect,
  onReasoningEffortSelect,
  onRemoveAttachment,
  onServiceTierSelect,
  onSubmit,
  pending,
  permissionMode,
  prompt,
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
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const minTextareaHeight = 32;
  const maxTextareaHeight = 300;
  const [compactToolbar, setCompactToolbar] = useState(false);
  const selectLocalResource = useCallback(async () => {
    if (!hasAppBridge()) {
      fileInputRef.current?.click();
      return;
    }
    try {
      const selection = await selectComposerFileOrDirectory();
      if (!selection) return;
      routeComposerLocalSelections([selection], {
        onDirectory: (path) => onProjectAdd(path),
        onFile: (file) => onAddAttachments(file),
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "选择文件或文件夹失败",
      );
    }
  }, [onAddAttachments, onProjectAdd]);
  const textareaHeights = useAutoTextareaHeight(
    prompt,
    minTextareaHeight,
    maxTextareaHeight,
    textareaRef,
  );
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
      <Card
        className={cn(
          "relative z-10 min-w-0 gap-0 overflow-visible rounded-3xl py-0 shadow-lg shadow-foreground/5",
          showProjectPicker && "-mt-4",
        )}
        ref={composerCardRef}
      >
        <CardContent className="px-5 pb-1 pt-4">
          <ComposerAttachmentList
            attachments={attachments}
            onRemoveAttachment={onRemoveAttachment}
          />
          <PromptComposerTextarea
            onAddAttachments={onAddAttachments}
            onPromptChange={onPromptChange}
            onPromptMentionRemove={onPromptMentionRemove}
            onPromptMentionSelect={onPromptMentionSelect}
            onSelectLocalResource={selectLocalResource}
            onSubmit={onSubmit}
            prompt={prompt}
            promptResourceReferences={promptResourceReferences}
            projectPath={projectPath}
            projects={projects}
            textareaHeights={textareaHeights}
            textareaRef={textareaRef}
          />
        </CardContent>
        <CardFooter className="min-w-0 px-3 pb-3 pt-1">
          <PromptComposerToolbar
            allModelOptions={allModelOptions}
            compact={compactToolbar}
            fileInputRef={fileInputRef}
            hasAttachments={attachments.length > 0}
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
