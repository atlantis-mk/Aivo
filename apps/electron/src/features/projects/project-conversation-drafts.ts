import {
  useCallback,
  type Dispatch,
  type SetStateAction,
} from "react";
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

const CONVERSATION_DRAFTS_STORAGE_KEY = "aivo:conversation-drafts:v1";

type ConversationDrafts = Record<string, string>;
type ConversationDraftsState = {
  drafts: ConversationDrafts;
  setPrompt: (sessionId: string, action: SetStateAction<string>) => void;
};

const useConversationDraftsStore = create<ConversationDraftsState>()(
  persist(
    (set) => ({
      drafts: {},
      setPrompt: (sessionId, action) =>
        set((state) => {
          const currentPrompt = state.drafts[sessionId] ?? "";
          const nextPrompt =
            typeof action === "function" ? action(currentPrompt) : action;
          if (currentPrompt === nextPrompt) return state;
          return {
            drafts: { ...state.drafts, [sessionId]: nextPrompt },
          };
        }),
    }),
    {
      name: CONVERSATION_DRAFTS_STORAGE_KEY,
      storage: createJSONStorage(() => window.localStorage),
      version: 1,
    },
  ),
);

export function useProjectConversationDrafts({
  activeSessionId,
}: {
  activeSessionId: string;
}): {
  prompt: string;
  setPrompt: Dispatch<SetStateAction<string>>;
} {
  const prompt = useConversationDraftsStore(
    (state) => state.drafts[activeSessionId] ?? "",
  );

  const setPrompt: Dispatch<SetStateAction<string>> = useCallback(
    (action) => {
      useConversationDraftsStore
        .getState()
        .setPrompt(activeSessionId, action);
    },
    [activeSessionId],
  );

  return { prompt, setPrompt };
}
