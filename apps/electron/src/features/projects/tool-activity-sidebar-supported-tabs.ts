import type { ComponentType } from "react";
import { FileText, Terminal } from "lucide-react";

export type SidebarSupportedTab = {
  id: string;
  icon: ComponentType<{ className?: string }>;
  label: string;
  shortcut?: string;
};

export const SUPPORTED_SIDEBAR_TABS: SidebarSupportedTab[] = [
  { id: "command", icon: Terminal, label: "命令输出" },
  { id: "file", icon: FileText, label: "文件改动" },
];
