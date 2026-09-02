import {
  formatModelTriggerLabel,
  getModelLabel,
  providerSupportsServiceTier,
} from "@/features/projects/project-model-options";
import type { ProjectWorkspaceScreenViewProps } from "@/features/projects/project-workspace-screen-view";

type ScreenViewProps = ProjectWorkspaceScreenViewProps;
type MainProps = ScreenViewProps["main"];
type LeftSidebarProps = ScreenViewProps["leftSidebar"];

type ActiveModelRef = {
  modelId: string;
  providerId: string;
};

type ProjectWorkspaceMainPropsInput = Omit<
  MainProps,
  | "modelLabel"
  | "onCancelSubagentRun"
  | "pendingQuestionRequest"
  | "showProjectPicker"
  | "showServiceTier"
> & {
  activeModelRef?: ActiveModelRef;
  activeRunningSubagentRun?: MainProps["activeSubagentRun"];
  cancelActiveSubagentRun: () => Promise<void> | void;
  pendingQuestionRequests: Array<NonNullable<MainProps["pendingQuestionRequest"]>>;
};

type ProjectWorkspaceLeftSidebarPropsInput = Omit<
  LeftSidebarProps,
  "onArchiveConversation"
> & {
  archiveConversation: (sessionId: string) => Promise<void> | void;
};

type ProjectWorkspaceScreenViewPropsInput = Omit<
  ScreenViewProps,
  "leftSidebar" | "main"
> & {
  leftSidebar: ProjectWorkspaceLeftSidebarPropsInput;
  main: ProjectWorkspaceMainPropsInput;
};

function buildProjectWorkspaceLeftSidebarProps({
  archiveConversation,
  ...leftSidebarProps
}: ProjectWorkspaceLeftSidebarPropsInput): LeftSidebarProps {
  return {
    ...leftSidebarProps,
    onArchiveConversation: (sessionId) => void archiveConversation(sessionId),
  };
}

function buildProjectWorkspaceMainProps({
  activeModelRef,
  activeRunningSubagentRun,
  cancelActiveSubagentRun,
  pendingQuestionRequests,
  ...mainProps
}: ProjectWorkspaceMainPropsInput): MainProps {
  return {
    ...mainProps,
    modelLabel:
      formatModelTriggerLabel(
        getModelLabel(mainProps.modelOptions, mainProps.modelId),
      ) || "模型",
    onCancelSubagentRun: activeRunningSubagentRun
      ? () => void cancelActiveSubagentRun()
      : undefined,
    pendingQuestionRequest: pendingQuestionRequests[0],
    showProjectPicker:
      !mainProps.activeSessionId && !mainProps.showConversationLayout,
    showServiceTier: Boolean(
      activeModelRef && providerSupportsServiceTier(activeModelRef.providerId),
    ),
  };
}

export function buildProjectWorkspaceScreenViewProps({
  leftSidebar,
  main,
  ...screenViewProps
}: ProjectWorkspaceScreenViewPropsInput): ScreenViewProps {
  return {
    ...screenViewProps,
    leftSidebar: buildProjectWorkspaceLeftSidebarProps(leftSidebar),
    main: buildProjectWorkspaceMainProps(main),
  };
}
