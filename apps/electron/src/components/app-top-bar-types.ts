import type { ComponentType } from "react";

export type AppTopBarProps = {
  hasMessage: boolean;
  sidebarExpanded?: boolean;
  title?: string;
  onToggleSidebar?: () => void;
  onBack?: () => void;
  onForward?: () => void;
  onNewPage?: () => void;
  onMore?: () => void;
  onModeSwitch?: () => void;
  onTogglePanel?: () => void;
  onToggleTerminal?: () => void;
  showTerminalButton?: boolean;
  onPinConversation?: () => void;
  onRenameConversation?: () => void;
  onArchiveConversation?: () => void;
  onOpenSideChat?: () => void;
  onCopy?: () => void;
  onBranch?: () => void;
  onAddScheduledTask?: () => void;
  onOpenInNewWindow?: () => void;
  onModelSelect?: () => void;
  onOpenLayoutPanel?: () => void;
};

export type MoreMenuItem = {
  id: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  shortcut?: string;
  hasSubmenu?: boolean;
  children?: MoreMenuItem[];
  onClick?: () => void;
  disabled?: boolean;
};
