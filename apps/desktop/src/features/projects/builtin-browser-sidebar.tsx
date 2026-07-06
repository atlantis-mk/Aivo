import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import {
  ArrowLeft,
  ArrowRight,
  Globe,
  Loader2,
  RefreshCw,
  Search,
  X,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { cn } from "@/lib/utils";

const EMPTY_BROWSER_STATE: AivoBrowserState = {
  url: "",
  title: "",
  favicon: "",
  loading: false,
  canGoBack: false,
  canGoForward: false,
  error: "",
};

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
    setState((currentState) => ({
      ...currentState,
      error: "",
      favicon: "",
      loading: true,
      url: nextUrl,
    }));
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
    const syncBoundsWhenReady = async () => {
      for (let attempt = 0; attempt < 8; attempt += 1) {
        if (disposed) return false;
        if (await updateBrowserBounds().catch(() => false)) {
          return true;
        }
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
      }
      return false;
    };
    const resizeObserver = new ResizeObserver(syncBounds);
    resizeObserver.observe(host);

    void syncBoundsWhenReady()
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
      for (let attempt = 0; attempt < 8; attempt += 1) {
        if (disposed) return;
        if (await updateBrowserBounds().catch(() => false)) {
          if (disposed) return;
          const nextState = await browser.setVisible(browserTabId, true);
          if (disposed) return;
          if (nextState) {
            applyBrowserState(nextState);
          }
          onReady?.();
          return;
        }
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
      }
      if (!disposed) onReady?.();
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
      for (let attempt = 0; attempt < 8; attempt += 1) {
        if (disposed) return;
        if (await updateBrowserBounds().catch(() => false)) {
          if (!disposed) onReady?.();
          return;
        }
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
      }
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

  async function runBrowserAction(action: "back" | "forward" | "reload" | "stop") {
    try {
      const browser = window.aivo?.browser;
      if (!browser || !webviewReady) return;
      let nextState: AivoBrowserState | undefined;
      if (action === "back") {
        nextState = await browser.goBack(browserTabId);
      } else if (action === "forward") {
        nextState = await browser.goForward(browserTabId);
      } else if (action === "stop") {
        nextState = await browser.stop(browserTabId);
      } else {
        nextState = await browser.reload(browserTabId);
      }
      if (nextState) {
        applyBrowserState(nextState);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "浏览器操作失败");
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-background" data-app-no-drag>
      <div className="flex shrink-0 flex-col gap-2 border-b border-border/70 p-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <Button
            aria-label="后退"
            disabled={!webviewReady || !state.canGoBack}
            onClick={() => void runBrowserAction("back")}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ArrowLeft />
          </Button>
          <Button
            aria-label="前进"
            disabled={!webviewReady || !state.canGoForward}
            onClick={() => void runBrowserAction("forward")}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <ArrowRight />
          </Button>
          <Button
            aria-label={state.loading ? "停止加载" : "刷新"}
            disabled={!webviewReady || !state.url}
            onClick={() => void runBrowserAction(state.loading ? "stop" : "reload")}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            {state.loading ? (
              <X />
            ) : (
              <RefreshCw className={cn(state.loading && "animate-spin")} />
            )}
          </Button>
          <form className="min-w-0 flex-1" onSubmit={navigateToAddress}>
            <InputGroup>
              <InputGroupAddon align="inline-start">
                {state.loading ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Globe />
                )}
              </InputGroupAddon>
              <InputGroupInput
                aria-label="浏览器地址"
                autoCapitalize="none"
                autoCorrect="off"
                onChange={(event) => setAddress(event.target.value)}
                placeholder="输入网址"
                ref={inputRef}
                spellCheck={false}
                value={address}
              />
              <InputGroupAddon align="inline-end">
                <InputGroupButton aria-label="打开地址" type="submit">
                  <Search />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
          </form>
          <Button
            aria-label="关闭内置浏览器"
            onClick={onClose}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <X />
          </Button>
        </div>
        {state.error ? (
          <div className="min-w-0 px-1 text-xs leading-5">
            <span className="block truncate text-destructive">
              {state.error}
            </span>
          </div>
        ) : null}
      </div>
      <div className="relative min-h-0 flex-1 overflow-hidden">
        <div
          className="absolute inset-0 h-full w-full"
          ref={browserHostRef}
        />
        {!state.url ? (
          <div className="flex h-full min-h-0 items-center justify-center px-5 py-8">
            <div className="flex max-w-72 flex-col items-center gap-3 text-center">
              <div className="flex size-10 items-center justify-center rounded-md bg-muted text-muted-foreground">
                <Globe />
              </div>
              <div className="text-sm font-medium text-foreground">
                内置浏览器
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function normalizeBrowserURL(input: string) {
  const raw = input.trim();
  if (!raw) {
    throw new Error("Browser address is required");
  }
  const withScheme = /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(raw)
    ? raw
    : `https://${raw}`;
  const parsed = new URL(withScheme);
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("Only http(s) addresses can be opened in the built-in browser");
  }
  return parsed.toString();
}
