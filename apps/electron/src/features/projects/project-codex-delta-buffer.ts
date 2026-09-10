import { useCallback, useEffect, useRef } from "react";

const RENDER_INTERVAL_MS = 24;
const MAX_LAG_MS = 2500;

type PendingCodexDelta = {
  text: string;
  threadId: string;
  turnId: string;
};

export type ProjectCodexDelta = {
  delta: string;
  threadId: string;
  turnId: string;
};

export function useProjectCodexDeltaBuffer({
  activeThreadIdRef,
  render,
}: {
  activeThreadIdRef: { current: string };
  render: (deltas: ProjectCodexDelta[]) => void;
}) {
  const pendingDeltasRef = useRef<PendingCodexDelta[]>([]);
  const renderTimerRef = useRef<number | null>(null);
  const renderStartedAtRef = useRef<number | null>(null);
  const renderRef = useRef(render);

  useEffect(() => {
    renderRef.current = render;
  }, [render]);

  const stopRenderTimer = useCallback(() => {
    if (renderTimerRef.current === null) return;
    window.clearInterval(renderTimerRef.current);
    renderTimerRef.current = null;
    renderStartedAtRef.current = null;
  }, []);

  const renderPendingDeltas = useCallback(() => {
    const now = Date.now();
    const startedAt = renderStartedAtRef.current ?? now;
    renderStartedAtRef.current = now;

    const activeThreadId = activeThreadIdRef.current;
    const activeIndex = pendingDeltasRef.current.findIndex(
      (delta) => delta.threadId === activeThreadId,
    );
    if (activeIndex < 0) {
      pendingDeltasRef.current = [];
      stopRenderTimer();
      return;
    }

    const activeDelta = pendingDeltasRef.current[activeIndex];
    const lagMs = now - startedAt;
    const characterCount =
      lagMs >= MAX_LAG_MS
        ? activeDelta.text.length
        : Math.max(
            1,
            Math.ceil((activeDelta.text.length * lagMs) / MAX_LAG_MS),
          );
    let renderCount = Math.min(characterCount, activeDelta.text.length);
    if (
      renderCount < activeDelta.text.length &&
      isLowSurrogate(activeDelta.text.charCodeAt(renderCount))
    ) {
      renderCount += 1;
    }

    const renderedDelta: ProjectCodexDelta = {
      delta: activeDelta.text.slice(0, renderCount),
      threadId: activeDelta.threadId,
      turnId: activeDelta.turnId,
    };
    const nextText = activeDelta.text.slice(renderCount);
    pendingDeltasRef.current = nextText
      ? [{ ...activeDelta, text: nextText }]
      : [];

    renderRef.current([renderedDelta]);
    if (pendingDeltasRef.current.length === 0) stopRenderTimer();
  }, [activeThreadIdRef, stopRenderTimer]);

  const startRenderTimer = useCallback(() => {
    if (renderTimerRef.current !== null) return;
    renderStartedAtRef.current = Date.now();
    renderTimerRef.current = window.setInterval(
      renderPendingDeltas,
      RENDER_INTERVAL_MS,
    );
  }, [renderPendingDeltas]);

  const enqueue = useCallback(
    ({ delta, threadId, turnId }: ProjectCodexDelta) => {
      const pendingDeltas = pendingDeltasRef.current;
      const lastIndex = pendingDeltas.length - 1;
      const lastDelta = pendingDeltas[lastIndex];
      if (
        lastDelta &&
        lastDelta.threadId === threadId &&
        lastDelta.turnId === turnId
      ) {
        pendingDeltas[lastIndex] = {
          ...lastDelta,
          text: `${lastDelta.text}${delta}`,
        };
      } else {
        pendingDeltas.push({ text: delta, threadId, turnId });
      }
      startRenderTimer();
    },
    [startRenderTimer],
  );

  const flush = useCallback(
    (turnId?: string) => {
      const pendingDeltas = pendingDeltasRef.current;
      pendingDeltasRef.current = [];
      stopRenderTimer();
      if (pendingDeltas.length === 0) return;

      const activeThreadId = activeThreadIdRef.current;
      const renderedDeltas = pendingDeltas
        .filter(
          (delta) =>
            delta.threadId === activeThreadId &&
            (!turnId || delta.turnId === turnId),
        )
        .map(
          ({ text, threadId: deltaThreadId, turnId: deltaTurnId }) => ({
            delta: text,
            threadId: deltaThreadId,
            turnId: deltaTurnId,
          }),
        );
      pendingDeltasRef.current = pendingDeltas.filter(
        (delta) =>
          delta.threadId !== activeThreadId ||
          (turnId !== undefined && delta.turnId !== turnId),
      );
      if (renderedDeltas.length > 0) renderRef.current(renderedDeltas);
    },
    [activeThreadIdRef, stopRenderTimer],
  );

  const dispose = useCallback(() => {
    pendingDeltasRef.current = [];
    stopRenderTimer();
  }, [stopRenderTimer]);

  useEffect(() => dispose, [dispose]);

  return { enqueue, flush };
}

function isLowSurrogate(codeUnit: number) {
  return codeUnit >= 0xdc00 && codeUnit <= 0xdfff;
}
