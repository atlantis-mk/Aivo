import { TerminalPanelContent } from "@/features/projects/terminal/terminal-panel";

export function ProjectWorkspaceBottomPanel({
  canUseTerminalPanel,
  height,
  workspaceRoot,
}: {
  canUseTerminalPanel: boolean;
  height: number;
  workspaceRoot: string;
}) {
  return (
    <TerminalPanelContent
      key={workspaceRoot || "no-workspace"}
      enabled
      height={height}
      terminalEnabled={canUseTerminalPanel}
      workspaceRoot={workspaceRoot}
    />
  );
}
