import {
  createContext,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type ToolTurnExpansionContextValue = {
  isExpanded: (turnId: string) => boolean;
  toggle: (turnId: string) => void;
};

const ToolTurnExpansionContext =
  createContext<ToolTurnExpansionContextValue | null>(null);

export function ToolTurnExpansionProvider({ children }: { children: ReactNode }) {
  const [expandedTurnIds, setExpandedTurnIds] = useState<Set<string>>(
    () => new Set(),
  );
  const value = useMemo<ToolTurnExpansionContextValue>(
    () => ({
      isExpanded: (turnId) => expandedTurnIds.has(turnId),
      toggle: (turnId) => {
        setExpandedTurnIds((current) => {
          const next = new Set(current);
          if (next.has(turnId)) {
            next.delete(turnId);
          } else {
            next.add(turnId);
          }
          return next;
        });
      },
    }),
    [expandedTurnIds],
  );

  return (
    <ToolTurnExpansionContext.Provider value={value}>
      {children}
    </ToolTurnExpansionContext.Provider>
  );
}

export function useToolTurnExpansion(turnId: string) {
  const context = useContext(ToolTurnExpansionContext);
  return {
    expanded: context?.isExpanded(turnId) ?? true,
    toggle: () => context?.toggle(turnId),
  };
}
