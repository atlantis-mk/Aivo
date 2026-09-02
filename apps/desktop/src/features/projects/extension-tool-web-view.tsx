import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
  extensionToolViewContext,
  extensionToolViewIdentity,
  type ExtensionToolViewContext,
  type ExtensionToolViewRef,
} from "@/features/projects/extension-tool-view-model";
import type { domain } from "../../../bridge/go/models";

export function ExtensionToolWebView({
  fallback,
  onRequestClose,
  toolCall,
  view,
}: {
  fallback: ReactNode;
  onRequestClose: () => void;
  toolCall: domain.ToolCall;
  view: ExtensionToolViewRef;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const onRequestCloseRef = useRef(onRequestClose);
  const context = extensionToolViewContext(toolCall);
  const contextRef = useRef(context);
  const identity = extensionToolViewIdentity(view);
  const mountedRef = useRef<{
    context: ExtensionToolViewContext;
    contextSequence: number;
    identity: string;
    mountId: string;
    requestId: string;
  } | null>(null);
  const [state, setState] = useState<"mounting" | "ready" | "error">(
    "mounting",
  );
  contextRef.current = context;

  useEffect(() => {
    onRequestCloseRef.current = onRequestClose;
  }, [onRequestClose]);

  useEffect(() => {
    let disposed = false;
    let mountId = "";
    const requestId = crypto.randomUUID();
    let animationFrame = 0;
    let readyFrame = 0;
    let resizeObserver: ResizeObserver | null = null;
    const closeSubscription = window.aivo.onEmbeddedExtensionViewClosed(
      (event) => {
        if (!mountId || event.mountId !== mountId || event.reason === "host") {
          return;
        }
        if (mountedRef.current?.mountId === event.mountId) {
          mountedRef.current = null;
        }
        mountId = "";
        if (event.reason === "guest") onRequestCloseRef.current();
        if (
          event.reason === "load-failed" ||
          event.reason === "render-process-gone" ||
          event.reason === "unresponsive"
        ) {
          setState("error");
        }
      },
    );

    const updateBounds = () => {
      if (!mountId || !hostRef.current) return;
      void window.aivo.updateEmbeddedExtensionViewBounds({
        mountId,
        bounds: elementBounds(hostRef.current),
      });
    };
    const followTransition = (until: number) => {
      updateBounds();
      if (!disposed && performance.now() < until) {
        animationFrame = requestAnimationFrame(() => followTransition(until));
      }
    };
    const mountWhenReady = async () => {
      const host = hostRef.current;
      if (!host) return;
      const bounds = elementBounds(host);
      if (bounds.width <= 1 || bounds.height <= 1) {
        readyFrame = requestAnimationFrame(() => void mountWhenReady());
        return;
      }
      try {
        const initialContext = contextRef.current;
        const mounted = await window.aivo.mountEmbeddedExtensionView({
          requestId,
          extensionId: view.extensionId,
          viewId: view.viewId,
          surface: view.surface,
          bounds,
          context: initialContext,
        });
        if (disposed) {
          await window.aivo.closeEmbeddedExtensionView({
            mountId: mounted.mountId,
            requestId,
          });
          return;
        }
        mountId = mounted.mountId;
        mountedRef.current = {
          context: initialContext,
          contextSequence: 0,
          identity,
          mountId,
          requestId,
        };
        setState("ready");
        resizeObserver = new ResizeObserver(updateBounds);
        resizeObserver.observe(host);
        window.addEventListener("resize", updateBounds);
        followTransition(performance.now() + 400);
        const latestContext = contextRef.current;
        if (!sameContext(initialContext, latestContext)) {
          const mountedView = mountedRef.current;
          mountedView.context = latestContext;
          const contextSequence = ++mountedView.contextSequence;
          void window.aivo
            .updateEmbeddedExtensionViewContext({
              mountId,
              context: latestContext,
            })
            .then((updated) => {
              if (!updated.updated) {
                throw new Error("extension view mount is stale");
              }
            })
            .catch(() => {
              if (
                mountedRef.current !== mountedView ||
                mountedView.contextSequence !== contextSequence
              ) {
                return;
              }
              mountedRef.current = null;
              setState("error");
              void window.aivo.closeEmbeddedExtensionView({
                mountId: mountedView.mountId,
                requestId: mountedView.requestId,
              });
            });
        }
      } catch {
        if (!disposed) {
          if (mountId) {
            if (mountedRef.current?.mountId === mountId) {
              mountedRef.current = null;
            }
            await window.aivo
              .closeEmbeddedExtensionView({ mountId, requestId })
              .catch(() => {});
            mountId = "";
          }
          setState("error");
        }
      }
    };

    setState("mounting");
    readyFrame = requestAnimationFrame(() => void mountWhenReady());
    return () => {
      disposed = true;
      cancelAnimationFrame(animationFrame);
      cancelAnimationFrame(readyFrame);
      resizeObserver?.disconnect();
      window.removeEventListener("resize", updateBounds);
      closeSubscription();
      if (mountedRef.current?.requestId === requestId) {
        mountedRef.current = null;
      }
      void window.aivo.closeEmbeddedExtensionView({ mountId, requestId });
    };
  }, [
    identity,
    view.extensionId,
    view.surface,
    view.viewId,
  ]);

  useEffect(() => {
    const mounted = mountedRef.current;
    if (
      !mounted ||
      mounted.identity !== identity ||
      sameContext(mounted.context, context)
    ) {
      return;
    }
    mounted.context = context;
    const contextSequence = ++mounted.contextSequence;
    void window.aivo
      .updateEmbeddedExtensionViewContext({
        mountId: mounted.mountId,
        context,
      })
      .then((updated) => {
        if (!updated.updated) throw new Error("extension view mount is stale");
      })
      .catch(() => {
        if (
          mountedRef.current !== mounted ||
          mounted.contextSequence !== contextSequence
        ) {
          return;
        }
        mountedRef.current = null;
        setState("error");
        void window.aivo.closeEmbeddedExtensionView({
          mountId: mounted.mountId,
          requestId: mounted.requestId,
        });
      });
  }, [context, identity]);

  return (
    <div
      aria-label={view.title || `${toolCall.name} 扩展页面`}
      className="relative h-full min-h-0 overflow-hidden rounded-b-lg bg-background"
      ref={hostRef}
    >
      {state === "mounting" ? (
        <div className="flex h-full flex-col gap-3 p-3">
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="min-h-0 flex-1" />
        </div>
      ) : null}
      {state === "error" ? (
        <div className="flex h-full min-h-0 flex-col gap-3 overflow-auto p-3">
          <Alert variant="destructive">
            <AlertTitle>扩展页面暂时不可用</AlertTitle>
            <AlertDescription>
              已保留安全的原生工具详情，你仍可查看本次调用记录。
            </AlertDescription>
          </Alert>
          {fallback}
        </div>
      ) : null}
    </div>
  );
}

function sameContext(
  left: ExtensionToolViewContext,
  right: ExtensionToolViewContext,
) {
  return (
    left.operationId === right.operationId &&
    left.sessionId === right.sessionId &&
    left.turnId === right.turnId &&
    left.toolName === right.toolName
  );
}

function elementBounds(element: HTMLElement) {
  const rect = element.getBoundingClientRect();
  return {
    x: Math.max(0, Math.round(rect.left)),
    y: Math.max(0, Math.round(rect.top)),
    width: Math.max(1, Math.round(rect.width)),
    height: Math.max(1, Math.round(rect.height)),
  };
}
