import type {
  DragEventHandler,
  RefObject,
} from "react";

import type { ConversationTurn } from "@/features/projects/conversation-timeline-model";
import type { ComposerAttachment, ComposerAttachmentInput } from "@/features/projects/project-composer-attachments";
import type { ModelOption } from "@/features/projects/project-model-options";
import type { PromptMentionReference } from "@/features/projects/project-prompt-mention-model";
import type { ProjectWorkspacePage } from "@/features/projects/project-workspace-derived-state";
import type { ConversationTimelineHandlerRefs } from "@/features/projects/project-workspace-state-model";
import type { ModelInfo } from "@/lib/provider-catalog";
import type {
  AgentModeDefinition,
  AgentModeId,
  AgentRun,
  PermissionMode,
  PermissionRequest,
  QuestionRequest,
  TodoItem,
} from "@/services/aivo";
import type { domain } from "../../../bridge/go/models";

export type ProjectWorkspaceMainContentProps = {
  activeProjectPage: ProjectWorkspacePage;
  activeSessionId: string;
  activeSubagentRun?: AgentRun;
  agentMode: AgentModeId;
  agentModes: AgentModeDefinition[];
  agentRuns: AgentRun[];
  allModelOptions: ModelOption[];
  attachments: ComposerAttachment[];
  composerBottom: string;
  composerBottomSm: string;
  composerFrameRef: RefObject<HTMLDivElement | null>;
  composerHeight: number;
  contentRef: RefObject<HTMLDivElement | null>;
  emptyComposerTop: string;
  hasPendingInteractionRequest: boolean;
  hasPendingPermissionRequest: boolean;
  hasPendingQuestionRequest: boolean;
  hasPendingTurn: boolean;
  hasTurns: boolean;
  isComposerDropActive: boolean;
  isRevealingHistoryConversation: boolean;
  isSubagentSession: boolean;
  isVisibleTodoPlanComplete: boolean;
  messagesScrollRootRef: RefObject<HTMLDivElement | null>;
  modelId: string;
  modelLabel: string;
  modelOptions: ModelInfo[];
  onAddAttachments: (files: ComposerAttachmentInput) => void;
  onAgentModeSelect: (mode: AgentModeId) => void;
  onBackToParentSession: () => void;
  onCancelSubagentRun?: () => void;
  onDragEnter: DragEventHandler<HTMLDivElement>;
  onDragLeave: DragEventHandler<HTMLDivElement>;
  onDragOver: DragEventHandler<HTMLDivElement>;
  onDrop: DragEventHandler<HTMLDivElement>;
  onExtraHeightChange: (height: number) => void;
  onHeightChange: (height: number) => void;
  onHideCompletedTodoPlan: () => void;
  onModelSelect: (option: ModelOption) => void;
  onOpenToolActivationDialog: () => void;
  onPermissionModeSelect: (mode: PermissionMode) => void;
  onProjectAdd: (rootPath?: string) => void;
  onProjectClear: () => void;
  onProjectSelect: (project: domain.AssistantProject) => void;
  onPromptChange: (prompt: string) => void;
  onPromptMentionRemove: (reference: PromptMentionReference) => void;
  onPromptMentionSelect: (reference: PromptMentionReference) => void;
  onReasoningEffortSelect: (reasoningEffort: string) => void;
  onRemoveAttachment: (id: string) => void;
  onScrollToBottom: () => void;
  onServiceTierSelect: (serviceTier: string) => void;
  onSubmit: () => void;
  pendingPermissionRequests: PermissionRequest[];
  pendingQuestionRequest?: QuestionRequest;
  permissionMode: PermissionMode;
  project: domain.AssistantProject | null;
  projectPath: string;
  projects: domain.AssistantProject[];
  prompt: string;
  promptResourceReferences: PromptMentionReference[];
  reasoningEffort: string;
  serviceTier: string;
  shouldShowTodoFloatingStatus: boolean;
  showConversationLayout: boolean;
  showProjectPicker: boolean;
  showScrollToBottomButton: boolean;
  showServiceTier: boolean;
  todoItems: TodoItem[];
  turns: ConversationTurn[];
  viewportHandlers: ConversationTimelineHandlerRefs;
  workspaceRoot: string;
};
