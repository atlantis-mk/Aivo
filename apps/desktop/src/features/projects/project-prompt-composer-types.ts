import type { RefObject } from "react";

import type { ComposerAttachment, ComposerAttachmentInput } from "@/features/projects/project-composer-attachments";
import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";
import type { ModelOption } from "@/features/projects/project-model-options";
import type { ModelInfo } from "@/lib/provider-catalog";
import type {
  AgentModeDefinition,
  AgentModeId,
  PermissionMode,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

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
  onCompactContext: () => Promise<void>;
  onPromptMentionRemove: (reference: PromptMentionReference) => void;
  onPromptMentionSelect: (reference: PromptMentionReference) => void;
  onOpenToolActivationDialog: () => void;
  onPermissionModeSelect: (mode: PermissionMode) => void;
  onPromptChange: (prompt: string) => void;
  onProjectAdd: (rootPath?: string) => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onRemoveAttachment: (id: string) => void;
  onServiceTierSelect: (serviceTier: string) => void;
  onSubmit: () => void;
  pending: boolean;
  permissionMode: PermissionMode;
  prompt: string;
  promptResourceReferences: PromptMentionReference[];
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
  attachments: ComposerAttachment[];
  reasoningEffort: string;
  runtimeStatsLine: string;
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

export type AutoTextareaHeightRef = RefObject<HTMLTextAreaElement | null>;
