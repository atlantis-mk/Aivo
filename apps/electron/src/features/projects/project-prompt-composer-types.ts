import type { RefObject } from "react";

import type {
  ComposerAttachment,
  ComposerAttachmentInput,
} from "@/features/projects/project-composer-attachments";
import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";
import type {
  PendingPromptPaste,
  PromptPasteInput,
  PromptPasteResult,
} from "@/features/projects/project-prompt-paste";
import type { ModelOption } from "@/features/projects/project-model-options";
import type { ModelInfo } from "@/lib/provider-catalog";
import type { PermissionMode } from "@/codex-app-server";
import type { AgentModeDefinition, AgentModeId } from "@/services/aivo";
import type { domain } from "@/types/codex-domain";

export type QueuedPrompt = {
  id: string;
  text: string;
  references?: PromptMentionReference[];
};

export type PromptComposerProps = {
  agentMode: AgentModeId;
  agentModes: AgentModeDefinition[];
  allModelOptions: ModelOption[];
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onAddAttachments: (files: ComposerAttachmentInput) => void;
  onAgentModeSelect: (mode: AgentModeId) => void;
  onExtraHeightChange: (height: number) => void;
  onHeightChange: (height: number) => void;
  onModelSelect: (option: ModelOption) => void;
  onPromptMentionRemove: (reference: PromptMentionReference) => void;
  onPromptMentionSelect: (reference: PromptMentionReference) => void;
  onPromptPaste: (
    pastedText: string,
    sourcePath?: string,
  ) => PromptPasteResult | null;
  onPromptPasteRemove: (id: string) => void;
  onOpenToolActivationDialog: () => void;
  onPermissionModeSelect: (mode: PermissionMode) => void;
  onPromptChange: (prompt: string) => void;
  onProjectAdd: (rootPath?: string) => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onRemoveAttachment: (id: string) => void;
  onRemoveQueuedPrompt: (id: string) => void;
  onServiceTierSelect: (serviceTier: string) => void;
  onSteerQueuedPrompt: (id: string) => void;
  onSubmit: (promptOverride?: string) => void;
  pending: boolean;
  hasPausedTurn?: boolean;
  pendingPromptPastes: PendingPromptPaste[];
  permissionMode: PermissionMode;
  prompt: string;
  queuedPrompts: QueuedPrompt[];
  promptResourceReferences: PromptMentionReference[];
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
  attachments: ComposerAttachment[];
  reasoningEffort: string;
  serviceTier: string;
  showProjectPicker: boolean;
  showServiceTier: boolean;
};

export type ComposerAttachmentListProps = {
  attachments: ComposerAttachment[];
  onRemoveAttachment: (id: string) => void;
};

export type ProjectPickerProps = {
  onAddProject: () => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
};

export type AutoTextareaHeightRef = RefObject<HTMLDivElement | null>;
