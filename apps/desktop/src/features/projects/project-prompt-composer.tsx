import { useLayoutEffect, useRef, useState } from "react";

import { ComposerAttachmentList } from "@/features/projects/project-prompt-attachments";
import { useAutoTextareaHeight } from "@/features/projects/project-prompt-composer-height";
import { PromptComposerTextarea } from "@/features/projects/project-prompt-composer-textarea";
import { PromptComposerToolbar } from "@/features/projects/project-prompt-composer-toolbar";
import type { PromptComposerProps } from "@/features/projects/project-prompt-composer-types";

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
  const minTextareaHeight = 0;
  const maxTextareaHeight = 300;
  const [compactToolbar, setCompactToolbar] = useState(false);
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
      setCompactToolbar(cardWidth < 560);
    };
    updateHeight();
    const resizeObserver = new ResizeObserver(updateHeight);
    resizeObserver.observe(rootElement);
    resizeObserver.observe(cardElement);
    return () => resizeObserver.disconnect();
  }, [onExtraHeightChange, onHeightChange, showProjectPicker]);

  return (
    <div className="flex min-w-0 flex-col" ref={rootRef}>
      <div
        className="relative z-10 flex min-w-0 flex-col overflow-hidden rounded-lg border border-border bg-card px-4 py-3 shadow-sm shadow-foreground/5"
        ref={composerCardRef}
      >
        <ComposerAttachmentList
          attachments={attachments}
          onRemoveAttachment={onRemoveAttachment}
        />
        <PromptComposerTextarea
          onAddAttachments={onAddAttachments}
          onPromptChange={onPromptChange}
          onSubmit={onSubmit}
          prompt={prompt}
          textareaHeights={textareaHeights}
          textareaRef={textareaRef}
        />
        <PromptComposerToolbar
          agentMode={agentMode}
          agentModes={agentModes}
          allModelOptions={allModelOptions}
          compact={compactToolbar}
          fileInputRef={fileInputRef}
          hasAttachments={attachments.length > 0}
          modelId={modelId}
          modelLabel={modelLabel}
          modelOptions={modelOptions}
          onAddAttachments={onAddAttachments}
          onAgentModeSelect={onAgentModeSelect}
          onModelSelect={onModelSelect}
          onPermissionModeSelect={onPermissionModeSelect}
          onProjectAdd={onProjectAdd}
          onProjectClear={onProjectClear}
          onProjectSelect={onProjectSelect}
          onReasoningEffortSelect={onReasoningEffortSelect}
          onServiceTierSelect={onServiceTierSelect}
          onSubmit={onSubmit}
          pending={pending}
          permissionMode={permissionMode}
          prompt={prompt}
          project={project}
          projectPath={projectPath}
          projects={projects}
          reasoningEffort={reasoningEffort}
          serviceTier={serviceTier}
          showProjectPicker={showProjectPicker}
          showServiceTier={showServiceTier}
        />
      </div>
    </div>
  );
}
