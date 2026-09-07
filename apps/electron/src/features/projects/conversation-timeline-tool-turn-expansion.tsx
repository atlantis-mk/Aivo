import {
  createContext,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

type ToolTurnExpansionContextValue = {
  isStatusExpanded: (turnId: string) => boolean | undefined;
  isToolGroupExpanded: (turnId: string, groupId: string) => boolean | undefined;
  toggleStatus: (turnId: string, currentExpanded: boolean) => void;
  toggleToolGroup: (
    turnId: string,
    groupId: string,
    currentExpanded: boolean,
  ) => void;
};

const ToolTurnExpansionContext =
  createContext<ToolTurnExpansionContextValue | null>(null);

export function ToolTurnExpansionProvider({
  children,
}: {
  children: ReactNode;
}) {
  const [statusExpansion, setStatusExpansion] = useState<Map<string, boolean>>(
    () => new Map(),
  );
  const [toolGroupExpansion, setToolGroupExpansion] = useState<
    Map<string, boolean>
  >(() => new Map());
  const toggleToolGroup = (
    turnId: string,
    groupId: string,
    currentExpanded: boolean,
  ) => {
    setToolGroupExpansion((current) => {
      const next = new Map(current);
      next.set(`${turnId}:${groupId}`, !currentExpanded);
      return next;
    });
  };
  const toggleStatus = (turnId: string, currentExpanded: boolean) => {
    setStatusExpansion((current) => {
      const next = new Map(current);
      next.set(turnId, !currentExpanded);
      return next;
    });
  };
  const value = useMemo<ToolTurnExpansionContextValue>(
    () => ({
      isStatusExpanded: (turnId) => statusExpansion.get(turnId),
      isToolGroupExpanded: (turnId, groupId) =>
        toolGroupExpansion.get(`${turnId}:${groupId}`),
      toggleStatus,
      toggleToolGroup,
    }),
    [statusExpansion, toolGroupExpansion],
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
    expanded: context?.isStatusExpanded(turnId),
    toggle: (currentExpanded: boolean) =>
      context?.toggleStatus(turnId, currentExpanded),
  };
}

export function useToolGroupExpansion(turnId: string, groupId: string) {
  const context = useContext(ToolTurnExpansionContext);
  return {
    expanded: context?.isToolGroupExpanded(turnId, groupId),
    toggle: (currentExpanded: boolean) =>
      context?.toggleToolGroup(turnId, groupId, currentExpanded),
  };
}
