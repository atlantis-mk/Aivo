import {
  annotateToolActivityTabsWithFileStates,
  type ToolActivityTab,
} from "@/features/projects/tool-activity-model";
import { getSessionTurnDiff } from "@/services/aivo";

export async function annotateToolActivityTabsWithTurnDiff(
  sessionId: string,
  tabs: ToolActivityTab[],
) {
  const turnIds = Array.from(
    new Set(
      tabs.flatMap((tab) =>
        tab.kind === "file" && tab.turnId ? [tab.turnId] : [],
      ),
    ),
  );
  if (turnIds.length === 0) return tabs;
  const states = (
    await Promise.all(
      turnIds.map(async (turnId) => {
        try {
          const diff = await getSessionTurnDiff({ sessionId, turnId });
          return diff.files;
        } catch {
          return [];
        }
      }),
    )
  ).flat();
  return annotateToolActivityTabsWithFileStates(tabs, states);
}
