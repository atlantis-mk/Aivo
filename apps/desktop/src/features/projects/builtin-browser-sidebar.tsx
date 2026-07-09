import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { toast } from "sonner";

import {
  BuiltinBrowserEmptyState,
  BuiltinBrowserToolbar,
} from "@/features/projects/builtin-browser-controls";
import {
  EMPTY_BROWSER_STATE,
  navigatingBrowserState,
  normalizeBrowserURL,
  runBuiltinBrowserAction,
  type BuiltinBrowserAction,
  waitForBuiltinBrowserBounds,
} from "@/features/projects/builtin-browser-model";

type BuiltinBrowserSidebarProps = {
  browserTabId: string;
  initialUrl?: string;
  onClose: () => void;
  onReady?: () => void;
  onStateChange?: (state: AivoBrowserState) => void;
  readyToken?: number;
  visible?: boolean;
};

export function BuiltinBrowserSidebar({
  browserTabId,
  initialUrl,
  onClose,
  onReady,
  onStateChange,
  readyToken,
  visible = true,
}: BuiltinBrowserSidebarProps) {
  const [address, setAddress] = useState("");
  const [state, setState] = useState<AivoBrowserState>(EMPTY_BROWSER_STATE);
  const [webviewReady, setWebviewReady] = useState(false);
  const browserHostRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const applyBrowserState = useCallback((nextState: AivoBrowserState) => {
    setState(() => {
      if (document.activeElement !== inputRef.current) {
        setAddress(nextState.url || "");
      }
      return nextState;
    });
  }, []);

  const updateBrowserBounds = useCallback(async () => {
    const browser = window.aivo?.browser;
    const host = browserHostRef.current;
    if (!browser || !host) return false;
    if (!visible) {
      await browser.setVisible(browserTabId, false);
      return false;
    }
    const rect = host.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) {
      await browser.setVisible(browserTabId, false);
      return false;
    }
    await browser.setBounds(browserTabId, {
      x: rect.left,
      y: rect.top,
      width: rect.width,
      height: rect.height,
    });
    return true;
  }, [browserTabId, visible]);

  const navigateCurrentTab = useCallback(async (target: string) => {
    const nextUrl = normalizeBrowserURL(target);
    setAddress(nextUrl);
    setState((currentState) => navigatingBrowserState(currentState, nextUrl));
    const nextState = await window.aivo?.browser?.navigate(browserTabId, nextUrl);
    if (nextState) {
      applyBrowserState(nextState);
    }
  }, [applyBrowserState, browserTabId]);

  useEffect(() => {
    setWebviewReady(false);
    setAddress("");
    setState(EMPTY_BROWSER_STATE);
  }, [browserTabId]);

  useEffect(() => {
    onStateChange?.(state);
  }, [onStateChange, state]);

  useLayoutEffect(() => {
    const browser = window.aivo?.browser;
    const host = browserHostRef.current;
    if (!browser || !host) return;

    let disposed = false;
    const syncBounds = () => {
      if (disposed) return;
      void updateBrowserBounds().catch(() => undefined);
    };
    const resizeObserver = new ResizeObserver(syncBounds);
    resizeObserver.observe(host);

    void waitForBuiltinBrowserBounds({
      isDisposed: () => disposed,
      updateBrowserBounds,
    })
      .then(async (boundsReady) => ({
        boundsReady,
        nextState: await browser.getState(browserTabId),
      }))
      .then(({ boundsReady, nextState }) => {
        if (disposed) return;
        applyBrowserState(nextState);
        setWebviewReady(true);
        if (!visible || !boundsReady) return undefined;
        return browser.setVisible(browserTabId, true);
      })
      .then((nextState) => {
        if (!disposed && nextState) {
          applyBrowserState(nextState);
          void updateBrowserBounds()
            .then(() => onReady?.())
            .catch(() => onReady?.());
        }
      })
      .catch((error) => {
        if (!disposed) {
          setState((currentState) => ({
            ...currentState,
            error: error instanceof Error ? error.message : "浏览器初始化失败",
          }));
        }
      })
      .finally(syncBounds);

    window.addEventListener("resize", syncBounds);
    requestAnimationFrame(syncBounds);

    return () => {
      disposed = true;
      resizeObserver.disconnect();
      window.removeEventListener("resize", syncBounds);
      void browser.setVisible(browserTabId, false).catch(() => undefined);
    };
  }, [applyBrowserState, browserTabId, onReady, updateBrowserBounds, visible]);

  useEffect(() => {
    const browser = window.aivo?.browser;
    if (!browser) return;

    let disposed = false;
    if (!visible) {
      void browser.setVisible(browserTabId, false).catch(() => undefined);
      return;
    }

    const showWhenBoundsAreStable = async () => {
      const boundsReady = await waitForBuiltinBrowserBounds({
        isDisposed: () => disposed,
        updateBrowserBounds,
      });
      if (disposed) return;
      if (boundsReady) {
        const nextState = await browser.setVisible(browserTabId, true);
        if (disposed) return;
        if (nextState) {
          applyBrowserState(nextState);
        }
        onReady?.();
        return;
      }
      onReady?.();
    };

    void showWhenBoundsAreStable().catch(() => {
      if (!disposed) onReady?.();
    });

    return () => {
      disposed = true;
    };
  }, [applyBrowserState, browserTabId, onReady, updateBrowserBounds, visible]);

  useEffect(() => {
    if (readyToken === undefined) return;
    if (!visible) return;
    let disposed = false;
    const syncReadyBounds = async () => {
      await waitForBuiltinBrowserBounds({
        isDisposed: () => disposed,
        updateBrowserBounds,
      });
      if (!disposed) onReady?.();
    };
    void syncReadyBounds();
    return () => {
      disposed = true;
    };
  }, [onReady, readyToken, updateBrowserBounds, visible]);

  useEffect(() => {
    const unsubscribe = window.aivo?.browser?.onStateChange?.((nextState) => {
      applyBrowserState(nextState);
      setWebviewReady(true);
    }, browserTabId);
    return () => unsubscribe?.();
  }, [applyBrowserState, browserTabId]);

  useEffect(() => {
    if (!initialUrl) return;
    try {
      void navigateCurrentTab(initialUrl);
    } catch {
      // Ignore invalid popup URLs. The embedded browser only supports http(s).
    }
  }, [browserTabId, initialUrl, navigateCurrentTab]);

  async function navigateToAddress(event?: FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    const target = address.trim();
    if (!target) {
      inputRef.current?.focus();
      return;
    }
    try {
      await navigateCurrentTab(target);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "无法打开地址");
    }
  }

  async function runBrowserAction(action: BuiltinBrowserAction) {
    try {
      const browser = window.aivo?.browser;
      if (!browser || !webviewReady) return;
      const nextState = await runBuiltinBrowserAction({
        action,
        browser,
        browserTabId,
      });
      if (nextState) {
        applyBrowserState(nextState);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "浏览器操作失败");
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-background" data-app-no-drag>
      <BuiltinBrowserToolbar
        address={address}
        inputRef={inputRef}
        onAddressChange={setAddress}
        onClose={onClose}
        onRunAction={(action) => void runBrowserAction(action)}
        onSubmitAddress={navigateToAddress}
        state={state}
        webviewReady={webviewReady}
      />
      <div className="relative min-h-0 flex-1 overflow-hidden">
        <div
          className="absolute inset-0 h-full w-full"
          ref={browserHostRef}
        />
        {!state.url ? <BuiltinBrowserEmptyState /> : null}
      </div>
    </div>
  );
}
