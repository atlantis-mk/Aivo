import {
  useTerminalPanelController,
  type TerminalPanelControllerProps,
} from "@/features/projects/terminal/terminal-panel-controller";
import { TerminalPanelView } from "@/features/projects/terminal/terminal-panel-view";

type TerminalPanelProps = TerminalPanelControllerProps;

export function TerminalPanelContent(props: TerminalPanelProps) {
  const { canOpenPanel, viewProps } = useTerminalPanelController(props);

  if (!canOpenPanel) return null;

  return <TerminalPanelView {...viewProps} />;
}
